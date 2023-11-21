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

func (s *Changes) IsEmpty() bool {
	return s.In6Months == -1 && s.Total == -1 && s.LinesModified == -1 && s.LinesAdded == -1 && s.LinesDeleted == -1
}

func (m *Changes) Clear() {
	m.In6Months = 0
	m.Total = 0
	m.LinesModified = 0
	m.LinesAdded = 0
	m.LinesDeleted = 0
}

func (m *Changes) Reset() {
	m.In6Months = -1
	m.Total = -1
	m.LinesModified = -1
	m.LinesAdded = -1
	m.LinesDeleted = -1
}
