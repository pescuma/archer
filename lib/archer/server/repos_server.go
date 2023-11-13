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
	r.GET("/api/stats/seen/repos", s.getReposSeenStats)
	r.GET("/api/stats/seen/commits", s.getCommitsSeenStats)
	r.GET("/api/stats/churn/lines", s.getChurnLines)
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

func (s *server) getReposSeenStats(c *gin.Context) {
	s1 := lo.GroupBy(s.repos.List(), func(repo *model.Repository) string {
		y, m, _ := repo.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(repos []*model.Repository, _ string) int {
		return len(repos)
	})

	c.JSON(http.StatusOK, s2)
}

func (s *server) getCommitsSeenStats(c *gin.Context) {
	s1 := lo.FlatMap(s.repos.List(), func(repo *model.Repository, index int) []*model.RepositoryCommit {
		return repo.ListCommits()
	})
	s2 := lo.GroupBy(s1, func(commit *model.RepositoryCommit) string {
		y, m, _ := commit.Date.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s3 := lo.MapValues(s2, func(commits []*model.RepositoryCommit, _ string) int {
		return len(commits)
	})

	c.JSON(http.StatusOK, s3)
}

func (s *server) getChurnLines(c *gin.Context) {
	s1 := lo.FlatMap(s.repos.List(), func(repo *model.Repository, index int) []*model.RepositoryCommit {
		return repo.ListCommits()
	})
	s2 := lo.GroupBy(s1, func(commit *model.RepositoryCommit) string {
		y, m, _ := commit.Date.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s3 := lo.MapValues(s2, func(commits []*model.RepositoryCommit, _ string) gin.H {
		return gin.H{
			"modified": lo.SumBy(commits, func(commit *model.RepositoryCommit) int { return commit.ModifiedLines }),
			"added":    lo.SumBy(commits, func(commit *model.RepositoryCommit) int { return commit.AddedLines }),
			"deleted":  lo.SumBy(commits, func(commit *model.RepositoryCommit) int { return commit.DeletedLines }),
		}
	})

	c.JSON(http.StatusOK, s3)
}
