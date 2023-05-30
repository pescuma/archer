package model

type Changes struct {
	In6Months     int
	Total         int
	ModifiedLines int
	AddedLines    int
	DeletedLines  int
}

func NewChanges() *Changes {
	result := &Changes{}
	result.Clear()
	return result
}

func (m *Changes) Clear() {
	m.In6Months = -1
	m.Total = -1
	m.ModifiedLines = -1
	m.AddedLines = -1
	m.DeletedLines = -1
}
