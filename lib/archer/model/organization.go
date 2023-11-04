package model

import (
	"github.com/samber/lo"
)

type Organization struct {
	Name string
	ID   UUID

	groupsByName map[string]*Group
	Size         *Size
	Blame        *Size
	Changes      *Changes
	Metrics      *Metrics
	Data         map[string]string
}

func NewOrganization(name string, id *UUID) *Organization {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("o")
	} else {
		uuid = *id
	}

	return &Organization{
		Name:         name,
		ID:           uuid,
		groupsByName: map[string]*Group{},
		Size:         NewSize(),
		Blame:        NewSize(),
		Changes:      NewChanges(),
		Metrics:      NewMetrics(),
		Data:         map[string]string{},
	}
}

func (o *Organization) GetOrCreateGroup(name string) *Group {
	return o.GetOrCreateGroupEx(name, nil)

}

func (o *Organization) GetOrCreateGroupEx(name string, id *UUID) *Group {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := o.groupsByName[name]

	if !ok {
		result = NewGroup(name, id)
		o.groupsByName[name] = result
	}

	return result
}

func (o *Organization) ListGroups() []*Group {
	return lo.Values(o.groupsByName)
}
