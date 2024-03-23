package filters

import (
	"github.com/pescuma/archer/lib/model"
)

type simpleFileFilterWithUsage struct {
	filter FileFilter
	usage  UsageType
}

func (s *simpleFileFilterWithUsage) Filter(file *model.File) UsageType {
	if s.filter(file) {
		return s.usage
	} else {
		return DontCare
	}
}

func (s *simpleFileFilterWithUsage) Decide(u UsageType) bool {
	return u.DecideFor(s.usage)
}

type fileFilterWithUsageGroup struct {
	filters []FileFilterWithUsage
}

func (g *fileFilterWithUsageGroup) Filter(file *model.File) UsageType {
	result := DontCare
	for _, f := range g.filters {
		result = result.Merge(f.Filter(file))
	}
	return result
}

func (g *fileFilterWithUsageGroup) Decide(u UsageType) bool {
	switch u {
	case Include:
		return true
	case Exclude:
		return false
	default:
		result := true
		for _, f := range g.filters {
			result = result && f.Decide(u)
		}
		return result
	}
}
