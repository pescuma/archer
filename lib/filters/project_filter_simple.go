package filters

import (
	"github.com/pescuma/archer/lib/model"
)

type simpleProjectFilter struct {
	filterProject    func(*model.Project) bool
	filterDependency func(*model.ProjectDependency) bool
}

func (s *simpleProjectFilter) FilterProject(proj *model.Project) bool {
	return s.filterProject == nil || s.filterProject(proj)
}

func (s *simpleProjectFilter) FilterDependency(dep *model.ProjectDependency) bool {
	return s.filterDependency == nil || s.filterDependency(dep)
}

type simpleProjectFilterWithUsage struct {
	filter ProjectFilter
	usage  UsageType
}

func (s *simpleProjectFilterWithUsage) FilterProject(proj *model.Project) UsageType {
	if s.filter.FilterProject(proj) {
		return s.usage
	} else {
		return DontCare
	}
}

func (s *simpleProjectFilterWithUsage) FilterDependency(dep *model.ProjectDependency) UsageType {
	if s.filter.FilterDependency(dep) {
		return s.usage
	} else {
		return DontCare
	}
}

func (s *simpleProjectFilterWithUsage) Decide(u UsageType) bool {
	return u.DecideFor(s.usage)
}

type projectFilterWithUsageGroup struct {
	filters []ProjectFilterWithUsage
}

func (m *projectFilterWithUsageGroup) FilterProject(proj *model.Project) UsageType {
	result := DontCare
	for _, f := range m.filters {
		result = result.Merge(f.FilterProject(proj))
	}
	return result
}

func (m *projectFilterWithUsageGroup) FilterDependency(dep *model.ProjectDependency) UsageType {
	result := DontCare
	for _, f := range m.filters {
		result = result.Merge(f.FilterDependency(dep))
	}
	return result
}

func (m *projectFilterWithUsageGroup) Decide(u UsageType) bool {
	switch u {
	case Include:
		return true
	case Exclude:
		return false
	default:
		result := true
		for _, f := range m.filters {
			result = result && f.Decide(u)
		}
		return result
	}
}
