package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlRepository struct {
	ID      model.ID
	Name    string
	RootDir string `gorm:"uniqueIndex"`
	VCS     string
	Branch  string

	CommitsTotal int
	FilesTotal   *int
	FilesHead    *int

	Data      map[string]string `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	Commits []sqlRepositoryCommit `gorm:"foreignKey:RepositoryID"`
	Files   []sqlFile             `gorm:"foreignKey:RepositoryID"`
	People  []sqlPersonRepository `gorm:"foreignKey:RepositoryID"`
}

func newSqlRepository(r *model.Repository) *sqlRepository {
	return &sqlRepository{
		ID:           r.ID,
		Name:         r.Name,
		RootDir:      r.RootDir,
		VCS:          r.VCS,
		Branch:       r.Branch,
		Data:         encodeMap(r.Data),
		FirstSeen:    r.FirstSeen,
		LastSeen:     r.LastSeen,
		CommitsTotal: r.CountCommits(),
		FilesTotal:   encodeMetric(r.FilesTotal),
		FilesHead:    encodeMetric(r.FilesHead),
	}
}

func (s *sqlRepository) CacheKey() string {
	return s.ID.String()
}
