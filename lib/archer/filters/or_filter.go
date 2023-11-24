package filters

import (
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
)

func GroupFilters(filters ...Filter) Filter {
	return &orFilter{filters}
}

type orFilter struct {
	filters []Filter
}

func (m *orFilter) FilterProject(proj *model.Project) UsageType {
	result := DontCare
	for _, f := range m.filters {
		result = result.Merge(f.FilterProject(proj))
	}
	return result
}

func (m *orFilter) FilterDependency(dep *model.ProjectDependency) UsageType {
	result := DontCare
	for _, f := range m.filters {
		result = result.Merge(f.FilterDependency(dep))
	}
	return result
}

func (m *orFilter) Decide(u UsageType) UsageType {
	result := utils.IIf(u == DontCare, Include, u)
	for _, f := range m.filters {
		result = result.Merge(f.Decide(u))
	}
	return result
}
