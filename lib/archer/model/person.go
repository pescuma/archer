package model

import (
	"sort"

	"github.com/samber/lo"
)

type Person struct {
	Name string
	ID   UUID

	names  map[string]bool
	emails map[string]bool
	Data   map[string]string
}

func NewPerson(name string) *Person {
	return &Person{
		Name:   name,
		ID:     NewUUID("a"),
		names:  map[string]bool{},
		emails: map[string]bool{},
		Data:   map[string]string{},
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
