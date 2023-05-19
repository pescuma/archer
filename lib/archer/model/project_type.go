package model

type ProjectType int

const (
	ExternalDependencyType ProjectType = iota
	CodeType
	DatabaseType
)

func (t ProjectType) String() string {
	switch t {
	case ExternalDependencyType:
		return "external dependency"
	case CodeType:
		return "code"
	case DatabaseType:
		return "db"
	default:
		return "<unknown>"
	}
}
