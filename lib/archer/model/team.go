package model

import (
	"sort"
	"time"
)

type Team struct {
	Name string
	ID   UUID

	members map[UUID][]*TeamMember
	areas   map[UUID][]*TeamArea

	Size    *Size
	Changes *Changes
	Metrics *Metrics
	Data    map[string]string
}

func NewTeam(name string, id *UUID) *Team {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("t")
	} else {
		uuid = *id
	}

	return &Team{
		Name:    name,
		ID:      uuid,
		members: map[UUID][]*TeamMember{},
		areas:   map[UUID][]*TeamArea{},
		Size:    NewSize(),
		Changes: NewChanges(),
		Metrics: NewMetrics(),
		Data:    map[string]string{},
	}
}

func (t *Team) AddMember(person *Person, start, end *time.Time) {
	t.AddMemberEx(person, start, end, nil)
}

func (t *Team) AddMemberEx(person *Person, start, end *time.Time, id *UUID) {
	result := append(t.members[person.ID], NewTeamMember(t, person, start, end, id))

	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]

		if a.Start == nil {
			return true
		} else if b.Start == nil {
			return false
		} else {
			return a.Start.Before(*b.Start)
		}
	})

	last := 0
	for i := 1; i < len(result); i++ {
		a := result[i-1]
		b := result[i]

		if a.End != nil && b.Start != nil && a.End.Before(*b.Start) {
			last++
			continue
		}

		// Fix first, ignore second

		if a.Start == nil || b.Start == nil {
			a.Start = nil
		} else if b.Start.Before(*a.Start) {
			a.Start = b.Start
		}

		if a.End == nil || b.End == nil {
			a.End = nil
		} else if b.End.After(*a.End) {
			a.End = b.End
		}
	}

	if last < len(result)-1 {
		result = result[:last+1]
	}

	t.members[person.ID] = result
}

func (t *Team) AddArea(area *ProductArea, start, end *time.Time) {
	t.AddAreaEx(area, start, end, nil)
}

func (t *Team) AddAreaEx(area *ProductArea, start, end *time.Time, id *UUID) {
	result := append(t.areas[area.ID], NewTeamArea(t, area, start, end, id))

	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]

		if a.Start == nil {
			return true
		} else if b.Start == nil {
			return false
		} else {
			return a.Start.Before(*b.Start)
		}
	})

	last := 0
	for i := 1; i < len(result); i++ {
		a := result[i-1]
		b := result[i]

		if a.End != nil && b.Start != nil && a.End.Before(*b.Start) {
			last++
			continue
		}

		// Fix first, ignore second

		if a.Start == nil || b.Start == nil {
			a.Start = nil
		} else if b.Start.Before(*a.Start) {
			a.Start = b.Start
		}

		if a.End == nil || b.End == nil {
			a.End = nil
		} else if b.End.After(*a.End) {
			a.End = b.End
		}
	}

	if last < len(result)-1 {
		result = result[:last+1]
	}

	t.areas[area.ID] = result
}
