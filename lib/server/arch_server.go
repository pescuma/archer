package server

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/model"
)

func (s *server) initArch(r *gin.Engine) {
	r.GET("/api/arch", getP[Filters](s.archList))
}

func (s *server) archList(params *Filters) (any, error) {
	projs, err := s.listProjects(params)
	if err != nil {
		return nil, err
	}

	var result []gin.H

	projIDs := lo.Associate(projs, func(proj *model.Project) (model.ID, bool) {
		return proj.ID, true
	})

	for _, proj := range projs {
		result = append(result, s.toProject(proj))

		for _, dep := range proj.Dependencies {
			if projIDs[dep.Source.ID] && projIDs[dep.Target.ID] {
				result = append(result, s.toDependency(dep))
			}
		}
	}

	return result, nil
}
