package git

import (
	"fmt"
	"strings"

	"github.com/pescuma/archer/lib/utils"

	"github.com/hashicorp/go-set/v2"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/model"
)

type nameEmailGrouper struct {
	peopleDB      *model.People
	auto          bool
	ignoredEmails map[string]bool

	byOne   map[string]*namesEmails
	byBoth  map[string]*namesEmails
	removed *set.Set[*namesEmails]
}

func newNameEmailGrouperFrom(configDB *map[string]string, peopleDB *model.People) *nameEmailGrouper {
	grouper := newNameEmailGrouper(configDB, peopleDB)

	for _, p := range peopleDB.ListPeople() {
		emails := p.ListEmails()
		if len(emails) == 0 {
			continue
		}

		r := newNamesEmails()
		r.Person = p
		r.Name = p.Name

		for _, n := range p.ListNames() {
			r.Names.Insert(n)
		}
		for _, e := range p.ListEmails() {
			r.Emails.Insert(e)
		}
		r.people.Insert(p)

		grouper.store(r)
	}

	return grouper
}

func newNameEmailGrouper(configDB *map[string]string, peopleDB *model.People) *nameEmailGrouper {
	result := &nameEmailGrouper{
		peopleDB:      peopleDB,
		ignoredEmails: map[string]bool{},
		byOne:         map[string]*namesEmails{},
		byBoth:        map[string]*namesEmails{},
		removed:       set.New[*namesEmails](10),
	}

	ignoreEmails := strings.Split((*configDB)["people:grouper:ignore-emails"], ",")
	for _, e := range ignoreEmails {
		e = strings.TrimSpace(e)
		if e != "" {
			result.addIgnoredEmail(e)
		}
	}

	result.auto = utils.ToBool((*configDB)["people:grouper:auto"], true)

	return result
}

func (g *nameEmailGrouper) store(r *namesEmails) {
	for _, n := range r.Names.Slice() {
		g.byOne[g.keyOne(n)] = r
	}
	for _, e := range r.Emails.Slice() {
		e = g.keyOne(e)
		if !g.ignoredEmails[e] {
			g.byOne[e] = r
		}
	}
	for _, n := range r.Names.Slice() {
		for _, e := range r.Emails.Slice() {
			g.byBoth[g.keyBoth(n, e)] = r
		}
	}
}

func (g *nameEmailGrouper) addIgnoredEmail(email string) {
	g.ignoredEmails[g.keyOne(email)] = true
}

func (g *nameEmailGrouper) add(name string, email string) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	var r *namesEmails

	isGithubEmail := strings.HasSuffix(email, "@users.noreply.github.com")

	if !g.auto && !isGithubEmail && name != "" {
		r = g.byBoth[g.keyBoth(name, email)]

	} else {
		var n, e *namesEmails

		if name != "" {
			n = g.byOne[g.keyOne(name)]
		}
		if !g.ignoredEmails[g.keyOne(email)] {
			e = g.byOne[g.keyOne(email)]
		}

		if n == nil && e == nil {
			r = g.byBoth[g.keyBoth(name, email)]

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
				g.removed.Insert(e)
			}
		}
	}

	if r == nil {
		r = newNamesEmails()
	}

	if name != "" {
		r.Names.Insert(name)
	}
	r.Emails.Insert(email)

	g.store(r)
}

func (g *nameEmailGrouper) copyToPeopleDB() {
	nes := g.list()

	for _, ne := range nes {
		g.removeNamesThatAreEmails(ne, ne.Names.Slice())
		ne.Name = g.findBestName(ne.Names.Slice())
		if ne.Person == nil {
			ne.Person = g.findBestPerson(ne.people.Slice(), ne.Name)
		}

		ne.Person.Name = ne.Name
		for _, n := range ne.Names.Slice() {
			ne.Person.AddName(n)
		}
		for _, e := range ne.Emails.Slice() {
			ne.Person.AddEmail(e)
		}
	}
}

func (g *nameEmailGrouper) removeNamesThatAreEmails(ne *namesEmails, names []string) {
	es := lo.Filter(names, func(i string, index int) bool { return utils.IsEmail(i) })
	if len(names) > len(es) {
		for _, name := range es {
			ne.Names.Remove(name)
		}
	}
}

func (g *nameEmailGrouper) findBestName(names []string) string {
	return lo.MaxBy(names, func(a string, b string) bool {
		ignoreA := utils.IsEmail(a)
		ignoreB := utils.IsEmail(b)
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

func (g *nameEmailGrouper) findBestPerson(people []*model.Person, name string) *model.Person {
	if len(people) == 0 {
		return g.peopleDB.GetOrCreatePerson(nil)
	}

	sameName := lo.Filter(people, func(p *model.Person, _ int) bool { return p.Name == name })
	if len(sameName) > 0 {
		return sameName[0]
	} else {
		return people[0]
	}
}

func (g *nameEmailGrouper) getPerson(name string, email string) *model.Person {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if !g.auto {
		key := g.keyBoth(name, email)

		r := g.byBoth[key]
		if r == nil {
			panic(fmt.Sprintf("Could not find person for %s <%s>", name, email))
		}

		return r.Person

	} else if !g.ignoredEmails[g.keyOne(email)] {
		e := g.byOne[g.keyOne(email)]
		if e == nil {
			panic(fmt.Sprintf("Could not find person for %s <%s>", name, email))
		}

		return e.Person
	} else {
		n := g.byOne[g.keyOne(name)]
		if n == nil {
			panic(fmt.Sprintf("Could not find person for %s <%s>", name, email))
		}

		return n.Person
	}
}

func (g *nameEmailGrouper) list() []*namesEmails {
	result := map[*namesEmails]bool{}

	for _, v := range g.byOne {
		result[v] = true
	}
	for _, v := range g.byBoth {
		result[v] = true
	}

	return lo.Keys(result)
}

func (g *nameEmailGrouper) keyOne(n string) string {
	return strings.TrimSpace(utils.ToLowerNoAccents(n))
}

func (g *nameEmailGrouper) keyBoth(n string, e string) string {
	return g.keyOne(n) + "\n" + g.keyOne(e)
}

type namesEmails struct {
	Person *model.Person
	Name   string

	Names  *set.Set[string]
	Emails *set.Set[string]
	people *set.Set[*model.Person]
}

func newNamesEmails() *namesEmails {
	return &namesEmails{
		Names:  set.New[string](10),
		Emails: set.New[string](10),
		people: set.New[*model.Person](10),
	}
}
