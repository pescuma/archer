package model

type Changes struct {
	In6Months     int
	Total         int
	LinesModified int
	LinesAdded    int
	LinesDeleted  int
}

func NewChanges() *Changes {
	return &Changes{
		In6Months:     -1,
		Total:         -1,
		LinesModified: -1,
		LinesAdded:    -1,
		LinesDeleted:  -1,
	}
}

func (m *Changes) Clear() {
	m.In6Months = 0
	m.Total = 0
	m.LinesModified = 0
	m.LinesAdded = 0
	m.LinesDeleted = 0
}
