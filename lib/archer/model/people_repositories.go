package model

import "github.com/samber/lo"

type PeopleRepositories struct {
	byID map[UUID]*PersonRepositories
}

func NewPeopleRepositories() *PeopleRepositories {
	return &PeopleRepositories{
		byID: make(map[UUID]*PersonRepositories),
	}
}

func (p *PeopleRepositories) GetOrCreatePerson(personID UUID) *PersonRepositories {
	pr, ok := p.byID[personID]

	if !ok {
		pr = NewPersonRepositories(personID)
		p.byID[personID] = pr
	}

	return pr
}

func (p *PeopleRepositories) List() []*PersonRepositories {
	return lo.Values(p.byID)
}
