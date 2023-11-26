package model

type ProjectDirectoryType int

const (
	SourceDir ProjectDirectoryType = iota
	TestsDir
	ConfigDir
)

func (t ProjectDirectoryType) String() string {
	switch t {
	case SourceDir:
		return "source"
	case TestsDir:
		return "tests"
	case ConfigDir:
		return "config"
	default:
		return "<unknown>"
	}
}
