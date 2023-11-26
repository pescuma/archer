package filters

import (
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

type basicFilter struct {
	filterProject    func(proj *model.Project) UsageType
	filterDependency func(dep *model.ProjectDependency) UsageType
	filterType       UsageType
}

func (b *basicFilter) FilterProject(proj *model.Project) UsageType {
	if b.filterProject == nil {
		return DontCare
	}

	return b.filterProject(proj)
}

func (b *basicFilter) FilterDependency(dep *model.ProjectDependency) UsageType {
	if b.filterDependency == nil {
		return DontCare
	}

	return b.filterDependency(dep)
}

func (b *basicFilter) Decide(u UsageType) UsageType {
	switch {
	case u == DontCare && b.filterType == Exclude:
		return Include
	case u == DontCare && b.filterType == Include:
		return Exclude
	default:
		return u
	}
}

func NewProjectFilter(filterType UsageType, projFilter func(*model.Project) bool) Filter {
	return &basicFilter{
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
