package orm

import "github.com/pescuma/archer/lib/model"

type sqlChanges struct {
	Semester      *int
	Total         *int
	LinesModified *int
	LinesAdded    *int
	LinesDeleted  *int
}

func newSqlChanges(c *model.Changes) *sqlChanges {
	return &sqlChanges{
		Semester:      encodeMetric(c.In6Months),
		Total:         encodeMetric(c.Total),
		LinesModified: encodeMetric(c.LinesModified),
		LinesAdded:    encodeMetric(c.LinesAdded),
		LinesDeleted:  encodeMetric(c.LinesDeleted),
	}
}

func (s *sqlChanges) ToModel() *model.Changes {
	return &model.Changes{
		In6Months:     decodeMetric(s.Semester),
		Total:         decodeMetric(s.Total),
		LinesModified: decodeMetric(s.LinesModified),
		LinesAdded:    decodeMetric(s.LinesAdded),
		LinesDeleted:  decodeMetric(s.LinesDeleted),
	}
}
