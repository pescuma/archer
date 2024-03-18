package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlProjectDependency struct {
	ID       model.ID
	Name     string
	SourceID model.ID `gorm:"index"`
	TargetID model.ID `gorm:"index"`

	Versions []string          `gorm:"serializer:json"`
	Data     map[string]string `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlProjectDependency(d *model.ProjectDependency) *sqlProjectDependency {
	return &sqlProjectDependency{
		ID:       d.ID,
		Name:     d.String(),
		SourceID: d.Source.ID,
		TargetID: d.Target.ID,
		Versions: d.Versions.Slice(),
		Data:     encodeMap(d.Data),
	}
}

func (s *sqlProjectDependency) CacheKey() string {
	return s.ID.String()
}
