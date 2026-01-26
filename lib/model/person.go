package model

import (
	"sort"

	"github.com/samber/lo"
)

type Person struct {
	Name string
	ID   ID

	names   map[string]bool
	emails  map[string]bool
	Blame   *Blame
	Changes *Changes
	SeenAt  *SeenAt
	Data    map[string]string
}

func NewPerson(id ID) *Person {
	return &Person{
		ID:      id,
		names:   map[string]bool{},
		emails:  map[string]bool{},
		Blame:   NewBlame(),
		Changes: NewChanges(),
		SeenAt:  NewSeenAt(),
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
