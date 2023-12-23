package git

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-set/v2"
	"github.com/oleiade/lane/v2"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"github.com/pescuma/archer/lib/linediff"
)

type blameLine struct {
	CommitHash string
	Text       string
}

func (b *blameLine) String() string {
	return fmt.Sprintf("blameLine[CommitHash=%v Text=%v]", b.CommitHash, b.Text)
}

func Blame(filename string, gitCommit *object.Commit, cache BlameCache) ([]*blameLine, error) {
	gitFile, err := gitCommit.File(filename)
	if err != nil {
		return nil, err
	}

	contents, err := gitFile.Contents()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSuffix(contents, "\n"), "\n")

	result := make([]*blameLine, len(lines))
	for i := range lines {
		result[i] = &blameLine{Text: strings.TrimRight(lines[i], "\r")}
	}

	queue := newBlameQueue(cache)

	err = queue.Push(gitCommit.Hash, filename, gitFile.Hash, contents, newRangesWithLines(len(lines)))
	if err != nil {
		return nil, err
	}

	for {
		i, ok := queue.Pop()
		if !ok {
			break
		}

		err = computeBlame(result, queue, cache, i)
		if err != nil {
			return nil, err
		}
	}

	for i, line := range result {
		if line.CommitHash == "" {
			panic(fmt.Sprintf("commit hash should not be empty on line %v", i))
		}
	}

	return result, nil
}

