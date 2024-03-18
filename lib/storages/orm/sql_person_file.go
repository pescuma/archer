package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlPersonFile struct {
	PersonID model.ID `gorm:"primaryKey"`
	FileID   model.ID `gorm:"primaryKey"`

	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlPersonFile(f *model.PersonFile) *sqlPersonFile {
	return &sqlPersonFile{
		PersonID:  f.PersonID,
		FileID:    f.FileID,
		FirstSeen: f.FirstSeen,
		LastSeen:  f.LastSeen,
	}
}

func (s *sqlPersonFile) CacheKey() string {
	return compositeKey(s.PersonID.String(), s.FileID.String())
}
