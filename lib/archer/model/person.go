package model

import (
	"sort"
	"time"

	"github.com/samber/lo"
)

type Person struct {
	Name string
	ID   UUID

	names     map[string]bool
	emails    map[string]bool
	Blame     *Size
	Changes   *Changes
	Data      map[string]string
	FirstSeen time.Time
	LastSeen  time.Time
}

func NewPerson(name string, id *UUID) *Person {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("i")
	} else {
		uuid = *id
	}

	return &Person{
		Name:    name,
		ID:      uuid,
		names:   map[string]bool{},
		emails:  map[string]bool{},
		Blame:   NewSize(),
		Changes: NewChanges(),
		Data:    map[string]string{},
	}
}

func (p *Person) AddName(name string) {
	p.names[name] = true
}

func (p *Person) ListNames() []string {
	result := lo.Keys(p.names)
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func (p *Person) AddEmail(email string) {
	p.emails[email] = true
}

func (p *Person) ListEmails() []string {
	result := lo.Keys(p.emails)
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func (p *Person) SeenAt(ts ...time.Time) {
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
