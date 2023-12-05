package main

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/pescuma/archer/lib/model"
)

type sizes struct {
	elements int
	lines    int
	files    int
	bytes    int
}

func (s *sizes) isEmpty() bool {
	return s.get() == 0
}

func (s *sizes) add(size *model.Size) {
	s.elements++
	s.lines += size.Lines
	s.files += size.Files
	s.bytes += size.Bytes
}

func (s *sizes) get() int {
	switch {
	case s.bytes > 0:
		return s.bytes
	case s.lines > 0:
		return s.lines
	case s.files > 0:
		return s.files
	default:
		return 0
	}
}

func (s *sizes) text() string {
	ts := s.prepareText()
	if len(ts) == 0 {
		return ""
	}

	result := strings.Join(ts, ", ")

	if s.elements > 1 {
		result = "∑ " + result
	}

	return result
}

func (s *sizes) html() string {
	ts := s.prepareText()
	if len(ts) == 0 {
		return ""
	}

	if s.elements > 1 {
		for i := range ts {
			ts[i] = "∑ " + ts[i]
		}
	}

	result := strings.Join(ts, "<br/>")

	return result
}

func (s *sizes) prepareText() []string {
	var result []string

	if s.lines > 0 {
		t := humanize.Bytes(uint64(s.lines))
		t = strings.TrimSuffix(t, "B")
		t = strings.TrimSpace(t)
		result = append(result, fmt.Sprintf("%v lines", t))
	}

	if s.files > 0 {
		t := humanize.Bytes(uint64(s.files))
		t = strings.TrimSuffix(t, "B")
		t = strings.TrimSpace(t)
		result = append(result, fmt.Sprintf("%v files", t))
	}

	if s.bytes > 0 {
		result = append(result, fmt.Sprintf("%v", humanize.IBytes(uint64(s.bytes))))
	}

	return result
}
