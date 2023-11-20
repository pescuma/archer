package model

import "github.com/samber/lo"

type PersonRepositories struct {
	PersonID UUID

	byID map[UUID]*PersonRepository
}

func NewPersonRepositories(personID UUID) *PersonRepositories {
	return &PersonRepositories{
		PersonID: personID,
		byID:     make(map[UUID]*PersonRepository),
	}
}

func (l *PersonRepositories) GetOrCreateRepository(repositoryID UUID) *PersonRepository {
	pr, ok := l.byID[repositoryID]

	if !ok {
		pr = NewPersonRepository(repositoryID)
		l.byID[repositoryID] = pr
	}

	return pr
}

func (l *PersonRepositories) List() []*PersonRepository {
	return lo.Values(l.byID)
}
