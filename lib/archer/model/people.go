package model

import (
	"github.com/samber/lo"
)

type People struct {
	all map[string]*Person
}

func NewPeople() *People {
	return &People{
		all: map[string]*Person{},
	}
}

func (ps *People) GetOrCreate(name string) *Person {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.all[name]

	if !ok {
		result = NewPerson(name)
		ps.all[name] = result
	}

	return result
}

func (ps *People) List() []*Person {
	return lo.Values(ps.all)
}

func (ps *People) ChangeName(person *Person, name string) {
	delete(ps.all, person.Name)

	person.Name = name

	ps.all[name] = person
}
