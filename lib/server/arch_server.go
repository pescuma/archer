package server

import (
	"github.com/gin-gonic/gin"

	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/model"
)

func (s *server) initArch(r *gin.Engine) {
	r.GET("/api/arch", getP[Filters](s.archList))
}

func (s *server) archList(params *Filters) (any, error) {
	filter, err := s.createProjectAndDepsFilter(params)
	if err != nil {
		return nil, err
	}

	var result []gin.H

	for _, proj := range filters.FilterProjects(filter, s.projects.ListProjects(model.FilterExcludeExternal)) {
		result = append(result, s.toProject(proj))

		for _, dep := range filters.FilterDependencies(filter, proj.Dependencies) {
			if !filter.Decide(filter.FilterDependency(dep)) {
				continue
			}

			result = append(result, s.toDependency(dep))
		}
	}

	return result, nil
}
