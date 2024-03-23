package filters

import "github.com/pescuma/archer/lib/model"

type FileFilter func(*model.File) bool

type FileFilterWithUsage interface {
	Filter(*model.File) UsageType

	// Decide does not return DontCase, so it should decide what to do in this case
	Decide(u UsageType) bool
}
