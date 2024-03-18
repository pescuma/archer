package model

import (
	"github.com/samber/lo"
)

type People struct {
	personMaxID ID
	peopleByID  map[ID]*Person

	productAreasByName map[string]*ProductArea
	productAreasByID   map[UUID]*ProductArea
}

func NewPeople() *People {
	return &People{
		peopleByID:         map[ID]*Person{},
		productAreasByName: map[string]*ProductArea{},
		productAreasByID:   map[UUID]*ProductArea{},
	}
}

func (ps *People) GetPersonByID(id ID) *Person {
	return ps.peopleByID[id]
}

func (ps *People) GetOrCreatePerson(id *ID) *Person {
	if id != nil {
		if result, ok := ps.peopleByID[*id]; ok {
			return result
		}
	}

	result := NewPerson(createID(&ps.personMaxID, id))
	ps.peopleByID[result.ID] = result
	return result
}

func (ps *People) ListPeople() []*Person {
	return lo.Values(ps.peopleByID)
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
