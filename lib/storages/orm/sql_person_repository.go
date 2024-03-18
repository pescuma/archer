package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlPersonRepository struct {
	PersonID     model.ID `gorm:"primaryKey"`
	RepositoryID model.ID `gorm:"primaryKey"`

	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlPersonRepository(r *model.PersonRepository) *sqlPersonRepository {
	return &sqlPersonRepository{
		PersonID:     r.PersonID,
		RepositoryID: r.RepositoryID,
		FirstSeen:    r.FirstSeen,
		LastSeen:     r.LastSeen,
	}
}

func (s *sqlPersonRepository) CacheKey() string {
	return compositeKey(s.PersonID.String(), s.RepositoryID.String())
}
