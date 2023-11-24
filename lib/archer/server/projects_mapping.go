package server

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
	"github.com/samber/lo"
)

func (s *server) createProjectFilter(proj string) (map[model.UUID]bool, error) {
	proj = prepareToSearch(proj)
	if proj == "" {
		return nil, nil
	}

	projects, err := s.listProjects("", proj, "", "")
	if err != nil {
		return nil, err
	}

	result := make(map[model.UUID]bool, len(projects))
	for _, p := range projects {
		result[p.ID] = true
	}
	return result, nil
}

func (s *server) listProjects(file string, proj string, repo string, person string) ([]*model.Project, error) {
	return s.filterProjects(s.projects.ListProjects(model.FilterExcludeExternal), file, proj, repo, person)
}

func (s *server) filterProjects(col []*model.Project, file string, proj string, repo string, person string) ([]*model.Project, error) {
	proj = prepareToSearch(proj)

	fileIDs, err := s.createFileFilter(file)
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

	filesPerProject := make(map[model.UUID]map[model.UUID]bool)
	for _, f := range s.files.ListFiles() {
		if f.ProjectID == nil {
			continue
		}

		projID := *f.ProjectID

		fs, ok := filesPerProject[projID]
		if !ok {
			fs = make(map[model.UUID]bool)
			filesPerProject[projID] = fs
		}

		fs[f.ID] = true
	}

	return lo.Filter(col, func(i *model.Project, index int) bool {
		if proj != "" && !strings.Contains(strings.ToLower(i.Name), proj) {
			return false
		}

		if fileIDs != nil {
			fs, ok := filesPerProject[i.ID]
			if !ok {
				return false
			}

			if !utils.MapKeysHaveIntersection(fs, fileIDs) {
				return false
			}
		}

		if repoIDs != nil && (i.RepositoryID == nil || !repoIDs[*i.RepositoryID]) {
			return false
		}

		if personIDs != nil {
			files, ok := filesPerProject[i.ID]
			if !ok {
				return false
			}

			found := false
			for fileID := range files {
				people := s.peopleRelations.ListPeopleByFile(fileID)
				if utils.MapKeysHaveIntersection(people, personIDs) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
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

func (s *server) toDependency(p *model.ProjectDependency) gin.H {
	return gin.H{
		"id":     p.ID,
		"source": p.Source.ID,
		"target": p.Target.ID,
	}
}
