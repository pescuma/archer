package model

import (
	"github.com/samber/lo"
)

type People struct {
	peopleByName        map[string]*Person
	peopleByID          map[UUID]*Person
	organizationsByName map[string]*Organization
	organizationsByID   map[UUID]*Organization
	productAreasByName  map[string]*ProductArea
	productAreasByID    map[UUID]*ProductArea
}

func NewPeople() *People {
	return &People{
		peopleByName:        map[string]*Person{},
		peopleByID:          map[UUID]*Person{},
		organizationsByName: map[string]*Organization{},
		organizationsByID:   map[UUID]*Organization{},
		productAreasByName:  map[string]*ProductArea{},
		productAreasByID:    map[UUID]*ProductArea{},
	}
}

func (ps *People) GetPerson(name string) *Person {
	return ps.peopleByName[name]
}

func (ps *People) GetOrCreatePerson(name string) *Person {
	return ps.GetOrCreatePersonEx(name, nil)
}

func (ps *People) GetOrCreatePersonEx(name string, id *UUID) *Person {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.peopleByName[name]

	if !ok {
		result = NewPerson(name, id)
		ps.peopleByName[name] = result
		ps.peopleByID[result.ID] = result
	}

	if id != nil && result.ID != *id {
		panic("id mismatch")
	}

	return result
}

func (ps *People) GetPersonByID(id UUID) *Person {
	return ps.peopleByID[id]
}

func (ps *People) ListPeople() []*Person {
	return lo.Values(ps.peopleByName)
}

func (ps *People) ChangePersonName(person *Person, name string) {
	if _, ok := ps.peopleByName[name]; ok {
		panic("name already exists")
	}

	person.Name = name

	ps.peopleByName[name] = person
}

func (ps *People) GetOrCreateOrganization(name string) *Organization {
	return ps.GetOrCreateOrganizationEx(name, nil)

}

func (ps *People) GetOrCreateOrganizationEx(name string, id *UUID) *Organization {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.organizationsByName[name]

	if !ok {
		result = NewOrganization(name, id)
		ps.organizationsByName[name] = result
		ps.organizationsByID[result.ID] = result
	}

	return result
}

func (ps *People) GetOrganizationByID(id UUID) *Organization {
	return ps.organizationsByID[id]
}

func (ps *People) ListOrganizations() []*Organization {
	return lo.Values(ps.organizationsByName)
}

func (ps *People) ListGroupsByID() map[UUID]*Group {
	result := map[UUID]*Group{}
	for _, o := range ps.ListOrganizations() {
		for _, g := range o.ListGroups() {
			result[g.ID] = g
		}
	}
	return result
}

func (ps *People) ListTeamsByID() map[UUID]*Team {
	result := map[UUID]*Team{}
	for _, o := range ps.ListOrganizations() {
		for _, g := range o.ListGroups() {
			for _, t := range g.ListTeams() {
				result[t.ID] = t
			}
		}
	}
	return result
}

func (ps *People) GetOrCreateProductArea(name string) *ProductArea {
	return ps.GetOrCreateProductAreaEx(name, nil)

}

func (ps *People) GetOrCreateProductAreaEx(name string, id *UUID) *ProductArea {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.productAreasByName[name]

	if !ok {
		result = NewProductArea(name, id)
		ps.productAreasByName[name] = result
		ps.productAreasByID[result.ID] = result
	}

	return result
}

func (ps *People) GetProductAreaByID(id UUID) *ProductArea {
	return ps.productAreasByID[id]
}

func (ps *People) ListProductAreas() []*ProductArea {
	return lo.Values(ps.productAreasByName)
}
