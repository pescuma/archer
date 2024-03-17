package filters

import "github.com/pescuma/archer/lib/model"

func ParseAndFilterProjects(ps *model.Projects, filters []string, ft model.FilterType) ([]*model.Project, error) {
	if len(filters) == 0 {
		return ps.ListProjects(ft), nil
	}

	var fs []ProjsAndDepsFilter
	for _, fe := range filters {
		filter, err := ParseProjsAndDepsFilter(ps, fe, Include)
		if err != nil {
			return nil, err
		}

		fs = append(fs, filter)
	}

	fs = append(fs, CreateProjsAndDepsIgnoreExternalDependenciesFilter())

	if ft == model.FilterExcludeExternal {
		fs = append(fs, CreateProjsAndDepsIgnoreExternalDependenciesFilter())
	}

	return FilterProjects(GroupProjsAnDepsFilters(fs...), ps.ListProjects(ft)), nil
}

func FilterProjects(filter ProjsAndDepsFilter, ps []*model.Project) []*model.Project {
	matched := map[*model.Project]bool{}

	for _, p := range ps {
		if IncludeProject(filter, p) {
			matched[p] = true
		}

		for _, d := range p.ListDependencies(model.FilterAll) {
			if IncludeDependency(filter, d) {
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

func FilterDependencies(filter ProjsAndDepsFilter, ds map[string]*model.ProjectDependency) []*model.ProjectDependency {
	var result []*model.ProjectDependency
	for _, d := range ds {
		if IncludeDependency(filter, d) {
			result = append(result, d)
		}
	}
	return result
}

func IncludeDependency(filter ProjsAndDepsFilter, d *model.ProjectDependency) bool {
	return filter.Decide(filter.FilterDependency(d))
}

func IncludeProject(filter ProjsAndDepsFilter, p *model.Project) bool {
	return filter.Decide(filter.FilterProject(p))
}

func GroupProjsAnDepsFilters(filters ...ProjsAndDepsFilter) ProjsAndDepsFilter {
	return &orProjsAndDepsFilter{filters}
}
