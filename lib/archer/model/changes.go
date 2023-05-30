package model

type Changes struct {
	In6Months     int
	Total         int
	ModifiedLines int
	AddedLines    int
	DeletedLines  int
}

func NewChanges() *Changes {
	return &Changes{}
}

func (m *Changes) Clear() {
	m.In6Months = 0
	m.Total = 0
	m.ModifiedLines = 0
	m.AddedLines = 0
	m.DeletedLines = 0
}
