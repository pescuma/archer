package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlMonthLines struct {
	ID model.ID `gorm:"primaryKey"`

	Month        string
	RepositoryID model.ID
	AuthorID     model.ID
	CommitterID  model.ID
	ProjectID    *model.ID

	Changes *sqlChanges `gorm:"embedded;embeddedPrefix:changes_"`
	Blame   *sqlBlame   `gorm:"embedded;embeddedPrefix:blame_"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlMonthLines(l *model.MonthlyStatsLine) *sqlMonthLines {
	return &sqlMonthLines{
		ID:           l.ID,
		Month:        l.Month,
		RepositoryID: l.RepositoryID,
		AuthorID:     l.AuthorID,
		CommitterID:  l.CommitterID,
		ProjectID:    l.ProjectID,
		Changes:      toSqlChanges(l.Changes),
		Blame:        toSqlBlame(l.Blame),
	}
}

func (s *sqlMonthLines) CacheKey() string {
	return s.ID.String()
}
