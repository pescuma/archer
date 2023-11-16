package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

type GridParams struct {
	Sort   string `form:"sort"`
	Asc    *bool  `form:"asc"`
	Offset *int   `form:"offset"`
	Limit  *int   `form:"limit"`
}
type ListRepoParams struct {
	GridParams
}
type ListCommitParams struct {
	GridParams
}

func (s *server) initRepos(r *gin.Engine) {
	r.GET("/api/repos", getP[ListRepoParams](s.listRepos))
	r.GET("/api/repos/:id", get(s.getRepo))
	r.GET("/api/commits", getP[ListCommitParams](s.listCommits))
	r.GET("/api/stats/count/repos", get(s.countRepos))
	r.GET("/api/stats/seen/repos", get(s.getReposSeenStats))
	r.GET("/api/stats/seen/commits", get(s.getCommitsSeenStats))
	r.GET("/api/stats/changed/lines", get(s.getChangedLines))
	r.GET("/api/stats/survived/lines", get(s.getSurvivedLines))
}

func (s *server) countRepos() (any, error) {
	repos := s.repos.List()

	return gin.H{
		"total": len(repos),
	}, nil
}

func (s *server) listRepos(params *ListRepoParams) (any, error) {
	repos := s.repos.List()

	err := s.sortRepos(repos, params.Sort, params.Asc)
	if err != nil {
		return nil, err
	}

	total := len(repos)

	repos = paginate(repos, params.Offset, params.Limit)

	var result []gin.H
	for _, r := range repos {
		result = append(result, s.toRepo(r))
	}

	return gin.H{
		"data":  result,
		"total": total,
	}, nil
}

func (s *server) getRepo() (any, error) {
	return nil, nil
}

func (s *server) listCommits(params *ListCommitParams) (any, error) {
	commits := lo.FlatMap(s.repos.List(), func(i *model.Repository, index int) []RepoAndCommit {
		return lo.Map(i.ListCommits(), func(c *model.RepositoryCommit, _ int) RepoAndCommit {
			return RepoAndCommit{
				Repo:   i,
				Commit: c,
			}
		})
	})

	err := s.sortCommits(commits, params.Sort, params.Asc)
	if err != nil {
		return nil, err
	}

	total := len(commits)

	commits = paginate(commits, params.Offset, params.Limit)

	var result []gin.H
	for _, rc := range commits {
		result = append(result, s.toCommit(rc.Commit, rc.Repo))
	}

	return gin.H{
		"data":  result,
		"total": total,
	}, nil
}

func (s *server) getReposSeenStats() (any, error) {
	s1 := lo.GroupBy(s.repos.List(), func(i *model.Repository) string {
		y, m, _ := i.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(is []*model.Repository, _ string) int {
		return len(is)
	})

	return s2, nil
}

func (s *server) getCommitsSeenStats() (any, error) {
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

	return s4, nil
}

func (s *server) getChangedLines() (any, error) {
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

	return s4, nil
}

func (s *server) getSurvivedLines() (any, error) {
	s1, err := s.storage.ComputeSurvivedLines()
	if err != nil {
		return nil, err
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

	return s3, nil
}
