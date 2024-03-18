package model

import "time"

type PersonRepository struct {
	PersonID     ID
	RepositoryID ID

	FirstSeen time.Time
	LastSeen  time.Time
}

func NewPersonRepository(personID ID, repositoryID ID) *PersonRepository {
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
