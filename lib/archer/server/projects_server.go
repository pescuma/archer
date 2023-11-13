package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initProjects(r *gin.Engine) {
	r.GET("/api/projects", s.listProjects)
	r.GET("/api/projects/:id", s.getProject)
	r.GET("/api/stats/count/projects", s.countProjects)
	r.GET("/api/stats/seen/projects", s.getProjectsSeenStats)
}

func (s *server) listProjects(c *gin.Context) {
}

func (s *server) countProjects(c *gin.Context) {
	all := s.projects.ListProjects(model.FilterAll)

	c.JSON(http.StatusOK, gin.H{
		"total":    len(all),
		"external": lo.CountBy(all, func(p *model.Project) bool { return p.IsExternalDependency() }),
	})
}

func (s *server) getProject(c *gin.Context) {
}

func (s *server) getProjectsSeenStats(c *gin.Context) {
	s1 := lo.GroupBy(s.projects.ListProjects(model.FilterExcludeExternal), func(proj *model.Project) string {
		y, m, _ := proj.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(projs []*model.Project, _ string) int {
		return len(projs)
	})

	c.JSON(http.StatusOK, s2)
}
