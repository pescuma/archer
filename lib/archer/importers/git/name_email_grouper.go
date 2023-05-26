package git

import (
	"strings"

	"github.com/samber/lo"

	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type nameEmailGrouper struct {
	byName  map[string]*namesEmails
	byEmail map[string]*namesEmails
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

		n.person = utils.Coalesce(n.person, person)

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
		ne.Name = lo.MaxBy(lo.Keys(ne.Names), func(a string, b string) bool {
			aIsEmail := strings.Contains(a, "@")
			bIsEmail := strings.Contains(b, "@")
			if aIsEmail != bIsEmail {
				return bIsEmail
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
	person *model.Person
	Name   string
	Names  map[string]bool
	Emails map[string]bool
}
