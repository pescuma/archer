package model

type Changes struct {
	In6Months     int
	Total         int
	ModifiedLines int
	AddedLines    int
	DeletedLines  int
}

func NewChanges() *Changes {
	return &Changes{
		In6Months:     -1,
		Total:         -1,
		ModifiedLines: -1,
		AddedLines:    -1,
		DeletedLines:  -1,
	}
}

func (m *Changes) Clear() {
	m.In6Months = 0
	m.Total = 0
	m.ModifiedLines = 0
	m.AddedLines = 0
	m.DeletedLines = 0
}
