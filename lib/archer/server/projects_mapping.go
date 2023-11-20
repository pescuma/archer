package server

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-set/v2"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
	"github.com/samber/lo"
)

func (s *server) listProjects(file string, proj string, repo string, person string) ([]*model.Project, error) {
	return s.filterProjects(proj, s.projects.ListProjects(model.FilterExcludeExternal), file, repo, person)
}

func (s *server) filterProjects(proj string, col []*model.Project, file string, repo string, person string) ([]*model.Project, error) {
	file = prepareToSearch(file)
	proj = prepareToSearch(proj)
	repo = prepareToSearch(repo)
	person = prepareToSearch(person)

	var ids *set.Set[model.UUID]
	if file != "" || repo != "" || person != "" {
		r, err := s.storage.QueryProjects(file, proj, repo, person)
		if err != nil {
			return nil, err
		}
		ids = set.From(r)
	}

	return lo.Filter(col, func(i *model.Project, index int) bool {
		if proj != "" && !strings.Contains(strings.ToLower(i.Name), proj) {
			return false
		}

		if ids != nil && !ids.Contains(i.ID) {
			return false
		}

		return true
	}), nil
}

func (s *server) sortProjects(col []*model.Project, field string, asc *bool) error {
	if field == "" {
		field = "path"
	}
	if asc == nil {
		asc = new(bool)
		*asc = utils.In(field, "root", "name", "type", "rootDir", "projectFile")
	}

	switch field {
	case "root":
		return sortBy(col, func(r *model.Project) string { return r.Root }, *asc)
	case "name":
		return sortBy(col, func(r *model.Project) string { return r.Name }, *asc)
	case "type":
		return sortBy(col, func(r *model.Project) string { return r.Type.String() }, *asc)
	case "rootDir":
		return sortBy(col, func(r *model.Project) string { return r.RootDir }, *asc)
	case "projectFile":
		return sortBy(col, func(r *model.Project) string { return r.ProjectFile }, *asc)
	case "repo.name":
		return sortBy(col, func(r *model.Project) string {
			if r.RepositoryID == nil {
				return ""
			} else {
				return s.repos.GetByID(*r.RepositoryID).Name
			}
		}, *asc)
	case "changes.total":
		return sortBy(col, func(r *model.Project) int { return r.Changes.Total }, *asc)
	case "changes.in6Months":
		return sortBy(col, func(r *model.Project) int { return r.Changes.In6Months }, *asc)
	case "metrics.guiceDependencies":
		return sortBy(col, func(r *model.Project) int { return r.Metrics.GuiceDependencies }, *asc)
	case "metrics.abstracts":
		return sortBy(col, func(r *model.Project) int { return r.Metrics.Abstracts }, *asc)
	case "metrics.cyclomaticComplexity":
		return sortBy(col, func(r *model.Project) int { return r.Metrics.CyclomaticComplexity }, *asc)
	case "metrics.cognitiveComplexity":
		return sortBy(col, func(r *model.Project) int { return r.Metrics.CognitiveComplexity }, *asc)
	case "metrics.focusedComplexity":
		return sortBy(col, func(r *model.Project) int { return r.Metrics.FocusedComplexity }, *asc)
	case "firstSeen":
		return sortBy(col, func(r *model.Project) int64 { return r.FirstSeen.UnixMilli() }, *asc)
	case "lastSeen":
		return sortBy(col, func(r *model.Project) int64 { return r.LastSeen.UnixMilli() }, *asc)
	default:
		return fmt.Errorf("unknown sort field: %s", field)
	}
}

func (s *server) toProject(p *model.Project) gin.H {
	return gin.H{
		"id":          p.ID,
		"root":        p.Root,
		"name":        p.Name,
		"nameParts":   p.NameParts,
		"type":        p.Type.String(),
		"rootDir":     p.RootDir,
		"projectFile": p.ProjectFile,
		"repo":        s.toRepoReference(p.RepositoryID),
		"sizes": lo.MapValues(p.Sizes, func(value *model.Size, key string) gin.H {
			return s.toSize(value)
		}),
		"size":      s.toSize(p.Size),
		"changes":   s.toChanges(p.Changes),
		"metrics":   s.toMetrics(p.Metrics),
		"firstSeen": encodeDate(p.FirstSeen),
		"lastSeen":  encodeDate(p.LastSeen),
	}
}

func (s *server) toProjectReference(id *model.UUID) gin.H {
	if id == nil {
		return nil
	}

	p := s.projects.GetByID(*id)

	return gin.H{
		"id":   p.ID,
		"root": p.Root,
		"name": p.Name,
		"type": p.String(),
	}
}
