package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlProject struct {
	ID          model.ID
	Name        string
	ProjectName string   `gorm:"index"`
	Groups      []string `gorm:"serializer:json"`
	Type        model.ProjectType

	RootDir     string
	ProjectFile string

	RepositoryID *model.ID `gorm:"index"`

	Sizes     map[string]*sqlSize  `gorm:"serializer:json"`
	Size      *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes   *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics   *sqlMetricsAggregate `gorm:"embedded"`
	Data      map[string]string    `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	DependencySources []sqlProjectDependency `gorm:"foreignKey:SourceID"`
	DependencyTargets []sqlProjectDependency `gorm:"foreignKey:TargetID"`
	Dirs              []sqlProjectDirectory  `gorm:"foreignKey:ProjectID"`
	Files             []sqlFile              `gorm:"foreignKey:ProjectID"`
}

func newSqlProject(p *model.Project) *sqlProject {
	sp := &sqlProject{
		ID:           p.ID,
		Name:         p.String(),
		ProjectName:  p.Name,
		Groups:       p.Groups,
		Type:         p.Type,
		RootDir:      p.RootDir,
		ProjectFile:  p.ProjectFile,
		RepositoryID: p.RepositoryID,
		Sizes:        map[string]*sqlSize{},
		Size:         toSqlSize(p.Size),
		Changes:      toSqlChanges(p.Changes),
		Metrics:      toSqlMetricsAggregate(p.Metrics, p.Size),
		Data:         encodeMap(p.Data),
		FirstSeen:    p.FirstSeen,
		LastSeen:     p.LastSeen,
	}

	for k, v := range p.Sizes {
		sp.Sizes[k] = toSqlSize(v)
	}

	if len(sp.Sizes) == 0 {
		sp.Sizes = nil
	}

	return sp
}

func (s *sqlProject) CacheKey() string {
	return s.ID.String()
}
