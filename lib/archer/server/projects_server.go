package server

import (
	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

type ProjectsFilters struct {
	FilterFile    string `form:"file"`
	FilterProject string `form:"q"`
	FilterRepo    string `form:"repo"`
	FilterPerson  string `form:"person"`
}

type ProjectsListParams struct {
	GridParams
	ProjectsFilters
}

type StatsProjectsParams struct {
	ProjectsFilters
}

func (s *server) initProjects(r *gin.Engine) {
	r.GET("/api/projects", getP[ProjectsListParams](s.projectsList))
	r.GET("/api/projects/:id", get(s.projectGet))
	r.GET("/api/stats/count/projects", getP[StatsProjectsParams](s.statsCountProjects))
	r.GET("/api/stats/seen/projects", getP[StatsProjectsParams](s.statsProjectsSeen))
}

func (s *server) projectsList(params *ProjectsListParams) (any, error) {
	projs, err := s.listProjects(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	err = s.sortProjects(projs, params.Sort, params.Asc)
	if err != nil {
		return nil, err
	}

	total := len(projs)

	projs = paginate(projs, params.Offset, params.Limit)

	var result []gin.H
	for _, r := range projs {
		result = append(result, s.toProject(r))
	}

	return gin.H{
		"data":  result,
		"total": total,
	}, nil
}

func (s *server) statsCountProjects(params *StatsProjectsParams) (any, error) {
	projs, err := s.listProjects(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	external := lo.Filter(s.projects.ListProjects(model.FilterAll), func(i *model.Project, index int) bool {
		return i.IsExternalDependency()
	})

	return gin.H{
		"total":    len(projs),
		"external": len(external),
	}, nil
}

func (s *server) projectGet() (any, error) {
	return nil, nil
}

func (s *server) statsProjectsSeen(params *StatsProjectsParams) (any, error) {
	projs, err := s.listProjects(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]int)
	for _, f := range projs {
		y, m, _ := f.FirstSeen.Date()
		s.incSeenStats(result, y, m, "firstSeen")

		y, m, _ = f.LastSeen.Date()
		s.incSeenStats(result, y, m, "lastSeen")
	}

	return result, nil
}
