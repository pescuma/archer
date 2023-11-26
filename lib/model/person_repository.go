package model

import "time"

type PersonRepository struct {
	PersonID     UUID
	RepositoryID UUID

	FirstSeen time.Time
	LastSeen  time.Time
}

func NewPersonRepository(personID UUID, repositoryID UUID) *PersonRepository {
	return &PersonRepository{
		PersonID:     personID,
		RepositoryID: repositoryID,
	}
}

func (p *PersonRepository) SeenAt(ts ...time.Time) {
	empty := time.Time{}

	for _, t := range ts {
		t = t.UTC().Round(time.Second)

		if p.FirstSeen == empty || t.Before(p.FirstSeen) {
			p.FirstSeen = t
		}
		if p.LastSeen == empty || t.After(p.LastSeen) {
			p.LastSeen = t
		}
	}
}
