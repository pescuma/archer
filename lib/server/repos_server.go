package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/model"
)

type CommitPatchParams struct {
	RepoID   model.ID   `uri:"repoID"`
	CommitID model.UUID `uri:"commitID"`
	Ignore   *bool      `json:"ignore"`
}

func (s *server) initRepos(r *gin.Engine) {
	r.GET("/api/repos", getP[ListParams](s.reposList))
	r.GET("/api/repos/:id", get(s.repoGet))
	r.GET("/api/commits", getP[ListParams](s.commitsList))
	r.PATCH("/api/repos/:repoID/commits/:commitID", patchP[CommitPatchParams](s.commitPatch))
	r.GET("/api/stats/count/repos", getP[StatsParams](s.statsCountRepos))
	r.GET("/api/stats/seen/repos", getP[StatsParams](s.statsSeenRepos))
	r.GET("/api/stats/seen/commits", getP[StatsParams](s.statsSeenCommits))
	r.GET("/api/stats/changed/lines", getP[StatsParams](s.statsChangedLines))
	r.GET("/api/stats/survived/lines", getP[StatsParams](s.statsSurvivedLines))
}

func (s *server) statsCountRepos(params *StatsParams) (any, error) {
	repos, err := s.listRepos(&params.Filters)
	if err != nil {
		return nil, err
	}

	return gin.H{
		"total": len(repos),
	}, nil
}

func (s *server) reposList(params *ListParams) (any, error) {
	repos, err := s.listRepos(&params.Filters)
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

func (s *server) commitsList(params *ListParams) (any, error) {
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

func (s *server) statsSeenRepos(params *StatsParams) (any, error) {
	repos, err := s.listRepos(&params.Filters)
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

func (s *server) statsSeenCommits(params *StatsParams) (any, error) {
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

func (s *server) statsChangedLines(params *StatsParams) (any, error) {
	//fileIDs, err := s.listFileIDsOrNil(params.FilterFile)
	//if err != nil {
	//	return nil, err
	//}
	projIDs, err := s.listProjectIDsOrNil(params.FilterProject)
	if err != nil {
		return nil, err
	}
	repoIDs, err := s.listRepoIDsOrNil(params.FilterRepo)
	if err != nil {
		return nil, err
	}
	personIDs, err := s.listPersonIDsOrNil(params.FilterPerson, params.FilterPersonID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]int)
	for _, l := range s.stats.ListLines() {
		if l.Changes.LinesTotal() <= 0 {
			continue
		}

		// TODO
		//if fileIDs != nil && !fileIDs[l.FileID] {
		//	continue
		//}
		if projIDs != nil && (l.ProjectID == nil || !projIDs[*l.ProjectID]) {
			continue
		}
		if repoIDs != nil && !repoIDs[l.RepositoryID] {
			continue
		}
		if personIDs != nil && !personIDs[l.AuthorID] && !personIDs[l.CommitterID] {
			continue
		}

		month, ok := result[l.Month]
		if !ok {
			month = make(map[string]int)
			month["modified"] = 0
			month["added"] = 0
			month["deleted"] = 0
			result[l.Month] = month
		}

		month["modified"] += l.Changes.LinesModified
		month["added"] += l.Changes.LinesAdded
		month["deleted"] += l.Changes.LinesDeleted
	}
	return result, nil
}

func (s *server) statsSurvivedLines(params *StatsParams) (any, error) {
	//fileIDs, err := s.listFileIDsOrNil(params.FilterFile)
	//if err != nil {
	//	return nil, err
	//}
	projIDs, err := s.listProjectIDsOrNil(params.FilterProject)
	if err != nil {
		return nil, err
	}
	repoIDs, err := s.listRepoIDsOrNil(params.FilterRepo)
	if err != nil {
		return nil, err
	}
	personIDs, err := s.listPersonIDsOrNil(params.FilterPerson, params.FilterPersonID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]int)
	for _, l := range s.stats.ListLines() {
		if l.Blame.Total() <= 0 {
			continue
		}

		// TODO
		//if fileIDs != nil && !fileIDs[l.FileID] {
		//	continue
		//}
		if projIDs != nil && (l.ProjectID == nil || !projIDs[*l.ProjectID]) {
			continue
		}
		if repoIDs != nil && !repoIDs[l.RepositoryID] {
			continue
		}
		if personIDs != nil && !personIDs[l.AuthorID] && !personIDs[l.CommitterID] {
			continue
		}

		month, ok := result[l.Month]
		if !ok {
			month = make(map[string]int)
			month["code"] = 0
			month["comment"] = 0
			month["blank"] = 0
			result[l.Month] = month
		}

		month["code"] += l.Blame.Code
		month["comment"] += l.Blame.Comment
		month["blank"] += l.Blame.Blank
	}
	return result, nil
}
