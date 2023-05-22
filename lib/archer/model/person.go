package model

import (
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
	return lo.Keys(p.names)
}

func (p *Person) AddEmail(email string) {
	p.emails[email] = true
}

func (p *Person) ListEmails() []string {
	return lo.Keys(p.emails)
}
