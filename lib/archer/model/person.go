package model

import (
	"sort"

	"github.com/samber/lo"
)

type Person struct {
	Name string
	ID   UUID
	Team *Team

	names   map[string]bool
	emails  map[string]bool
	Size    *Size
	Metrics *Metrics
	Data    map[string]string
}

func NewPerson(name string, id *UUID) *Person {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("a")
	} else {
		uuid = *id
	}

	return &Person{
		Name:    name,
		ID:      uuid,
		names:   map[string]bool{},
		emails:  map[string]bool{},
		Size:    NewSize(),
		Metrics: NewMetrics(),
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
