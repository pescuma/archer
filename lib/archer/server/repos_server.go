package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initRepos(r *gin.Engine) {
	r.GET("/api/repos", s.listRepos)
	r.GET("/api/repos/:id", s.getRepo)
	r.GET("/api/stats/count/repos", s.countRepos)
	r.GET("/api/stats/monthly/repos", s.getReposMonthlyStats)
}

func (s *server) listRepos(c *gin.Context) {
}

func (s *server) countRepos(c *gin.Context) {
	repos := s.repos.List()

	c.JSON(http.StatusOK, gin.H{
		"total": len(repos),
	})
}

func (s *server) getRepo(c *gin.Context) {
}

func (s *server) getReposMonthlyStats(c *gin.Context) {
	s1 := lo.GroupBy(s.repos.List(), func(repo *model.Repository) string {
		y, m, _ := repo.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(repos []*model.Repository, _ string) int {
		return len(repos)
	})

	c.JSON(http.StatusOK, s2)
}
