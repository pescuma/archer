package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlSeenAt struct {
	FirstSeen *time.Time
	LastSeen  *time.Time
}

func newSqlSeenAt(m *model.SeenAt) *sqlSeenAt {
	return &sqlSeenAt{
		FirstSeen: encodeTime(m.FirstSeen),
		LastSeen:  encodeTime(m.LastSeen),
	}
}

func (s *sqlSeenAt) ToModel() *model.SeenAt {
	if s == nil {
		return &model.SeenAt{}
	}

	return &model.SeenAt{
		FirstSeen: decodeTime(s.FirstSeen),
		LastSeen:  decodeTime(s.LastSeen),
	}
}
