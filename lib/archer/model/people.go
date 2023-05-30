package model

import (
	"github.com/samber/lo"
)

type People struct {
	peopleByName map[string]*Person
	teamsByName  map[string]*Team
	teamsByID    map[UUID]*Team
}

func NewPeople() *People {
	return &People{
		peopleByName: map[string]*Person{},
		teamsByName:  map[string]*Team{},
		teamsByID:    map[UUID]*Team{},
	}
}

func (ps *People) GetOrCreatePerson(name string) *Person {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.peopleByName[name]

	if !ok {
		result = NewPerson(name)
		ps.peopleByName[name] = result
	}

	return result
}

func (ps *People) ListPeople() []*Person {
	return lo.Values(ps.peopleByName)
}

func (ps *People) ChangePersonName(person *Person, name string) {
	delete(ps.peopleByName, person.Name)

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
	}

	return result
}

func (ps *People) GeTeamByID(id UUID) *Team {
	return ps.teamsByID[id]
}

func (ps *People) ListTeams() []*Team {
	return lo.Values(ps.teamsByName)
}
