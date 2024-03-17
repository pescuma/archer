package filters

import "github.com/pescuma/archer/lib/model"

func CreateProjsAndDepsIgnoreFilter() ProjsAndDepsFilter {
	return NewSimpleProjectFilter(Exclude, func(proj *model.Project) bool { return proj.Ignore })
}

func CreateProjsAndDepsIgnoreExternalDependenciesFilter() ProjsAndDepsFilter {
	return NewSimpleProjectFilter(Exclude, func(proj *model.Project) bool { return proj.IsExternalDependency() })
}
