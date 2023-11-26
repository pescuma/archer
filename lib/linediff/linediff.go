package linediff

import (
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type Diff struct {
	Type  Operation
	Lines int
}

type Operation int8

const (
	DiffDelete Operation = Operation(diffmatchpatch.DiffDelete)
	DiffInsert Operation = Operation(diffmatchpatch.DiffInsert)
	DiffEqual  Operation = Operation(diffmatchpatch.DiffEqual)
)

func DoWithTimeout(src, dst string, timeout time.Duration) []Diff {
	dmp := diffmatchpatch.New()
	dmp.DiffTimeout = timeout
	wSrc, wDst := textsToLineIndexes(src, dst)
	dmpd := dmp.DiffMainRunes(wSrc, wDst, false)
	diffs := lineIndexesToDiff(dmpd)
	return diffs
}

func lineIndexesToDiff(diffs []diffmatchpatch.Diff) []Diff {
	hydrated := make([]Diff, 0, len(diffs))
	for _, aDiff := range diffs {
		hydrated = append(hydrated, Diff{
			Type:  Operation(aDiff.Type),
			Lines: len(aDiff.Text),
		})
	}
	return hydrated
}

func textsToLineIndexes(text1, text2 string) ([]rune, []rune) {
	lineToIndex := make(map[string]int)
	indexes1 := textToLineIndexes(text1, lineToIndex)
	indexes2 := textToLineIndexes(text2, lineToIndex)
	return indexes1, indexes2
}

func textToLineIndexes(text string, lineToIndex map[string]int) []rune {
	lines := strings.SplitAfter(text, "\n")

	result := make([]rune, len(lines))
	for i, line := range lines {
		lineValue, ok := lineToIndex[line]

		if !ok {
			lineValue = len(lineToIndex)
			lineToIndex[line] = lineValue
		}

		result[i] = rune(lineValue)
	}
	return result
}
