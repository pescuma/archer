package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlPersonFile struct {
	PersonID model.ID `gorm:"primaryKey"`
	FileID   model.ID `gorm:"primaryKey"`

	SeenAt *sqlSeenAt `gorm:"embedded"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlPersonFile(f *model.PersonFile) *sqlPersonFile {
	return &sqlPersonFile{
		PersonID: f.PersonID,
		FileID:   f.FileID,
		SeenAt:   newSqlSeenAt(f.SeenAt),
	}
}

func (s *sqlPersonFile) CacheKey() string {
	return compositeKey(s.PersonID.String(), s.FileID.String())
}
