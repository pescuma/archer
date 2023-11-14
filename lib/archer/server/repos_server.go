package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initRepos(r *gin.Engine) {
	r.GET("/api/repos", s.listRepos)
	r.GET("/api/repos/:id", s.getRepo)
	r.GET("/api/stats/count/repos", s.countRepos)
	r.GET("/api/stats/seen/repos", s.getReposSeenStats)
	r.GET("/api/stats/seen/commits", s.getCommitsSeenStats)
	r.GET("/api/stats/changed/lines", s.getChangedLines)
	r.GET("/api/stats/survived/lines", s.getSurvivedLines)
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
	s1 := lo.GroupBy(s.repos.List(), func(i *model.Repository) string {
		y, m, _ := i.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(is []*model.Repository, _ string) int {
		return len(is)
	})

	c.JSON(http.StatusOK, s2)
}

func (s *server) getCommitsSeenStats(c *gin.Context) {
	s1 := lo.FlatMap(s.repos.List(), func(i *model.Repository, index int) []*model.RepositoryCommit {
		return i.ListCommits()
	})
	s2 := lo.Filter(s1, func(i *model.RepositoryCommit, index int) bool {
		return !i.Ignore
	})
	s3 := lo.GroupBy(s2, func(i *model.RepositoryCommit) string {
		y, m, _ := i.Date.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s4 := lo.MapValues(s3, func(is []*model.RepositoryCommit, _ string) int {
		return len(is)
	})

	c.JSON(http.StatusOK, s4)
}

func (s *server) getChangedLines(c *gin.Context) {
	s1 := lo.FlatMap(s.repos.List(), func(i *model.Repository, index int) []*model.RepositoryCommit {
		return i.ListCommits()
	})
	s2 := lo.Filter(s1, func(i *model.RepositoryCommit, index int) bool {
		return !i.Ignore
	})
	s3 := lo.GroupBy(s2, func(i *model.RepositoryCommit) string {
		y, m, _ := i.Date.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s4 := lo.MapValues(s3, func(i []*model.RepositoryCommit, _ string) gin.H {
		return gin.H{
			"modified": lo.SumBy(i, func(commit *model.RepositoryCommit) int { return commit.ModifiedLines }),
			"added":    lo.SumBy(i, func(commit *model.RepositoryCommit) int { return commit.AddedLines }),
			"deleted":  lo.SumBy(i, func(commit *model.RepositoryCommit) int { return commit.DeletedLines }),
		}
	})

	c.JSON(http.StatusOK, s4)
}

func (s *server) getSurvivedLines(c *gin.Context) {
	s1, err := s.storage.ComputeSurvivedLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s2 := lo.GroupBy(s1, func(i *archer.SurvivedLineCount) string {
		return i.Month
	})
	s3 := lo.MapValues(s2, func(is []*archer.SurvivedLineCount, _ string) gin.H {
		blank := 0
		comment := 0
		code := 0
		for _, i := range is {
			switch i.LineType {
			case model.BlankFileLine:
				blank += i.Lines
			case model.CommentFileLine:
				comment += i.Lines
			case model.CodeFileLine:
				code += i.Lines

			}
		}
		return gin.H{
			"blank":   blank,
			"comment": comment,
			"code":    code,
		}
	})

	c.JSON(http.StatusOK, s3)
}
