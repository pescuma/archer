package filters

import (
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

type andFilter struct {
	filters []Filter
}

func (m *andFilter) FilterProject(proj *model.Project) UsageType {
	result := lo.Map(m.filters, func(f Filter, _ int) UsageType { return f.FilterProject(proj) })
	result = lo.Uniq(result)

	if len(result) != 1 {
		return DontCare
	} else {
		return result[0]
	}
}

func (m *andFilter) FilterDependency(dep *model.ProjectDependency) UsageType {
	result := lo.Map(m.filters, func(f Filter, _ int) UsageType { return f.FilterDependency(dep) })
	result = lo.Uniq(result)

	if len(result) != 1 {
		return DontCare
	} else {
		return result[0]
	}
}

func (m *andFilter) Decide(u UsageType) UsageType {
	result := utils.IIf(u == DontCare, Include, u)
	for _, f := range m.filters {
		result = result.Merge(f.Decide(u))
	}
	return result
}
