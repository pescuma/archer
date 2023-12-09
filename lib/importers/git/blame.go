package git

import (
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pescuma/archer/lib/linediff"
	"strings"
)

func Blame(filename string, commitHash plumbing.Hash, cache BlameCache) ([]plumbing.Hash, error) {
	file, err := cache.GetFile(filename, commitHash)
	if err != nil {
		return nil, err
	}

	contents, err := file.Contents()
	if err != nil {
		return nil, err
	}

	result := make([]plumbing.Hash, len(strings.Split(contents, "\n")))

	queue := make(chan *blameItem, 100)
	queue <- newBlameItemComplete(commitHash, filename, file.Hash, contents)
	for i := range queue {
		err = computeBlame(result, i, queue, cache)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func computeBlame(result []plumbing.Hash, i *blameItem, queue chan *blameItem, cache BlameCache) error {
	commit, err := cache.GetCommit(i.CommitHash)
	if err != nil {
		return err
	}

	commit, err = ignoreUninterestingCommits(commit, i.File, cache)
	if err != nil {
		return err
	}

	fileInfo, changedFile := commit.Changes[i.File]

	if !changedFile {
		for hash, parent := range commit.Parents {
			parentFile, renamed := parent.Renames[i.File]
			if !renamed {
				parentFile = i.File
			}

			// Only follow the side where the file came from
			if parent.FileHashes[parentFile] == i.FileHash {
				queue <- newBlameItem(hash, parentFile, i.FileHash, i.Contents, i.Ranges)
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

func computeParentsInfo(i *blameItem, commit *BlameCommitCache, cache BlameCache) ([]parentItem, error) {
	result := make([]parentItem, 0, len(commit.Parents))
	for hash, parent := range commit.Parents {
		fileName, renamed := parent.Renames[i.File]
		if !renamed {
			fileName = i.File
		}

		fileHash := parent.FileHashes[fileName]

		file, err := cache.GetFile(fileName, fileHash)
		if err != nil {
			return nil, err
		}

		contents, err := file.Contents()
		if err != nil {
			return nil, err
		}

		diff := linediff.Do(contents, i.Contents)

		result = append(result, parentItem{
			Hash:     hash,
			File:     fileName,
			FileHash: fileHash,
			Contents: contents,
			Diff:     diff,
		})
	}
	return result, nil
}

func ignoreUninterestingCommits(commit *BlameCommitCache, file string, cache BlameCache) (*BlameCommitCache, error) {
	for len(commit.Parents) == 1 && !commit.Touched(file) {
		parentHash, parentInfo := first(commit.Parents)
		if _, ok := parentInfo.Renames[file]; ok {
			break
		}

		parent, err := cache.GetCommit(parentHash)
		if err != nil {
			return nil, err
		}

		commit = parent
	}
	return commit, nil
}

func mergeParentChanges(parents []parentItem) []linediff.Diff {
	// TODO
	return parents[0].Diff
}

func computeAffected(ranges []linesRange, changes []linediff.Diff) (changed []linesRange, notChanged []linesRange) {
	// TODO
	return nil, nil
}

func updateRanges(ranges []linesRange, diffs []linediff.Diff) []linesRange {
	// TODO
	return ranges
}

func fill(result []plumbing.Hash, rs []linesRange, hash plumbing.Hash) {
	for _, r := range rs {
		for i := r.OriginalStart; i <= r.OriginalEnd; i++ {
			result[i] = hash
		}
	}
}

func first[K comparable, V any](m map[K]V) (K, V) {
	for k, v := range m {
		return k, v
	}

	panic("empty map")
}

type blameItem struct {
	CommitHash plumbing.Hash
	File       string
	FileHash   plumbing.Hash
	Contents   string
	Ranges     []linesRange
}

type parentItem struct {
	Hash     plumbing.Hash
	File     string
	FileHash plumbing.Hash
	Contents string
	Diff     []linediff.Diff
}

func newBlameItem(commitHash plumbing.Hash, file string, fileHash plumbing.Hash, contents string, ranges []linesRange) *blameItem {
	return &blameItem{
		CommitHash: commitHash,
		File:       file,
		FileHash:   fileHash,
		Contents:   contents,
		Ranges:     ranges,
	}
}

func newBlameItemComplete(commitHash plumbing.Hash, file string, fileHash plumbing.Hash, contents string) *blameItem {
	lines := len(strings.Split(contents, "\n"))
	return &blameItem{
		CommitHash: commitHash,
		File:       file,
		FileHash:   fileHash,
		Contents:   contents,
		Ranges: []linesRange{
			{
				Start:         0,
				End:           lines - 1,
				OriginalStart: 0,
				OriginalEnd:   lines - 1,
			},
		},
	}
}

type linesRange struct {
	Start         int
	End           int
	OriginalStart int
	OriginalEnd   int
}

func allLinesRanges(lines int) []linesRange {
	return []linesRange{
		linesRange{
			Start:         0,
			End:           lines - 1,
			OriginalStart: 0,
			OriginalEnd:   lines - 1,
		},
	}
}
