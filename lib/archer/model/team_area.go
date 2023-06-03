package model

import (
	"time"
)

type TeamArea struct {
	Team  *Team
	Area  *ProductArea
	Start *time.Time
	End   *time.Time
	ID    UUID
}

func NewTeamArea(team *Team, area *ProductArea, start *time.Time, end *time.Time, id *UUID) *TeamArea {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("b")
	} else {
		uuid = *id
	}

	return &TeamArea{
		Team:  team,
		Area:  area,
		Start: start,
		End:   end,
		ID:    uuid,
	}
}
