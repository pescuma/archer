package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/filters"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
	"github.com/samber/lo"
)

func (s *server) createFileFilter(file string) (map[model.UUID]bool, error) {
	file = prepareToSearch(file)
	if file == "" {
		return nil, nil
	}

	files, err := s.listFiles(file, "", "", "")
	if err != nil {
		return nil, err
	}

	result := make(map[model.UUID]bool, len(files))
	for _, p := range files {
		result[p.ID] = true
	}
	return result, nil
}

func (s *server) listFiles(file string, proj string, repo string, person string) ([]*model.File, error) {
	return s.filterFiles(s.files.ListFiles(), file, proj, repo, person)
}

func (s *server) filterFiles(col []*model.File, file string, proj string, repo string, person string) ([]*model.File, error) {
	fileFilter, err := filters.ParseStringFilter(file)
	if err != nil {
		return nil, err
	}

	projIDs, err := s.createProjectFilter(proj)
	if err != nil {
		return nil, err
	}
	repoIDs, err := s.createRepoFilter(repo)
	if err != nil {
		return nil, err
	}
	personIDs, err := s.createPersonFilter(person)
	if err != nil {
		return nil, err
	}

	return lo.Filter(col, func(i *model.File, index int) bool {
		if !fileFilter(i.Path) {
			return false
		}

		if projIDs != nil && (i.ProjectID == nil || !projIDs[*i.ProjectID]) {
			return false
		}

		if repoIDs != nil && (i.RepositoryID == nil || !repoIDs[*i.RepositoryID]) {
			return false
		}

		if personIDs != nil {
			ps := s.peopleRelations.ListPeopleByFile(i.ID)
			if !utils.MapKeysHaveIntersection(ps, personIDs) {
				return false
			}
		}

		return true
	}), nil
}

func (s *server) sortFiles(col []*model.File, field string, asc *bool) error {
	if field == "" {
		field = "path"
	}
	if asc == nil {
		asc = new(bool)
		*asc = utils.In(field, "path", "repo.name")
	}

	switch field {
	case "path":
		return sortBy(col, func(r *model.File) string { return r.Path }, *asc)
	case "project.name":
		return sortBy(col, func(r *model.File) string {
			if r.ProjectID == nil {
				return ""
			} else {
				return s.projects.GetByID(*r.ProjectID).Name
			}
		}, *asc)
	case "repo.name":
		return sortBy(col, func(r *model.File) string {
			if r.RepositoryID == nil {
				return ""
			} else {
				return s.repos.GetByID(*r.RepositoryID).Name
			}
		}, *asc)
	case "exists":
		return sortBy(col, func(r *model.File) string { return utils.IIf(r.Exists, "1", "0") }, *asc)
	case "size.lines":
		return sortBy(col, func(r *model.File) int { return r.Size.Lines }, *asc)
	case "size.files":
		return sortBy(col, func(r *model.File) int { return r.Size.Files }, *asc)
	case "size.bytes":
		return sortBy(col, func(r *model.File) int { return r.Size.Bytes }, *asc)
	case "changes.total":
		return sortBy(col, func(r *model.File) int { return r.Changes.Total }, *asc)
	case "changes.in6Months":
		return sortBy(col, func(r *model.File) int { return r.Changes.In6Months }, *asc)
	case "metrics.guiceDependencies":
		return sortBy(col, func(r *model.File) int { return r.Metrics.GuiceDependencies }, *asc)
	case "metrics.abstracts":
		return sortBy(col, func(r *model.File) int { return r.Metrics.Abstracts }, *asc)
	case "metrics.cyclomaticComplexity":
		return sortBy(col, func(r *model.File) int { return r.Metrics.CyclomaticComplexity }, *asc)
	case "metrics.cognitiveComplexity":
		return sortBy(col, func(r *model.File) int { return r.Metrics.CognitiveComplexity }, *asc)
	case "metrics.focusedComplexity":
		return sortBy(col, func(r *model.File) int { return r.Metrics.FocusedComplexity }, *asc)
	case "firstSeen":
		return sortBy(col, func(r *model.File) int64 { return r.FirstSeen.UnixMilli() }, *asc)
	case "lastSeen":
		return sortBy(col, func(r *model.File) int64 { return r.LastSeen.UnixMilli() }, *asc)
	default:
		return fmt.Errorf("unknown sort field: %s", field)
	}
}

func (s *server) toFile(f *model.File) gin.H {
	return gin.H{
		"id":        f.ID,
		"path":      f.Path,
		"project":   s.toProjectReference(f.ProjectID),
		"dir":       f.ProjectDirectoryID,
		"area":      s.toProductAreaReference(f.ProductAreaID),
		"repo":      s.toRepoReference(f.RepositoryID),
		"exists":    f.Exists,
		"size":      s.toSize(f.Size),
		"changes":   s.toChanges(f.Changes),
		"metrics":   s.toMetrics(f.Metrics),
		"firstSeen": encodeDate(f.FirstSeen),
		"lastSeen":  encodeDate(f.LastSeen),
	}
}
