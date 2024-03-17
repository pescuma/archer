package filters

import "github.com/pescuma/archer/lib/model"

func CreateIgnoreExternalDependenciesFilter() ProjectFilterWithUsage {
	return LiftProjAndDepFilters(
		func(proj *model.Project) bool {
			return proj.Type == model.Library
		},
		func(dep *model.ProjectDependency) bool {
			return dep.Target.IsExternalDependency()
		},
		Exclude,
	)
}
