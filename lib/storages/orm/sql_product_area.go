package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlProductArea struct {
	ID   model.ID
	Name string

	Size    *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetricsAggregate `gorm:"embedded"`
	Data    map[string]string    `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlProductArea(a *model.ProductArea) *sqlProductArea {
	return &sqlProductArea{
		ID:      a.ID,
		Name:    a.Name,
		Size:    newSqlSize(a.Size),
		Changes: toSqlChanges(a.Changes),
		Metrics: toSqlMetricsAggregate(a.Metrics, a.Size),
		Data:    encodeMap(a.Data),
	}
}

func (s *sqlProductArea) CacheKey() string {
	return s.ID.String()
}
