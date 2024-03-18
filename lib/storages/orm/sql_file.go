package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlFile struct {
	ID   model.ID
	Name string

	ProjectID          *model.UUID `gorm:"index"`
	ProjectDirectoryID *model.UUID `gorm:"index"`
	RepositoryID       *model.UUID `gorm:"index"`

	ProductAreaID *model.ID `gorm:"index"`

	Exists    bool
	Size      *sqlSize          `gorm:"embedded;embeddedPrefix:size_"`
	Changes   *sqlChanges       `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics   *sqlMetrics       `gorm:"embedded"`
	Data      map[string]string `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitFiles []sqlRepositoryCommitFile `gorm:"foreignKey:FileID"`
	People      []sqlPersonFile           `gorm:"foreignKey:FileID"`
}

func newSqlFile(f *model.File) *sqlFile {
	return &sqlFile{
		ID:                 f.ID,
		Name:               f.Path,
		ProjectID:          f.ProjectID,
		ProjectDirectoryID: f.ProjectDirectoryID,
		RepositoryID:       f.RepositoryID,
		ProductAreaID:      f.ProductAreaID,
		Exists:             f.Exists,
		Size:               toSqlSize(f.Size),
		Changes:            toSqlChanges(f.Changes),
		Metrics:            toSqlMetrics(f.Metrics),
		Data:               encodeMap(f.Data),
		FirstSeen:          f.FirstSeen,
		LastSeen:           f.LastSeen,
	}
}

func (s *sqlFile) CacheKey() string {
	return s.ID.String()
}
