package filters

import "github.com/pescuma/archer/lib/model"

type ProjectFilter interface {
	FilterProject(proj *model.Project) bool

	FilterDependency(dep *model.ProjectDependency) bool
}

type ProjectFilterWithUsage interface {
	FilterProject(proj *model.Project) UsageType

	FilterDependency(dep *model.ProjectDependency) UsageType

	// Decide does not return DontCase, so it should decide what to do in this case
	Decide(u UsageType) bool
}
