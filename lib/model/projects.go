package model

import (
	"sort"
	"strings"
)

type Projects struct {
	projectMaxID ID
	byName       map[string]*Project
	byID         map[ID]*Project

	dependencyMaxID ID
	directoryMaxID  ID
}

func NewProjects() *Projects {
	return &Projects{
		byName: map[string]*Project{},
		byID:   map[ID]*Project{},
	}
}

func (ps *Projects) GetOrCreate(name string) *Project {
	return ps.GetOrCreateEx(name, nil)
}

func (ps *Projects) GetOrCreateEx(name string, id *ID) *Project {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.byName[name]

	if !ok {
		result = NewProject(createID(&ps.projectMaxID, id), name, ps)
		ps.byName[name] = result
		ps.byID[result.ID] = result
	}

	return result
}

func (ps *Projects) GetByID(id ID) *Project {
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
