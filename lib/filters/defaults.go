package filters

import "github.com/pescuma/archer/lib/model"

func CreateIgnoreFilter() Filter {
	return NewProjectFilter(Exclude, func(proj *model.Project) bool { return proj.Ignore })
}

func CreateIgnoreExternalDependenciesFilter() Filter {
	return NewProjectFilter(Exclude, func(proj *model.Project) bool { return proj.IsExternalDependency() })
}
