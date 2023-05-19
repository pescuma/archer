package model

type FilterType int

const (
	FilterAll FilterType = iota
	FilterExcludeExternal
)
