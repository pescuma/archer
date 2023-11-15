package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initProjects(r *gin.Engine) {
	r.GET("/api/projects", get(s.listProjects))
	r.GET("/api/projects/:id", get(s.getProject))
	r.GET("/api/stats/count/projects", get(s.countProjects))
	r.GET("/api/stats/seen/projects", get(s.getProjectsSeenStats))
}

func (s *server) listProjects() (any, error) {
	return nil, nil
}

func (s *server) countProjects() (any, error) {
	all := s.projects.ListProjects(model.FilterAll)

	return gin.H{
		"total":    len(all),
		"external": lo.CountBy(all, func(p *model.Project) bool { return p.IsExternalDependency() }),
	}, nil
}

func (s *server) getProject() (any, error) {
	return nil, nil
}

func (s *server) getProjectsSeenStats() (any, error) {
	s1 := lo.GroupBy(s.projects.ListProjects(model.FilterExcludeExternal), func(proj *model.Project) string {
		y, m, _ := proj.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(projs []*model.Project, _ string) int {
		return len(projs)
	})

	return s2, nil
}
