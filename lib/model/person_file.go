package model

import "time"

type PersonFile struct {
	PersonID ID
	FileID   ID

	FirstSeen time.Time
	LastSeen  time.Time
}

func NewPersonFile(personID ID, fileID ID) *PersonFile {
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
