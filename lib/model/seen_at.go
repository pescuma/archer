package model

import "time"

type SeenAt struct {
	FirstSeen time.Time
	LastSeen  time.Time
}

func NewSeenAt() *SeenAt {
	result := &SeenAt{}
	return result
}

func (s *SeenAt) Clear() {
	empty := time.Time{}

	s.FirstSeen = empty
	s.LastSeen = empty
}

func (s *SeenAt) Add(ts ...time.Time) {
	empty := time.Time{}

	for _, t := range ts {
		t = t.UTC().Round(time.Second)

		if s.FirstSeen == empty || t.Before(s.FirstSeen) {
			s.FirstSeen = t
		}
		if s.LastSeen == empty || t.After(s.LastSeen) {
			s.LastSeen = t
		}
	}
}

func (s *SeenAt) Merge(o *SeenAt) {
	empty := time.Time{}

	if o.FirstSeen != empty {
		s.Add(o.FirstSeen)
	}
	if o.LastSeen != empty {
		s.Add(o.LastSeen)
	}
}
