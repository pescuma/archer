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

func (c *Changes) LinesTotal() int {
	return c.LinesModified + c.LinesAdded + c.LinesDeleted
}

func (c *Changes) IsEmpty() bool {
	return c.In6Months == -1 && c.Total == -1 && c.LinesModified == -1 && c.LinesAdded == -1 && c.LinesDeleted == -1
}

func (c *Changes) Clear() {
	c.In6Months = 0
	c.Total = 0
	c.LinesModified = 0
	c.LinesAdded = 0
	c.LinesDeleted = 0
}

func (c *Changes) Reset() {
	c.In6Months = -1
	c.Total = -1
	c.LinesModified = -1
	c.LinesAdded = -1
	c.LinesDeleted = -1
}
