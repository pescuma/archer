package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlPerson struct {
	ID   model.ID
	Name string

	Names     []string          `gorm:"serializer:json"`
	Emails    []string          `gorm:"serializer:json"`
	Changes   *sqlChanges       `gorm:"embedded;embeddedPrefix:changes_"`
	Blame     *sqlBlame         `gorm:"embedded;embeddedPrefix:blame_"`
	Data      map[string]string `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	Commits      []sqlRepositoryCommitPerson `gorm:"foreignKey:PersonID"`
	Repositories []sqlPersonRepository       `gorm:"foreignKey:PersonID"`
}

func newSqlPerson(p *model.Person) *sqlPerson {
	return &sqlPerson{
		ID:        p.ID,
		Name:      p.Name,
		Names:     p.ListNames(),
		Emails:    p.ListEmails(),
		Changes:   toSqlChanges(p.Changes),
		Blame:     toSqlBlame(p.Blame),
		Data:      encodeMap(p.Data),
		FirstSeen: p.FirstSeen,
		LastSeen:  p.LastSeen,
	}
}

func (s *sqlPerson) CacheKey() string {
	return s.ID.String()
}
