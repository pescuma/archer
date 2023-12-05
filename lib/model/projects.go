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

func (ps *Projects) GetOrCreate(name string) *Project {
	return ps.GetOrCreateEx(name, nil)
}

func (ps *Projects) GetOrCreateEx(name string, id *UUID) *Project {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.byName[name]

	if !ok {
		result = NewProject(name, id)
		ps.byName[name] = result
		ps.byID[result.ID] = result
	}

	return result
}

func (ps *Projects) GetByID(id UUID) *Project {
	return ps.byID[id]
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
