package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlRepositoryCommitPerson struct {
	CommitID model.ID      `gorm:"primaryKey"`
	PersonID model.ID      `gorm:"primaryKey"`
	Role     sqlCommitRole `gorm:"primaryKey"`
	Order    int

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlRepositoryCommitPerson(commit *model.RepositoryCommit, personID model.ID, role sqlCommitRole, order int) *sqlRepositoryCommitPerson {
	return &sqlRepositoryCommitPerson{
		CommitID: commit.ID,
		PersonID: personID,
		Role:     role,
		Order:    order,
	}
}

func (s *sqlRepositoryCommitPerson) CacheKey() string {
	return compositeKey(s.CommitID.String(), s.PersonID.String(), s.Role.String())
}
