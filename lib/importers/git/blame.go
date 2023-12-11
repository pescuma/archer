package git

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-set/v2"
	"github.com/samber/lo"

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

	commits := cache.CommitCount()
	seen := set.New[plumbing.Hash](commits)
	queue := make(chan *blameItem, commits)

	queue <- newBlameItemComplete(gitCommit.Hash, filename, gitFile.Hash, contents, len(lines))
	for len(queue) > 0 {
		i := <-queue

		if !seen.Insert(i.CommitHash) {
			continue
		}

		err = computeBlame(result, i, queue, cache)
		if err != nil {
			return nil, err
		}
	}

	for _, line := range result {
		if line.CommitHash == "" {
			panic("commit hash should not be empty")
		}
	}

	return result, nil
}

func computeBlame(result []*blameLine, i *blameItem, queue chan *blameItem, cache BlameCache) error {
	commit, err := cache.GetCommit(i.CommitHash)
	if err != nil {
		return err
	}

	fileInfo, changedFile := checkChanged(i, commit)

	if !changedFile {
		for hash, parent := range commit.Parents {
			parentFileName := parent.FileName(i.FileName)
			parentFileHash := parent.FileHash(i.FileName, i.FileHash)

			// Only follow the side(s) where the file came from
			if parentFileHash == i.FileHash {
				queue <- newBlameItem(hash, parentFileName, i.FileHash, i.Contents, i.Ranges)
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

	if len(notChanged) > 0 {
		for _, parent := range parentsInfo {
			rs := updateRanges(notChanged, parent.Diff)
			queue <- newBlameItem(parent.Hash, parent.File, parent.FileHash, parent.Contents, rs)
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

		diff := linediff.Do(contents, i.Contents)

		result = append(result, &parentItem{
			Hash:     hash,
			File:     fileName,
			FileHash: fileHash,
			Contents: contents,
			Diff:     diff,
		})
	}
	return result, nil
}

func mergeParentChanges(parents []*parentItem) []linediff.Diff {
	if len(parents) == 1 {
		return parents[0].Diff
	}

	diffs := lo.Map(parents, func(p *parentItem, _ int) []linediff.Diff {
		return lo.Filter(p.Diff, func(d linediff.Diff, _ int) bool { return d.Type != linediff.DiffDelete })
	})

	result := make([]linediff.Diff, 0, len(diffs[0]))
	for {
		finished := lo.CountBy(diffs, func(d []linediff.Diff) bool { return len(d) == 0 })
		if finished > 0 {
			if finished < len(diffs) {
				panic("should finish all at same time")
			}
			break
		}

		lines, dt := getNextBlock(diffs)

		removeBlocks(diffs, lines)

		result = addBlock(result, dt, lines)
	}

	return result
}

func getNextBlock(diffs [][]linediff.Diff) (int, linediff.Operation) {
	lines := diffs[0][0].Lines
	dt := diffs[0][0].Type

	for _, d := range diffs[1:] {
		lines = min(lines, d[0].Lines)
		if d[0].Type != dt {
			// One is equal and the other is insert
			dt = linediff.DiffEqual
		}
	}

	return lines, dt
}

func removeBlocks(diffs [][]linediff.Diff, lines int) {
	for i, d := range diffs {
		if d[0].Lines != lines {
			d[0].Lines = d[0].Lines - lines
		} else {
			diffs[i] = d[1:]
		}
	}
}

func addBlock(result []linediff.Diff, dt linediff.Operation, lines int) []linediff.Diff {
	if len(result) > 0 && result[len(result)-1].Type == dt {
		result[len(result)-1].Lines += lines
	} else {
		result = append(result, linediff.Diff{Type: dt, Lines: lines})
	}
	return result
}

func computeAffected(ranges []linesRange, changes []linediff.Diff) (changed []linesRange, notChanged []linesRange) {
	// ranges will be changed, so make a copy
	tmp := make([]linesRange, len(ranges))
	copy(tmp, ranges)
	ranges = tmp

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
				result = newLinesRange(candidate.Start, end, candidate.Offset)
				ranges[0].Start = end + 1
			}

			switch change.Type {
			case linediff.DiffEqual:
				notChanged = appendRange(notChanged, result)
			case linediff.DiffInsert:
				changed = appendRange(changed, result)
			default:
				panic("unexpected change type")
			}
		}
	}

	return
}

func updateRanges(ranges []linesRange, diffs []linediff.Diff) []linesRange {
	// ranges will be changed, so make a copy
	tmp := make([]linesRange, len(ranges))
	copy(tmp, ranges)
	ranges = tmp

	result := make([]linesRange, 0, len(ranges))

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
					result = appendRange(result, newLinesRange(r.Start+offset, r.End+offset, r.Offset-offset))
					ranges = ranges[1:]
				} else {
					result = appendRange(result, newLinesRange(r.Start+offset, endNew+offset, r.Offset-offset))
					ranges[0] = newLinesRange(endNew+1, r.End, r.Offset)
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

func appendRange(list []linesRange, i linesRange) []linesRange {
	l := len(list)
	if l > 0 && list[l-1].End == i.Start-1 && list[l-1].Offset == i.Offset {
		list[l-1].End = i.End
	} else {
		list = append(list, i)
	}

	return list
}

func fill(result []*blameLine, rs []linesRange, hash plumbing.Hash) {
	for _, r := range rs {
		for i := r.Start; i <= r.End; i++ {
			result[i+r.Offset].CommitHash = hash.String()
		}
	}
}

type blameItem struct {
	CommitHash plumbing.Hash
	FileName   string
	FileHash   plumbing.Hash
	Contents   string
	Ranges     []linesRange
}

func (b *blameItem) String() string {
	return fmt.Sprintf("blameItem[CommitHash=%v FileName=%v FileHash=%v Contents=%v Ranges=%v]",
		b.CommitHash, b.FileName, b.FileHash, b.Contents, b.Ranges)
}

type parentItem struct {
	Hash     plumbing.Hash
	File     string
	FileHash plumbing.Hash
	Contents string
	Diff     []linediff.Diff
}

func (p *parentItem) String() string {
	return fmt.Sprintf("parentItem[Hash=%v File=%v FileHash=%v Contents=%v Diff=%v]",
		p.Hash, p.File, p.FileHash, p.Contents, len(p.Diff))
}

func newBlameItem(commitHash plumbing.Hash, file string, fileHash plumbing.Hash, contents string, ranges []linesRange) *blameItem {
	return &blameItem{
		CommitHash: commitHash,
		FileName:   file,
		FileHash:   fileHash,
		Contents:   contents,
		Ranges:     ranges,
	}
}

func newBlameItemComplete(commitHash plumbing.Hash, file string, fileHash plumbing.Hash, contents string, lines int) *blameItem {
	return &blameItem{
		CommitHash: commitHash,
		FileName:   file,
		FileHash:   fileHash,
		Contents:   contents,
		Ranges:     newLinesRangesAll(lines),
	}
}

type linesRange struct {
	Start  int
	End    int
	Offset int
}

func (l *linesRange) String() string {
	return fmt.Sprintf("linesRange[Start=%v End=%v Offset=%v]", l.Start, l.End, l.Offset)
}

func newLinesRange(start, end, offset int) linesRange {
	return linesRange{
		Start:  start,
		End:    end,
		Offset: offset,
	}
}

func newLinesRangesAll(lines int) []linesRange {
	return []linesRange{{
		Start:  0,
		End:    lines - 1,
		Offset: 0,
	}}
}
