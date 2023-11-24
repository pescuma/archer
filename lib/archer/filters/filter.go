package filters

import "github.com/pescuma/archer/lib/archer/model"

type Filter interface {
	FilterProject(proj *model.Project) UsageType

	FilterDependency(dep *model.ProjectDependency) UsageType

	// Decide does not return DontCase, so it should decide what to do in this case
	Decide(u UsageType) UsageType
}
