package model

import (
	"github.com/samber/lo"
)

type People struct {
	peopleByName       map[string]*Person
	peopleByID         map[UUID]*Person
	productAreasByName map[string]*ProductArea
	productAreasByID   map[UUID]*ProductArea
}

func NewPeople() *People {
	return &People{
		peopleByName:       map[string]*Person{},
		peopleByID:         map[UUID]*Person{},
		productAreasByName: map[string]*ProductArea{},
		productAreasByID:   map[UUID]*ProductArea{},
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
