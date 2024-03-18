package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlIgnoreRule struct {
	ID      model.ID `gorm:"primaryKey"`
	Type    model.IgnoreRuleType
	Rule    string
	Deleted bool

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

func newSqlIgnoreRule(r *model.IgnoreRule) *sqlIgnoreRule {
	return &sqlIgnoreRule{
		ID:        r.ID,
		Type:      r.Type,
		Rule:      r.Rule,
		Deleted:   r.Deleted,
		DeletedAt: r.DeletedAt,
	}
}

func (s *sqlIgnoreRule) ToModel() *model.IgnoreRule {
	return &model.IgnoreRule{
		ID:        s.ID,
		Type:      s.Type,
		Rule:      s.Rule,
		Deleted:   s.Deleted,
		DeletedAt: s.DeletedAt,
	}
}

func (s *sqlIgnoreRule) CacheKey() string {
	return s.ID.String()
}
