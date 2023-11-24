package server

import (
	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initArch(r *gin.Engine) {
	r.GET("/api/arch", getP[Filters](s.archList))
}

func (s *server) archList(params *Filters) (any, error) {
	projs, err := s.listProjects(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	projsByID := lo.GroupBy(projs, func(i *model.Project) model.UUID {
		return i.ID
	})

	result := []gin.H{}

	for _, proj := range projs {
		el := s.toProject(proj)
		result = append(result, el)

		for _, dep := range proj.Dependencies {
			if projsByID[dep.Source.ID] == nil || projsByID[dep.Target.ID] == nil {
				continue
			}

			el = s.toDependency(dep)
			result = append(result, el)
		}
	}

	return result, nil
}
