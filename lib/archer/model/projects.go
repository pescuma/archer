package model

import (
	"sort"
	"strings"
)

type Projects struct {
	byName map[string]*Project
	byID   map[UUID]*Project
}

func NewProjects() *Projects {
	return &Projects{
		byName: map[string]*Project{},
		byID:   map[UUID]*Project{},
	}
}

func (ps *Projects) GetOrCreate(root, name string) *Project {
	if len(root) == 0 {
		panic("empty root not supported")
	}
	if len(name) == 0 {
		panic("empty name not supported")
	}

	key := root + "\n" + name
	result, ok := ps.byName[key]

	if !ok {
		result = NewProject(root, name)
		ps.byName[key] = result
		ps.byID[result.ID] = result
	}

	return result
}

func (ps *Projects) GetByID(id UUID) *Project {
	return ps.byID[id]
}

func (ps *Projects) ChangeID(proj *Project, id UUID) {
	delete(ps.byID, proj.ID)

	proj.ID = id
	ps.byID[proj.ID] = proj
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
	result := make([]*Project, 0, len(ps.byName))

	for _, v := range ps.byName {
		if ft == FilterExcludeExternal && v.IsExternalDependency() {
			continue
		}

		result = append(result, v)
	}

	sortProjects(result)

	return result
}

func (ps *Projects) ListProjectsByRoot(root string, ft FilterType) []*Project {
	result := make([]*Project, 0, len(ps.byName))

	for _, v := range ps.byName {
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
