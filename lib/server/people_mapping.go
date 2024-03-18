package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func (s *server) listPeople(params *Filters) ([]*model.Person, error) {
	return s.filterPeople(s.people.ListPeople(), params)
}

func (s *server) filterPeople(col []*model.Person, params *Filters) ([]*model.Person, error) {
	personFilter, err := s.createPersonFilter(params.FilterPerson, params.FilterPersonID)
	if err != nil {
		return nil, err
	}

	fileIDs, err := s.listFileIDsOrNil(params.FilterFile)
	if err != nil {
		return nil, err
	}
	projIDs, err := s.listProjectIDsOrNil(params.FilterProject)
	if err != nil {
		return nil, err
	}
	repoIDs, err := s.listRepoIDsOrNil(params.FilterRepo)
	if err != nil {
		return nil, err
	}

	return lo.Filter(col, func(i *model.Person, index int) bool {
		if !personFilter(i) {
			return false
		}

		if fileIDs != nil {
			fs := s.peopleRelations.ListFilesByPerson(i.ID)
			if !utils.MapKeysHaveIntersection(fs, fileIDs) {
				return false
			}
		}

		if projIDs != nil {
			fs := s.peopleRelations.ListFilesByPerson(i.ID)
			found := false
			for _, pf := range fs {
				f := s.files.GetFileByID(pf.FileID)
				if f != nil && f.ProjectID != nil && projIDs[*f.ProjectID] {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}

		if repoIDs != nil {
			rs := s.peopleRelations.ListReposByPerson(i.ID)
			if !utils.MapKeysHaveIntersection(rs, repoIDs) {
				return false
			}
		}

		return true
	}), nil
}

func (s *server) createPersonFilter(person string, id model.ID) (func(*model.Person) bool, error) {
	person = prepareToSearch(person)

	switch {
	case id != 0:
		return func(p *model.Person) bool {
			return p.ID == id
		}, nil

	case person != "":
		f, err := filters.ParseStringFilter(person)
		if err != nil {
			return nil, err
		}

		return func(p *model.Person) bool {
			return lo.ContainsBy(p.ListNames(), func(j string) bool {
				return f(j)
			}) ||
				lo.ContainsBy(p.ListEmails(), func(j string) bool {
					return f(j)
				})
		}, nil

	default:
		return func(_ *model.Person) bool { return true }, nil
	}
}

func (s *server) listPersonIDsOrNil(person string, id model.ID) (map[model.ID]bool, error) {
	person = prepareToSearch(person)

	switch {
	case id != 0:
		result := make(map[model.ID]bool, 1)
		result[id] = true
		return result, nil

	case person != "":
		people, err := s.listPeople(&Filters{FilterPerson: person})
		if err != nil {
			return nil, err
		}

		result := make(map[model.ID]bool, len(people))
		for _, p := range people {
			result[p.ID] = true
		}
		return result, nil

	default:
		return nil, nil
	}
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
	case "blame.total":
		return sortBy(col, func(r *model.Person) int { return r.Blame.Total() }, *asc)
	case "changes.total":
		return sortBy(col, func(r *model.Person) int { return r.Changes.Total }, *asc)
	case "changes.in6Months":
		return sortBy(col, func(r *model.Person) int { return r.Changes.In6Months }, *asc)
	case "changes.linesModified":
		return sortBy(col, func(r *model.Person) int { return r.Changes.LinesModified }, *asc)
	case "changes.linesAdded":
		return sortBy(col, func(r *model.Person) int { return r.Changes.LinesAdded }, *asc)
	case "changes.linesDeleted":
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
		"blame":     s.toBlame(p.Blame),
		"changes":   s.toChanges(p.Changes),
		"firstSeen": encodeDate(p.FirstSeen),
		"lastSeen":  encodeDate(p.LastSeen),
	}
}

func (s *server) toPersonReference(id *model.ID) gin.H {
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

func (s *server) toProductAreaReference(id *model.ID) gin.H {
	if id == nil {
		return nil
	}

	p := s.people.GetProductAreaByID(*id)

	return gin.H{
		"id":   p.ID,
		"name": p.Name,
	}
}
