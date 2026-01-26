package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlFile struct {
	ID   model.ID
	Name string

	ProjectID          *model.ID `gorm:"index"`
	ProjectDirectoryID *model.ID `gorm:"index"`
	RepositoryID       *model.ID `gorm:"index"`

	ProductAreaID *model.ID `gorm:"index"`

	Exists bool
	Ignore bool

	Size    *sqlSize          `gorm:"embedded;embeddedPrefix:size_"`
	Changes *sqlChanges       `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetrics       `gorm:"embedded"`
	SeenAt  *sqlSeenAt        `gorm:"embedded"`
	Data    map[string]string `gorm:"serializer:json"`

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
		Ignore:             f.Ignore,
		Size:               newSqlSize(f.Size),
		Changes:            newSqlChanges(f.Changes),
		Metrics:            newSqlMetrics(f.Metrics),
		SeenAt:             newSqlSeenAt(f.SeenAt),
		Data:               encodeMap(f.Data),
	}
}

func (s *sqlFile) ToModel() *model.File {
	return &model.File{
		ID:                 s.ID,
		Path:               s.Name,
		ProjectID:          s.ProjectID,
		ProjectDirectoryID: s.ProjectDirectoryID,
		RepositoryID:       s.RepositoryID,
		ProductAreaID:      s.ProductAreaID,
		Exists:             s.Exists,
		Ignore:             s.Ignore,
		Size:               s.Size.ToModel(),
		Changes:            s.Changes.ToModel(),
		Metrics:            s.Metrics.toModel(),
		SeenAt:             s.SeenAt.ToModel(),
		Data:               decodeMap(s.Data),
	}
}

func (s *sqlFile) CacheKey() string {
	return s.ID.String()
}
