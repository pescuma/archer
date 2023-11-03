package git

import (
	"sort"
	"strings"

	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/archer/model"
)

type nameEmailGrouper struct {
	byName  map[string]*namesEmails
	byEmail map[string]*namesEmails
}

func newNameEmailGrouperFrom(peopleDB *model.People) *nameEmailGrouper {
	grouper := newNameEmailGrouper()

	for _, p := range peopleDB.ListPeople() {
		emails := p.ListEmails()
		if len(emails) == 0 {
			continue
		}

		for _, email := range emails {
			grouper.add(p.Name, email, p)
		}
		for _, name := range p.ListNames() {
			grouper.add(name, emails[0], p)
		}
	}

	return grouper
}

func newNameEmailGrouper() *nameEmailGrouper {
	return &nameEmailGrouper{
		byName:  map[string]*namesEmails{},
		byEmail: map[string]*namesEmails{},
	}
}

func (g *nameEmailGrouper) add(name string, email string, person *model.Person) {
	if name == "" {
		name = email
	}

	n := g.byName[name]
	e := g.byEmail[email]

	if n == nil && e == nil {
		n = &namesEmails{
			Names:  map[string]bool{},
			Emails: map[string]bool{},
		}

		n.people = append(n.people, person)

		n.Names[name] = true
		g.byName[name] = n

		n.Emails[email] = true
		g.byEmail[email] = n

	} else if n == nil && e != nil {
		e.Names[name] = true
		g.byName[name] = e

	} else if n != nil && e == nil {
		n.Emails[email] = true
		g.byEmail[email] = n

	} else {
		if n != e {
			for _, p := range e.people {
				n.people = append(n.people, p)
			}
			for k := range e.Names {
				n.Names[k] = true
				g.byName[k] = n
			}
			for k := range e.Emails {
				n.Emails[k] = true
				g.byEmail[k] = n
			}
		}
	}
}

func (g *nameEmailGrouper) prepare() {
	nes := g.list()

	for _, ne := range nes {
		names := lo.Keys(ne.Names)
		sort.Slice(names, func(i, j int) bool {
			return names[i] < names[j]
		})
		ne.Name = lo.MaxBy(names, func(a string, b string) bool {
			ignoreA := strings.Contains(a, "@")
			ignoreB := strings.Contains(b, "@")
			if ignoreA != ignoreB {
				return ignoreB
			}

			ignoreA = strings.Contains(a, "-")
			ignoreB = strings.Contains(b, "-")
			if ignoreA != ignoreB {
				return ignoreB
			}

			return len(a) > len(b)
		})
	}
}

func (g *nameEmailGrouper) getName(email string) string {
	return g.byEmail[email].Name
}

func (g *nameEmailGrouper) list() []*namesEmails {
	result := map[*namesEmails]bool{}

	for _, v := range g.byName {
		result[v] = true
	}

	return lo.Keys(result)
}

type namesEmails struct {
	people []*model.Person
	Name   string
	Names  map[string]bool
	Emails map[string]bool
}
