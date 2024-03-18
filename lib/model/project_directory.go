package model

import "time"

type ProjectDirectory struct {
	RelativePath string
	Type         ProjectDirectoryType
	ID           ID

	Size      *Size
	Changes   *Changes
	Metrics   *Metrics
	Data      map[string]string
	FirstSeen time.Time
	LastSeen  time.Time
}

func NewProjectDirectory(id ID, relativePath string) *ProjectDirectory {
	return &ProjectDirectory{
		RelativePath: relativePath,
		ID:           id,
		Size:         NewSize(),
		Changes:      NewChanges(),
		Metrics:      NewMetrics(),
		Data:         map[string]string{},
	}
}

func (d *ProjectDirectory) SeenAt(ts ...time.Time) {
	empty := time.Time{}

	for _, t := range ts {
		t = t.UTC().Round(time.Second)

		if d.FirstSeen == empty || t.Before(d.FirstSeen) {
			d.FirstSeen = t
		}
		if d.LastSeen == empty || t.After(d.LastSeen) {
			d.LastSeen = t
		}
	}
}
