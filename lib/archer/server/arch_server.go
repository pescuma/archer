package server

import (
	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/filters"
	"github.com/pescuma/archer/lib/archer/model"
)

func (s *server) initArch(r *gin.Engine) {
	r.GET("/api/arch", getP[Filters](s.archList))
}

func (s *server) archList(params *Filters) (any, error) {
	filter, err := s.createProjectAndDepsFilter(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	result := []gin.H{}

	for _, proj := range filters.FilterProjects(filter, s.projects.ListProjects(model.FilterExcludeExternal)) {
		result = append(result, s.toProject(proj))

		for _, dep := range filters.FilterDependencies(filter, proj.Dependencies) {
			if filter.Decide(filter.FilterDependency(dep)) == filters.Exclude {
				continue
			}

			result = append(result, s.toDependency(dep))
		}
	}

	return result, nil
}
