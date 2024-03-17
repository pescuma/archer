package filters

import (
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

type simpleProjectFilter struct {
	filterProject    func(proj *model.Project) UsageType
	filterDependency func(dep *model.ProjectDependency) UsageType
	filterType       UsageType
}

func (f *simpleProjectFilter) FilterProject(proj *model.Project) UsageType {
	if f.filterProject == nil {
		return DontCare
	}

	return f.filterProject(proj)
}

func (f *simpleProjectFilter) FilterDependency(dep *model.ProjectDependency) UsageType {
	if f.filterDependency == nil {
		return DontCare
	}

	return f.filterDependency(dep)
}

func (f *simpleProjectFilter) Decide(u UsageType) bool {
	switch {
	case u == Include:
		return true
	case u == Exclude:
		return false
	case u == DontCare && f.filterType == Exclude:
		return true
	case u == DontCare && f.filterType == Include:
		return false
	default:
		return true
	}
}

func NewSimpleProjectFilter(filterType UsageType, projFilter func(*model.Project) bool) ProjsAndDepsFilter {
	return &simpleProjectFilter{
		filterProject: func(src *model.Project) UsageType {
			if !projFilter(src) {
				return DontCare
			}

			return filterType
		},
		filterDependency: func(dep *model.ProjectDependency) UsageType {
			sm := projFilter(dep.Source)
			dm := projFilter(dep.Target)

			if filterType == Include {
				return utils.IIf(sm && dm, Include, DontCare)

			} else {
				return utils.IIf(sm || dm, Exclude, DontCare)
			}
		},
		filterType: filterType,
	}
}

type andProjsAndDepsFilter struct {
	filters []ProjsAndDepsFilter
}

func (m *andProjsAndDepsFilter) FilterProject(proj *model.Project) UsageType {
	result := lo.Map(m.filters, func(f ProjsAndDepsFilter, _ int) UsageType { return f.FilterProject(proj) })
	result = lo.Uniq(result)

	if len(result) != 1 {
		return DontCare
	} else {
		return result[0]
	}
}

func (m *andProjsAndDepsFilter) FilterDependency(dep *model.ProjectDependency) UsageType {
	result := lo.Map(m.filters, func(f ProjsAndDepsFilter, _ int) UsageType { return f.FilterDependency(dep) })
	result = lo.Uniq(result)

	if len(result) != 1 {
		return DontCare
	} else {
		return result[0]
	}
}

func (m *andProjsAndDepsFilter) Decide(u UsageType) bool {
	switch u {
	case Include:
		return true
	case Exclude:
		return false
	default:
		for _, f := range m.filters {
			if !f.Decide(u) {
				return false
			}
		}
		return true
	}
}

type orProjsAndDepsFilter struct {
	filters []ProjsAndDepsFilter
}

func (m *orProjsAndDepsFilter) FilterProject(proj *model.Project) UsageType {
	result := DontCare
	for _, f := range m.filters {
		result = result.Merge(f.FilterProject(proj))
	}
	return result
}

func (m *orProjsAndDepsFilter) FilterDependency(dep *model.ProjectDependency) UsageType {
	result := DontCare
	for _, f := range m.filters {
		result = result.Merge(f.FilterDependency(dep))
	}
	return result
}

func (m *orProjsAndDepsFilter) Decide(u UsageType) bool {
	switch u {
	case Include:
		return true
	case Exclude:
		return false
	default:
		for _, f := range m.filters {
			if !f.Decide(u) {
				return false
			}
		}
		return true
	}
}
