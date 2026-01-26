package model

type PersonRepository struct {
	PersonID     ID
	RepositoryID ID

	SeenAt *SeenAt
}

func NewPersonRepository(personID ID, repositoryID ID) *PersonRepository {
	return &PersonRepository{
		PersonID:     personID,
		RepositoryID: repositoryID,
		SeenAt:       NewSeenAt(),
	}
}
