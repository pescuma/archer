package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func (s *server) listProjects(params *Filters) ([]*model.Project, error) {
	return s.filterProjects(s.projects.ListProjects(model.FilterExcludeExternal), params)
}

func (s *server) filterProjects(col []*model.Project, params *Filters) ([]*model.Project, error) {
	filter, err := s.createProjectAndDepsFilter(params)
	if err != nil {
		return nil, err
	}

	externalFilter, err := s.createProjectsExternalFilters(params)
	if err != nil {
		return nil, err
	}

	projs := filters.FilterProjects(filter, col)

	var result []*model.Project

	if externalFilter == nil {
		result = projs

	} else {
		result = lo.Filter(projs, func(proj *model.Project, index int) bool {
			return externalFilter(proj)
		})
	}

	return result, nil
}

func (s *server) createProjectAndDepsFilter(params *Filters) (filters.ProjectFilter, error) {
	var fs []filters.ProjectFilterWithUsage

	fi, err := filters.ParseProjectFilterWithUsage(s.projects, params.FilterProject, filters.Include)
	if err != nil {
		return nil, err
	}

	fs = append(fs, fi)
	fs = append(fs, filters.CreateIgnoreExternalDependenciesFilter())

	return filters.UnliftProjectFilter(filters.GroupProjectFilters(fs...)), nil
}

func (s *server) createProjectsExternalFilters(params *Filters) (func(*model.Project) bool, error) {
	fileIDs, err := s.listFileIDsOrNil(params.FilterFile)
	if err != nil {
		return nil, err
	}
	repoIDs, err := s.listRepoIDsOrNil(params.FilterRepo)
	if err != nil {
		return nil, err
	}
	personIDs, err := s.listPersonIDsOrNil(params.FilterPerson, params.FilterPersonID)
	if err != nil {
		return nil, err
	}

	if fileIDs == nil && repoIDs == nil && personIDs == nil {
		return nil, nil
	}

	filesPerProject := make(map[model.ID]map[model.ID]bool)
	if fileIDs != nil || personIDs != nil {
		for _, f := range s.files.List() {
			if f.ProjectID == nil {
				continue
			}

			projID := *f.ProjectID

			fs, ok := filesPerProject[projID]
			if !ok {
				fs = make(map[model.ID]bool)
				filesPerProject[projID] = fs
			}

			fs[f.ID] = true
		}
	}

	result := func(i *model.Project) bool {
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
	}

	return result, nil
}

func (s *server) listProjectIDsOrNil(proj string) (map[model.ID]bool, error) {
	proj = prepareToSearch(proj)

	switch {
	case proj != "":
		projects, err := s.listProjects(&Filters{FilterProject: proj})
		if err != nil {
			return nil, err
		}

		result := make(map[model.ID]bool, len(projects))
		for _, p := range projects {
			result[p.ID] = true
		}
		return result, nil

	default:
		return nil, nil
	}
}

func (s *server) sortProjects(col []*model.Project, field string, asc *bool) error {
	if field == "" {
		field = "path"
	}
	if asc == nil {
		asc = new(bool)
		*asc = utils.In(field, "name", "type", "rootDir", "projectFile")
	}

	switch field {
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
		"name":        p.Name,
		"groups":      p.Groups,
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

func (s *server) toProjectReference(id *model.ID) gin.H {
	if id == nil {
		return nil
	}

	p := s.projects.GetByID(*id)

	return gin.H{
		"id":   p.ID,
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
