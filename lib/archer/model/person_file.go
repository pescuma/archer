package model

import "time"

type PersonFile struct {
	PersonID UUID
	FileID   UUID

	FirstSeen time.Time
	LastSeen  time.Time
}

func NewPersonFile(personID UUID, fileID UUID) *PersonFile {
	return &PersonFile{
		PersonID: personID,
		FileID:   fileID,
	}
}

func (p *PersonFile) SeenAt(ts ...time.Time) {
	empty := time.Time{}

	for _, t := range ts {
		t = t.UTC().Round(time.Second)

		if p.FirstSeen == empty || t.Before(p.FirstSeen) {
			p.FirstSeen = t
		}
		if p.LastSeen == empty || t.After(p.LastSeen) {
			p.LastSeen = t
		}
	}
}
