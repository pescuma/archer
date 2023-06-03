package model

import (
	"time"
)

type TeamMember struct {
	Team   *Team
	Person *Person
	Start  *time.Time
	End    *time.Time
	ID     UUID
}

func NewTeamMember(team *Team, person *Person, start *time.Time, end *time.Time, id *UUID) *TeamMember {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("n")
	} else {
		uuid = *id
	}

	return &TeamMember{
		Team:   team,
		Person: person,
		Start:  start,
		End:    end,
		ID:     uuid,
	}
}
