package filters

import "github.com/pescuma/archer/lib/model"

func ParseAndFilterProjects(ps *model.Projects, filters []string, ft model.FilterType) ([]*model.Project, error) {
	if len(filters) == 0 {
		return ps.ListProjects(ft), nil
	}

	var fs []ProjectFilterWithUsage
	for _, fe := range filters {
		filter, err := ParseProjectFilterWithUsage(ps, fe, Include)
		if err != nil {
			return nil, err
		}

		fs = append(fs, filter)
	}

	if ft == model.FilterExcludeExternal {
		fs = append(fs, CreateIgnoreExternalDependenciesFilter())
	}

	filter := UnliftProjectFilter(GroupProjectFilters(fs...))

	return FilterProjects(filter, ps.ListProjects(ft)), nil
}

func FilterProjects(filter ProjectFilter, ps []*model.Project) []*model.Project {
	matched := map[*model.Project]bool{}

	for _, p := range ps {
		if filter.FilterProject(p) {
			matched[p] = true
		}

		for _, d := range p.ListDependencies(model.FilterAll) {
			if filter.FilterDependency(d) {
				matched[d.Source] = true
				matched[d.Target] = true
			}
		}
	}

	result := make([]*model.Project, 0, len(matched))
	for _, p := range ps {
		if matched[p] {
			result = append(result, p)
		}
	}
	return result
}

func FilterDependencies(filter ProjectFilter, ds map[string]*model.ProjectDependency) []*model.ProjectDependency {
	var result []*model.ProjectDependency
	for _, d := range ds {
		if filter.FilterDependency(d) {
			result = append(result, d)
		}
	}
	return result
}

func GroupProjectFilters(filters ...ProjectFilterWithUsage) ProjectFilterWithUsage {
	return &projectFilterWithUsageGroup{filters}
}

func LiftProjectFilter(filter ProjectFilter, usage UsageType) ProjectFilterWithUsage {
	return &simpleProjectFilterWithUsage{filter, usage}
}

func LiftProjAndDepFilters(
	filterProject func(*model.Project) bool,
	filterDependency func(*model.ProjectDependency) bool,
	usage UsageType,
) ProjectFilterWithUsage {
	return &simpleProjectFilterWithUsage{
		&simpleProjectFilter{filterProject, filterDependency},
		usage,
	}
}

func UnliftProjectFilter(filter ProjectFilterWithUsage) ProjectFilter {
	return &simpleProjectFilter{
		func(p *model.Project) bool {
			return filter.Decide(filter.FilterProject(p))
		},
		func(d *model.ProjectDependency) bool {
			return filter.Decide(filter.FilterDependency(d))
		},
	}
}
