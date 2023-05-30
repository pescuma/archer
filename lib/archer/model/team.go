package model

import (
	"github.com/samber/lo"
)

type Team struct {
	Name string
	ID   UUID

	people  map[string]*Person
	Size    *Size
	Changes *Changes
	Metrics *Metrics
	Data    map[string]string
}

func NewTeam(name string, id *UUID) *Team {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("t")
	} else {
		uuid = *id
	}

	return &Team{
		Name:    name,
		ID:      uuid,
		people:  map[string]*Person{},
		Size:    NewSize(),
		Changes: NewChanges(),
		Metrics: NewMetrics(),
		Data:    map[string]string{},
	}
}

func (t *Team) AddPerson(person *Person) {
	t.people[person.Name] = person
}

func (t *Team) ListPeople() []*Person {
	return lo.Values(t.people)
}
