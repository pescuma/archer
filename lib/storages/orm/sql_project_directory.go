package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlProjectDirectory struct {
	ID        model.ID
	ProjectID model.ID `gorm:"index"`
	Name      string
	Type      model.ProjectDirectoryType

	Size      *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes   *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics   *sqlMetricsAggregate `gorm:"embedded"`
	Data      map[string]string    `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

	Files []sqlFile `gorm:"foreignKey:ProjectDirectoryID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlProjectDirectory(d *model.ProjectDirectory, p *model.Project) *sqlProjectDirectory {
	return &sqlProjectDirectory{
		ID:        d.ID,
		ProjectID: p.ID,
		Name:      d.RelativePath,
		Type:      d.Type,
		Size:      toSqlSize(d.Size),
		Changes:   toSqlChanges(d.Changes),
		Metrics:   toSqlMetricsAggregate(d.Metrics, d.Size),
		Data:      encodeMap(d.Data),
		FirstSeen: d.FirstSeen,
		LastSeen:  d.LastSeen,
	}
}

func (s *sqlProjectDirectory) CacheKey() string {
	return string(s.ID)
}
