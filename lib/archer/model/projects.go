package model

import (
	"sort"
	"strings"
)

type Projects struct {
	all map[string]*Project
}

func NewProjects() *Projects {
	return &Projects{
		all: map[string]*Project{},
	}
}

func (ps *Projects) Get(root, name string) *Project {
	if len(root) == 0 {
		panic("empty root not supported")
	}
	if len(name) == 0 {
		panic("empty name not supported")
	}

	key := root + "\n" + name
	result, ok := ps.all[key]

	if !ok {
		result = NewProject(root, name)
		ps.all[key] = result
	}

	return result
}

func (ps *Projects) FilterProjects(filters []string, ft FilterType) ([]*Project, error) {
	if len(filters) == 0 {
		return ps.ListProjects(ft), nil
	}

	matched := map[*Project]bool{}
	for _, fe := range filters {
		filter, err := ParseFilter(ps, fe, Include)
		if err != nil {
			return nil, err
		}

		for _, p := range ps.ListProjects(ft) {
			if filter.Decide(filter.FilterProject(p)) == Include {
				matched[p] = true
			}

			for _, d := range p.ListDependencies(ft) {
				if filter.Decide(filter.FilterDependency(d)) == Include {
					matched[d.Source] = true
					matched[d.Target] = true
				}
			}
		}
	}

	result := make([]*Project, 0, len(matched))
	for p := range matched {
		result = append(result, p)
	}

	sortProjects(result)

	return result, nil
}

func (ps *Projects) ListProjects(ft FilterType) []*Project {
	result := make([]*Project, 0, len(ps.all))

	for _, v := range ps.all {
		if ft == FilterExcludeExternal && v.IsExternalDependency() {
			continue
		}

		result = append(result, v)
	}

	sortProjects(result)

	return result
}

func (ps *Projects) ListProjectsByRoot(root string, ft FilterType) []*Project {
	result := make([]*Project, 0, len(ps.all))

	for _, v := range ps.all {
		if ft == FilterExcludeExternal && v.IsExternalDependency() {
			continue
		}

		if v.Root != root {
			continue
		}

		result = append(result, v)
	}

	sortProjects(result)

	return result
}

func sortProjects(result []*Project) {
	sort.Slice(result, func(i, j int) bool {
		pi := result[i]
		pj := result[j]

		if pi.IsCode() && pj.IsExternalDependency() {
			return true
		}

		if pi.IsExternalDependency() && pj.IsCode() {
			return false
		}

		return strings.TrimLeft(pi.Name, ":") < strings.TrimLeft(pj.Name, ":")
	})
}