func computeBlame(result []*blameLine, queue *blameQueue, cache BlameCache, i *blameItem) error {
	commit := i.CommitCache

	fileInfo, changedFile := checkChanged(i, commit)

	if !changedFile {
		for parentHash, parent := range commit.Parents {
			parentFileName := parent.FileName(i.FileName)
			parentFileHash := parent.FileHash(i.FileName, i.FileHash)

			// Only follow the side(s) where the file came from
			if parentFileHash == i.FileHash {
				err := queue.Push(parentHash, parentFileName, i.FileHash, i.FileContents, i.Ranges)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	if fileInfo.Created {
		fill(result, i.Ranges, commit.Hash)
		return nil
	}

	parentsInfo, err := computeParentsInfo(i, commit, cache)
	if err != nil {
		return err
	}

	mergedParentChanges := mergeParentChanges(parentsInfo)

	changed, notChanged := computeAffected(i.Ranges, mergedParentChanges)

	fill(result, changed, commit.Hash)

	for _, parent := range parentsInfo {
		fromParent := newRangesWithCapacity(len(notChanged))
		for _, c := range notChanged {
			if c.source.Contains(parent.Hash) {
				fromParent = fromParent.append(c.linesRange)
			}
		}

		if len(fromParent) > 0 {
			rs := updateRanges(fromParent, parent.Diff)
			err = queue.Push(parent.Hash, parent.FileName, parent.FileHash, parent.FileContents, rs)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkChanged(i *blameItem, commit *BlameCommitCache) (*BlameFileCache, bool) {
	fileInfo, ok := commit.Changes[i.FileName]
	if !ok {
		return nil, false
	}

	if fileInfo.Created {
		return fileInfo, true
	}

	// Consider changes only hash changes and not renames
	changes := 0
	for _, parent := range commit.Parents {
		parentFileHash := parent.FileHash(i.FileName, i.FileHash)
		if parentFileHash != i.FileHash {
			changes++
		}
	}

	return fileInfo, changes == len(commit.Parents)
}

func computeParentsInfo(i *blameItem, commit *BlameCommitCache, cache BlameCache) ([]*parentItem, error) {
	result := make([]*parentItem, 0, len(commit.Parents))
	for hash, parent := range commit.Parents {
		fileName := parent.FileName(i.FileName)
		fileHash := parent.FileHash(i.FileName, i.FileHash)

		if fileHash == plumbing.ZeroHash {
			continue
		}

		if fileHash == i.FileHash {
			panic("should be different")
		}

		file, err := cache.GetFile(fileName, fileHash)
		if err != nil {
			return nil, err
		}

		contents, err := file.Contents()
		if err != nil {
			return nil, err
		}

		diff := linediff.Do(contents, i.FileContents)

		result = append(result, &parentItem{
			Hash:         hash,
			FileName:     fileName,
			FileHash:     fileHash,
			FileContents: contents,
			Diff:         diff,
		})
	}
	return result, nil
}

type mergedDiff struct {
	linediff.Diff
	sources *set.Set[plumbing.Hash]
}

func mergeParentChanges(parents []*parentItem) []*mergedDiff {
	diffs := lo.Map(parents, func(p *parentItem, _ int) []*mergedDiff {
		result := make([]*mergedDiff, 0, len(p.Diff))
		for _, d := range p.Diff {
			if d.Type == linediff.DiffDelete {
				continue
			}

			result = append(result, &mergedDiff{
				Diff:    d,
				sources: set.From[plumbing.Hash]([]plumbing.Hash{p.Hash}),
			})
		}
		return result
	})

	result := make([]*mergedDiff, 0, len(diffs[0]))
	for {
		finished := lo.CountBy(diffs, func(d []*mergedDiff) bool { return len(d) == 0 })
		if finished > 0 {
			if finished < len(diffs) {
				panic("should finish all at same time")
			}
			break
		}

		block := getNextBlock(diffs)

		removeBlocks(diffs, block)

		result = addBlock(result, block)
	}

	return result
}

func getNextBlock(parentDiffs [][]*mergedDiff) *mergedDiff {
	lines := parentDiffs[0][0].Lines
	dt := parentDiffs[0][0].Type
	sources := set.New[plumbing.Hash](len(parentDiffs))

	for _, diffs := range parentDiffs {
		d := diffs[0]

		lines = min(lines, d.Lines)

		if d.Type == linediff.DiffEqual {
			sources.InsertSet(d.sources)
		}

		if d.Type != dt {
			// One is DiffEqual and the other is DiffInsert
			dt = linediff.DiffEqual
		}
	}

	return &mergedDiff{
		Diff: linediff.Diff{
			Type:  dt,
			Lines: lines,
		},
		sources: sources,
	}
}

func removeBlocks(diffs [][]*mergedDiff, block *mergedDiff) {
	for i, d := range diffs {
		if d[0].Lines > block.Lines {
			d[0].Lines = d[0].Lines - block.Lines
		} else {
			diffs[i] = d[1:]
		}
	}
}

func addBlock(result []*mergedDiff, block *mergedDiff) []*mergedDiff {
	if len(result) > 0 {
		last := result[len(result)-1]
		if last.Type == block.Type && last.sources.Equal(block.sources) {
			last.Lines += block.Lines
			return result
		}
	}

	result = append(result, block)
	return result
}

type linesRangeWithSource struct {
	linesRange
	source *set.Set[plumbing.Hash]
}

func computeAffected(ranges linesRanges, changes []*mergedDiff) (changed linesRanges, notChanged []*linesRangeWithSource) {
	// ranges will be changed, so make a copy
	ranges = ranges.clone()

	line := 0
	for _, change := range changes {
		if change.Type == linediff.DiffDelete {
			continue
		}

		end := line + change.Lines - 1
		line = end + 1

		if len(ranges) == 0 {
			break
		}

		if ranges[0].Start > end {
			continue
		}

		for len(ranges) > 0 && ranges[0].Start <= end {
			candidate := ranges[0]
			var result linesRange

			if candidate.End <= end {
				result = candidate
				ranges = ranges[1:]
			} else {
				a, b := candidate.split(end + 1)
				result = a
				ranges[0] = b
			}

			switch change.Type {
			case linediff.DiffEqual:
				notChanged = append(notChanged,
					&linesRangeWithSource{
						linesRange: result,
						source:     change.sources,
					})
			case linediff.DiffInsert:
				changed = changed.append(result)
			default:
				panic("unexpected change type")
			}
		}
	}

	return
}

// This destroys ranges
func updateRanges(ranges linesRanges, diffs []linediff.Diff) linesRanges {
	result := newRangesWithCapacity(len(ranges))

	lineNew := 0
	offset := 0
	for _, diff := range diffs {
		if len(ranges) == 0 {
			break
		}

		switch diff.Type {
		case linediff.DiffEqual:
			endNew := lineNew + diff.Lines - 1
			lineNew = endNew + 1

			for len(ranges) > 0 && ranges[0].Start <= endNew {
				r := ranges[0]
				if r.End <= endNew {
					result = result.append(r.drift(offset))
					ranges = ranges[1:]
				} else {
					a, b := r.split(endNew + 1)
					result = result.append(a.drift(offset))
					ranges[0] = b
				}
			}

		case linediff.DiffInsert:
			endNew := lineNew + diff.Lines - 1
			lineNew = endNew + 1
			offset -= diff.Lines

			if ranges[0].Start <= endNew {
				panic("should not happen")
			}

		case linediff.DiffDelete:
			offset += diff.Lines

		default:
			panic("unexpected change type")
		}
	}

	return result
}

func fill(result []*blameLine, rs linesRanges, hash plumbing.Hash) {
	for _, r := range rs {
		for i := r.Start; i <= r.End; i++ {
			for _, offset := range r.Offsets {
				result[i+offset].CommitHash = hash.String()
			}
		}
	}
}

type blameQueue struct {
	cache BlameCache
	items map[plumbing.Hash]*blameItem
	queue *lane.PriorityQueue[plumbing.Hash, int]
}

func newBlameQueue(cache BlameCache) *blameQueue {
	return &blameQueue{
		cache: cache,
		items: make(map[plumbing.Hash]*blameItem),
		queue: lane.NewMaxPriorityQueue[plumbing.Hash, int](),
	}
}

func (q *blameQueue) Push(commitHash plumbing.Hash,
	fileName string, fileHash plumbing.Hash, fileContents string,
	ranges linesRanges,
) error {
	if item, ok := q.items[commitHash]; ok {
		item.Ranges = item.Ranges.merge(ranges)

	} else {
		commitCache, err := q.cache.GetCommit(commitHash)
		if err != nil {
			return err
		}

		i := newBlameItem(commitHash, commitCache, fileName, fileHash, fileContents, ranges)

		q.items[commitHash] = i
		q.queue.Push(commitHash, i.CommitCache.Order)
	}

	return nil
}

func (q *blameQueue) Pop() (*blameItem, bool) {
	hash, _, ok := q.queue.Pop()
	if !ok {
		return nil, false
	}

	result := q.items[hash]
	delete(q.items, hash)

	return result, true
}

type blameItem struct {
	CommitHash   plumbing.Hash
	CommitCache  *BlameCommitCache
	FileName     string
	FileHash     plumbing.Hash
	FileContents string
	Ranges       linesRanges
}

func (b *blameItem) String() string {
	return fmt.Sprintf("blameItem[CommitHash=%v FileName=%v FileHash=%v Contents=%v Ranges=%v]",
		b.CommitHash, b.FileName, b.FileHash, b.FileContents, b.Ranges)
}

type parentItem struct {
	Hash         plumbing.Hash
	FileName     string
	FileHash     plumbing.Hash
	FileContents string
	Diff         []linediff.Diff
}

func (p *parentItem) String() string {
	return fmt.Sprintf("parentItem[Hash=%v FileName=%v FileHash=%v FileContents=%v Diff=%v]",
		p.Hash, p.FileName, p.FileHash, p.FileContents, len(p.Diff))
}

func newBlameItem(commitHash plumbing.Hash, commitCache *BlameCommitCache,
	fileName string, fileHash plumbing.Hash, fileContents string,
	ranges linesRanges,
) *blameItem {
	return &blameItem{
		CommitHash:   commitHash,
		CommitCache:  commitCache,
		FileName:     fileName,
		FileHash:     fileHash,
		FileContents: fileContents,
		Ranges:       ranges,
	}
}

type linesRanges []linesRange

func newRanges(items ...linesRange) linesRanges {
	return items
}

func newRangesWithCapacity(size int) linesRanges {
	return make([]linesRange, 0, size)
}

func newRangesWithLines(lines int) linesRanges {
	return []linesRange{newRange(0, lines-1, 0)}
}

// This destroys the current range
func (s linesRanges) append(i linesRange) []linesRange {
	r := s

	l := len(r)
	if l > 0 && r[l-1].End == i.Start-1 && r[l-1].Offsets.equals(i.Offsets) {
		r[l-1].End = i.End

	} else {
		r = append(r, i)
	}

	return r
}

// This range is destroyed by the merge
func (s linesRanges) merge(r2 linesRanges) linesRanges {
	r1 := slices.Clone(s)
	r2 = slices.Clone(r2)

	result := newRangesWithCapacity(len(r1) + len(r2))

	for len(r1) > 0 && len(r2) > 0 {
		if r1[0].Start > r2[0].Start {
			r1, r2 = r2, r1
		}

		if r1[0].End < r2[0].Start {
			result = result.append(r1[0])
			r1 = r1[1:]
			continue
		}

		if r1[0].Start < r2[0].Start {
			a, b := r1[0].split(r2[0].Start)
			result = result.append(a)
			r1[0] = b
			continue
		}

		// r1[0].Start == r2[0].Start

		if r1[0].End > r2[0].End {
			r1, r2 = r2, r1
		}

		if r1[0].End < r2[0].End {
			a, b := r2[0].split(r1[0].End + 1)
			result = result.append(newRange(a.Start, a.End, a.Offsets.join(r1[0].Offsets)...))
			r1 = r1[1:]
			r2[0] = b
			continue
		}

		// r1[0].Start == r2[0].Start

		result = result.append(newRange(r1[0].Start, r1[0].End, r1[0].Offsets.join(r2[0].Offsets)...))
		r1 = r1[1:]
		r2 = r2[1:]
	}

	for _, r := range r1 {
		result = result.append(r)
	}
	for _, r := range r2 {
		result = result.append(r)
	}

	return result
}

func (s linesRanges) clone() linesRanges {
	return slices.Clone(s)
}

type linesRange struct {
	Start   int
	End     int
	Offsets offsets
}

func newRange(start, end int, offsets ...int) linesRange {
	sort.Ints(offsets)

	return linesRange{
		Start:   start,
		End:     end,
		Offsets: offsets,
	}
}

func (r *linesRange) drift(offset int) linesRange {
	return newRange(r.Start+offset, r.End+offset, r.Offsets.drift(-offset)...)
}

func (r *linesRange) split(start int) (linesRange, linesRange) {
	return newRange(r.Start, start-1, r.Offsets...),
		newRange(start, r.End, r.Offsets...)
}

func (r *linesRange) String() string {
	return fmt.Sprintf("linesRange[Start=%v End=%v Offset=%v]", r.Start, r.End, r.Offsets)
}

type offsets []int

func (o offsets) drift(diff int) []int {
	result := slices.Clone(o)
	for i := range result {
		result[i] += diff
	}
	return result
}

func (o offsets) equals(o2 offsets) bool {
	if len(o) != len(o2) {
		return false
	}

	for i := range o {
		if o[i] != o2[i] {
			return false
		}
	}

	return true
}

func (o offsets) join(o2 []int) []int {
	result := make([]int, 0, len(o)+len(o2))
	result = append(result, o...)
	result = append(result, o2...)
	result = lo.Uniq(result)
	sort.Ints(result)
	return result
}
