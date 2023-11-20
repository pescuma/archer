package server

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-set/v2"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) listPeople(file string, proj string, repo string, person string) ([]*model.Person, error) {
	return s.filterPeople(s.people.ListPeople(), file, proj, repo, person)
}

func (s *server) filterPeople(col []*model.Person, file string, proj string, repo string, person string) ([]*model.Person, error) {
	file = prepareToSearch(file)
	proj = prepareToSearch(proj)
	repo = prepareToSearch(repo)
	person = prepareToSearch(person)

	var ids *set.Set[model.UUID]
	if file != "" || proj != "" || repo != "" {
		r, err := s.storage.QueryPeople(file, proj, repo, person)
		if err != nil {
			return nil, err
		}
		ids = set.From(r)
	}

	return lo.Filter(col, func(i *model.Person, index int) bool {
		if person != "" {
			hasName := lo.ContainsBy(i.ListNames(), func(j string) bool {
				return strings.Contains(strings.ToLower(j), person)
			})
			hasEmail := lo.ContainsBy(i.ListEmails(), func(j string) bool {
				return strings.Contains(strings.ToLower(j), person)
			})
			if !hasName && !hasEmail {
				return false
			}
		}

		if ids != nil && !ids.Contains(i.ID) {
			return false
		}

		return true
	}), nil
}

func (s *server) sortPeople(col []*model.Person, field string, asc *bool) error {
	if field == "" {
		field = "name"
	}
	if asc == nil {
		asc = new(bool)
		*asc = field == "name" || field == "rootDir" || field == "vcs"
	}

	switch field {
	case "name":
		return sortBy(col, func(r *model.Person) string { return r.Name }, *asc)
	case "names":
		return sortBy(col, func(r *model.Person) string { return r.ListNames()[0] }, *asc)
	case "emails":
		return sortBy(col, func(r *model.Person) string { return r.ListEmails()[0] }, *asc)
	case "blame.lines":
		return sortBy(col, func(r *model.Person) int { return r.Blame.Lines }, *asc)
	case "blame.files":
		return sortBy(col, func(r *model.Person) int { return r.Blame.Files }, *asc)
	case "blame.bytes":
		return sortBy(col, func(r *model.Person) int { return r.Blame.Bytes }, *asc)
	case "changes.total":
		return sortBy(col, func(r *model.Person) int { return r.Changes.Total }, *asc)
	case "changes.in6Months":
		return sortBy(col, func(r *model.Person) int { return r.Changes.In6Months }, *asc)
	case "changes.modifiedLines":
		return sortBy(col, func(r *model.Person) int { return r.Changes.LinesModified }, *asc)
	case "changes.addedLines":
		return sortBy(col, func(r *model.Person) int { return r.Changes.LinesAdded }, *asc)
	case "changes.deletedLines":
		return sortBy(col, func(r *model.Person) int { return r.Changes.LinesDeleted }, *asc)
	case "firstSeen":
		return sortBy(col, func(r *model.Person) int64 { return r.FirstSeen.UnixMilli() }, *asc)
	case "lastSeen":
		return sortBy(col, func(r *model.Person) int64 { return r.LastSeen.UnixMilli() }, *asc)
	default:
		return fmt.Errorf("unknown sort field: %s", field)
	}
}

func (s *server) toPerson(p *model.Person) gin.H {
	return gin.H{
		"id":        p.ID,
		"name":      p.Name,
		"names":     p.ListNames(),
		"emails":    p.ListEmails(),
		"blame":     s.toSize(p.Blame),
		"changes":   s.toChanges(p.Changes),
		"firstSeen": encodeDate(p.FirstSeen),
		"lastSeen":  encodeDate(p.LastSeen),
	}
}

func (s *server) toPersonReference(id *model.UUID) gin.H {
	if id == nil {
		return nil
	}

	p := s.people.GetPersonByID(*id)

	return gin.H{
		"id":     p.ID,
		"name":   p.Name,
		"emails": p.ListEmails(),
	}
}

func (s *server) toProductAreaReference(id *model.UUID) gin.H {
	if id == nil {
		return nil
	}

	p := s.people.GetProductAreaByID(*id)

	return gin.H{
		"id":   p.ID,
		"name": p.Name,
	}
}
