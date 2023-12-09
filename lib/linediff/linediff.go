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

type Options struct {
	ConsiderWhitespace bool
	Timeout            time.Duration
}

func Do(src, dst string) []Diff {
	return DoWithOpts(src, dst, nil)
}

func DoWithOpts(src, dst string, opts *Options) []Diff {
	if opts == nil {
		opts = &Options{}
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	differ := diffmatchpatch.New()
	differ.DiffTimeout = opts.Timeout
	runesSrc, runesDst := textsToLineIndexes(src, dst, !opts.ConsiderWhitespace)
	runesDiff := differ.DiffMainRunes(runesSrc, runesDst, false)
	diffs := lineIndexesToDiff(runesDiff)
	return diffs
}

func lineIndexesToDiff(diffs []diffmatchpatch.Diff) []Diff {
	result := make([]Diff, 0, len(diffs))
	for _, aDiff := range diffs {
		result = append(result, Diff{
			Type:  Operation(aDiff.Type),
			Lines: len(aDiff.Text),
		})
	}
	return result
}

func textsToLineIndexes(text1, text2 string, ignoreWhitespace bool) ([]rune, []rune) {
	lineToIndex := make(map[string]int)
	indexes1 := textToLineIndexes(text1, lineToIndex, ignoreWhitespace)
	indexes2 := textToLineIndexes(text2, lineToIndex, ignoreWhitespace)
	return indexes1, indexes2
}

func textToLineIndexes(text string, lineToIndex map[string]int, ignoreWhitespace bool) []rune {
	lines := strings.SplitAfter(text, "\n")

	result := make([]rune, len(lines))
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		if ignoreWhitespace {
			line = strings.TrimSpace(line)
			line = strings.Join(strings.Fields(line), " ")
		}

		lineValue, ok := lineToIndex[line]

		if !ok {
			lineValue = len(lineToIndex)
			lineToIndex[line] = lineValue
		}

		result[i] = rune(lineValue)
	}
	return result
}
