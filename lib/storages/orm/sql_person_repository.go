package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlPersonRepository struct {
	PersonID     model.ID `gorm:"primaryKey"`
	RepositoryID model.ID `gorm:"primaryKey"`

	SeenAt *sqlSeenAt `gorm:"embedded"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlPersonRepository(r *model.PersonRepository) *sqlPersonRepository {
	return &sqlPersonRepository{
		PersonID:     r.PersonID,
		RepositoryID: r.RepositoryID,
		SeenAt:       newSqlSeenAt(r.SeenAt),
	}
}

func (s *sqlPersonRepository) CacheKey() string {
	return compositeKey(s.PersonID.String(), s.RepositoryID.String())
}
