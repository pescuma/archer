package git

import (
	"sort"
	"strings"

	"github.com/hashicorp/go-set/v2"
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

	grouper.prepare()

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

	var r *namesEmails
	n := g.byName[name]
	e := g.byEmail[email]

	if n == nil && e == nil {
		r = &namesEmails{
			Names:  set.New[string](10),
			Emails: set.New[string](10),
			people: set.New[*model.Person](10),
		}

	} else if n == nil && e != nil {
		r = e

	} else if n != nil && e == nil {
		r = n

	} else {
		r = n

		if n != e {
			n.people.InsertSet(e.people)
			n.Names.InsertSet(e.Names)
			n.Emails.InsertSet(e.Emails)
		}
	}

	r.Names.Insert(name)
	g.byName[name] = r

	r.Emails.Insert(email)
	g.byEmail[email] = r

	if person != nil {
		r.people.Insert(person)
	}
}

func (g *nameEmailGrouper) prepare() {
	nes := g.list()

	for _, ne := range nes {
		names := ne.Names.Slice()
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
	Name   string
	Names  *set.Set[string]
	Emails *set.Set[string]
	people *set.Set[*model.Person]
}
