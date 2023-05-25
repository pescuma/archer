package model

type ProjectType int

const (
	Library ProjectType = iota
	CodeType
	DatabaseType
)

func (t ProjectType) String() string {
	switch t {
	case Library:
		return "lib"
	case CodeType:
		return "code"
	case DatabaseType:
		return "db"
	default:
		return "<unknown>"
	}
}
