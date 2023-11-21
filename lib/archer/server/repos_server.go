package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

type RepoFilters struct {
	FilterFile    string `form:"file"`
	FilterProject string `form:"proj"`
	FilterRepo    string `form:"repo"`
	FilterPerson  string `form:"person"`
}
type RepoListParams struct {
	GridParams
	RepoFilters
}

type CommitFilters struct {
	FilterFile    string `form:"file"`
	FilterProject string `form:"proj"`
	FilterRepo    string `form:"repo"`
	FilterPerson  string `form:"person"`
}

type CommitListParams struct {
	GridParams
	CommitFilters
}
type CommitPatchParams struct {
	RepoID   model.UUID `uri:"repoID"`
	CommitID model.UUID `uri:"commitID"`
	Ignore   *bool      `json:"ignore"`
}

type StatsReposParams struct {
	RepoFilters
}
type StatsCommitsParams struct {
	CommitFilters
}
type StatsLinesParams struct {
	CommitFilters
}

func (s *server) initRepos(r *gin.Engine) {
	r.GET("/api/repos", getP[RepoListParams](s.reposList))
	r.GET("/api/repos/:id", get(s.repoGet))
	r.GET("/api/commits", getP[CommitListParams](s.commitsList))
	r.PATCH("/api/repos/:repoID/commits/:commitID", patchP[CommitPatchParams](s.commitPatch))
	r.GET("/api/stats/count/repos", getP[StatsReposParams](s.statsCountRepos))
	r.GET("/api/stats/seen/repos", getP[StatsReposParams](s.statsSeenRepos))
	r.GET("/api/stats/seen/commits", getP[StatsCommitsParams](s.statsSeenCommits))
	r.GET("/api/stats/changed/lines", getP[StatsLinesParams](s.statsChangedLines))
	r.GET("/api/stats/survived/lines", getP[StatsLinesParams](s.statsSurvivedLines))
}

func (s *server) statsCountRepos(params *StatsReposParams) (any, error) {
	repos, err := s.listRepos(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	return gin.H{
		"total": len(repos),
	}, nil
}

func (s *server) reposList(params *RepoListParams) (any, error) {
	repos, err := s.listRepos(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	err = s.sortRepos(repos, params.Sort, params.Asc)
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

func (s *server) repoGet() (any, error) {
	return nil, nil
}

func (s *server) commitsList(params *CommitListParams) (any, error) {
	commits, err := s.listReposAndCommits(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	err = s.sortCommits(commits, params.Sort, params.Asc)
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

func (s *server) commitPatch(params *CommitPatchParams) (any, error) {
	repo := s.repos.GetByID(params.RepoID)
	if repo == nil {
		return nil, errorNotFound
	}

	commit, ok := s.commits[params.CommitID]
	if !ok {
		return nil, errorNotFound
	}

	changed := false

	if params.Ignore != nil && commit.Ignore != *params.Ignore {
		commit.Ignore = *params.Ignore
		changed = true
	}

	if changed {
		err := s.storage.WriteCommit(repo, commit)
		if err != nil {
			return nil, err
		}
	}

	return commit, nil
}

func (s *server) statsSeenRepos(params *StatsReposParams) (any, error) {
	repos, err := s.listRepos(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]int)
	for _, f := range repos {
		y, m, _ := f.FirstSeen.Date()
		s.incSeenStats(result, y, m, "firstSeen")

		y, m, _ = f.LastSeen.Date()
		s.incSeenStats(result, y, m, "lastSeen")
	}

	return result, nil
}

func (s *server) statsSeenCommits(params *StatsCommitsParams) (any, error) {
	commits, _ := s.listReposAndCommits(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)

	s3 := lo.GroupBy(commits, func(i RepoAndCommit) string {
		y, m, _ := i.Commit.Date.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s4 := lo.MapValues(s3, func(is []RepoAndCommit, _ string) int {
		return len(is)
	})

	return s4, nil
}

func (s *server) statsChangedLines(params *StatsLinesParams) (any, error) {
	commits, _ := s.listReposAndCommits(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)

	s1 := lo.GroupBy(commits, func(i RepoAndCommit) string {
		y, m, _ := i.Commit.Date.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(i []RepoAndCommit, _ string) gin.H {
		return gin.H{
			"modified": lo.SumBy(i, func(j RepoAndCommit) int { return j.Commit.LinesModified }),
			"added":    lo.SumBy(i, func(j RepoAndCommit) int { return j.Commit.LinesAdded }),
			"deleted":  lo.SumBy(i, func(j RepoAndCommit) int { return j.Commit.LinesDeleted }),
		}
	})

	return s2, nil
}

func (s *server) statsSurvivedLines(params *StatsLinesParams) (any, error) {
	//fileIDs, err := s.createFileFilter(params.FilterFile)
	//if err != nil {
	//	return nil, err
	//}
	//
	//projIDs, err := s.createProjectFilter(params.FilterProject)
	//if err != nil {
	//	return nil, err
	//}

	repoIDs, err := s.createRepoFilter(params.FilterRepo)
	if err != nil {
		return nil, err
	}

	personIDs, err := s.createPersonFilter(params.FilterPerson)

	result := make(map[string]map[string]int)
	for _, l := range s.stats.ListLines() {
		if repoIDs != nil && !repoIDs[l.RepositoryID] {
			continue
		}
		if personIDs != nil && !personIDs[l.AuthorID] && !personIDs[l.CommitterID] {
			continue
		}

		m, ok := result[l.Month]
		if !ok {
			m = make(map[string]int, 3)
			m["blank"] = 0
			m["comment"] = 0
			m["code"] = 0
			result[l.Month] = m
		}

		m["blank"] += l.Blame.Blank
		m["comment"] += l.Blame.Comment
		m["code"] += l.Blame.Code
	}
	return result, nil
}
