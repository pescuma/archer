package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlRepositoryCommit struct {
	ID           model.ID
	RepositoryID model.ID `gorm:"index"`
	Name         string
	Message      string
	Parents      []model.ID `gorm:"serializer:json"`
	Children     []model.ID `gorm:"serializer:json"`
	Date         time.Time  `gorm:"index"`
	DateAuthored time.Time
	Ignore       bool

	FilesModified *int
	FilesCreated  *int
	FilesDeleted  *int
	LinesModified *int
	LinesAdded    *int
	LinesDeleted  *int
	Blame         *sqlBlame `gorm:"embedded;embeddedPrefix:blame_"`

	CreatedAt time.Time
	UpdatedAt time.Time

	People []sqlRepositoryCommitPerson `gorm:"foreignKey:CommitID"`
	Files  []sqlRepositoryCommitFile   `gorm:"foreignKey:CommitID"`
}

func newSqlRepositoryCommit(r *model.Repository, c *model.RepositoryCommit) *sqlRepositoryCommit {
	return &sqlRepositoryCommit{
		ID:            c.ID,
		RepositoryID:  r.ID,
		Name:          c.Hash,
		Message:       c.Message,
		Parents:       c.Parents,
		Children:      c.Children,
		Date:          c.Date,
		DateAuthored:  c.DateAuthored,
		Ignore:        c.Ignore,
		FilesModified: encodeMetric(c.FilesModified),
		FilesCreated:  encodeMetric(c.FilesCreated),
		FilesDeleted:  encodeMetric(c.FilesDeleted),
		LinesModified: encodeMetric(c.LinesModified),
		LinesAdded:    encodeMetric(c.LinesAdded),
		LinesDeleted:  encodeMetric(c.LinesDeleted),
		Blame:         toSqlBlame(c.Blame),
	}
}

func (s *sqlRepositoryCommit) CacheKey() string {
	return string(s.ID)
}
