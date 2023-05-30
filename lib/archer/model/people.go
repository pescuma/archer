package model

import (
	"github.com/samber/lo"
)

type People struct {
	peopleByName map[string]*Person
	peopleByID   map[UUID]*Person
	teamsByName  map[string]*Team
	teamsByID    map[UUID]*Team
}

func NewPeople() *People {
	return &People{
		peopleByName: map[string]*Person{},
		peopleByID:   map[UUID]*Person{},
		teamsByName:  map[string]*Team{},
		teamsByID:    map[UUID]*Team{},
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

func (ps *People) GetOrCreateTeam(name string) *Team {
	return ps.GetOrCreateTeamEx(name, nil)

}

func (ps *People) GetOrCreateTeamEx(name string, id *UUID) *Team {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.teamsByName[name]

	if !ok {
		result = NewTeam(name, id)
		ps.teamsByName[name] = result
		ps.teamsByID[result.ID] = result
	}

	return result
}

func (ps *People) GetTeamByID(id UUID) *Team {
	return ps.teamsByID[id]
}

func (ps *People) ListTeams() []*Team {
	return lo.Values(ps.teamsByName)
}
