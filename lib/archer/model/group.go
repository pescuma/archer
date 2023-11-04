package model

import (
	"github.com/samber/lo"
)

type Group struct {
	Name string
	ID   UUID

	teamsByName map[string]*Team
	Size        *Size
	Blame       *Size
	Changes     *Changes
	Metrics     *Metrics
	Data        map[string]string
}

func NewGroup(name string, id *UUID) *Group {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("g")
	} else {
		uuid = *id
	}

	return &Group{
		Name:        name,
		ID:          uuid,
		teamsByName: map[string]*Team{},
		Size:        NewSize(),
		Blame:       NewSize(),
		Changes:     NewChanges(),
		Metrics:     NewMetrics(),
		Data:        map[string]string{},
	}
}

func (g *Group) GetOrCreateTeam(name string) *Team {
	return g.GetOrCreateTeamEx(name, nil)

}

func (g *Group) GetOrCreateTeamEx(name string, id *UUID) *Team {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := g.teamsByName[name]

	if !ok {
		result = NewTeam(name, id)
		g.teamsByName[name] = result
	}

	return result
}

func (g *Group) ListTeams() []*Team {
	return lo.Values(g.teamsByName)
}
