package model

import (
	"strings"

	"github.com/samber/lo"
)

type MonthlyStats struct {
	lines map[string]*MonthlyStatsLine
}

func NewMonthlyStats() *MonthlyStats {
	return &MonthlyStats{
		lines: make(map[string]*MonthlyStatsLine),
	}
}

func (s *MonthlyStats) GetOrCreateLines(month string, repositoryID UUID, authorID UUID, committerID UUID, projectID *UUID) *MonthlyStatsLine {
	key := s.createKey(month, repositoryID, authorID, committerID, projectID)

	line, ok := s.lines[key]
	if !ok {
		line = NewMonthlyStatsLine(month, repositoryID, authorID, committerID, projectID)
		s.lines[key] = line
	}

	return line
}

func (s *MonthlyStats) ListLines() []*MonthlyStatsLine {
	return lo.Values(s.lines)
}

func (s *MonthlyStats) createKey(month string, repositoryID UUID, authorID UUID, committerID UUID, projectID *UUID) string {
	pid := ""
	if projectID != nil {
		pid = string(*projectID)
	}

	return strings.Join([]string{month, string(repositoryID), string(authorID), string(committerID), pid}, "\n")

}
