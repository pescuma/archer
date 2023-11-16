package server

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) filterPeople(col []*model.Person, search string) []*model.Person {
	if search != "" {
		search = strings.ToLower(search)
	}

	return lo.Filter(col, func(p *model.Person, index int) bool {
		if !s.filterPerson(p, search) {
			return false
		}

		return true
	})
}

func (s *server) filterPerson(p *model.Person, search string) bool {
	if search != "" {
		hasName := lo.ContainsBy(p.ListNames(), func(j string) bool {
			return strings.Contains(strings.ToLower(j), search)
		})
		hasEmail := lo.ContainsBy(p.ListEmails(), func(j string) bool {
			return strings.Contains(strings.ToLower(j), search)
		})
		if !hasName && !hasEmail {
			return false
		}
	}

	return true
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
		return sortBy(col, func(r *model.Person) int { return r.Changes.ModifiedLines }, *asc)
	case "changes.addedLines":
		return sortBy(col, func(r *model.Person) int { return r.Changes.AddedLines }, *asc)
	case "changes.deletedLines":
		return sortBy(col, func(r *model.Person) int { return r.Changes.DeletedLines }, *asc)
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
		"firstSeen": p.FirstSeen,
		"lastSeen":  p.LastSeen,
	}
}

func (s *server) toPersonReference(p *model.Person) gin.H {
	return gin.H{
		"id":     p.ID,
		"name":   p.Name,
		"emails": p.ListEmails(),
	}
}
