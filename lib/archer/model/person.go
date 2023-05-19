package model

import (
	"github.com/samber/lo"
)

type Person struct {
	Name string
	ID   UUID

	emails map[string]bool
	Data   map[string]string
}

func NewPerson(name string) *Person {
	return &Person{
		Name:   name,
		ID:     NewUUID("a"),
		emails: map[string]bool{},
		Data:   map[string]string{},
	}
}

func (p *Person) AddEmail(email string) {
	p.emails[email] = true
}

func (p *Person) ListEmails() []string {
	return lo.Keys(p.emails)
}
