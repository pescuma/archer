package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-set/v2"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func (s *server) listRepos(params *Filters) ([]*model.Repository, error) {
	return s.filterRepos(s.repos.List(), params)
}

func (s *server) filterRepos(col []*model.Repository, params *Filters) ([]*model.Repository, error) {
	repoFilter, err := s.createRepoFilter(params.FilterRepo)
	if err != nil {
		return nil, err
	}

	projIDs, err := s.listProjectIDsOrNil(params.FilterProject)
	if err != nil {
		return nil, err
	}
	fileIDs, err := s.listFileIDsOrNil(params.FilterFile)
	if err != nil {
		return nil, err
	}
	personIDs, err := s.listPersonIDsOrNil(params.FilterPerson, params.FilterPersonID)
	if err != nil {
		return nil, err
	}

	return lo.Filter(col, func(i *model.Repository, index int) bool {
		if !repoFilter(i) {
			return false
		}

		if projIDs != nil {
			found := false
			for _, f := range s.files.ListFiles() {
				if f.RepositoryID != nil && *f.RepositoryID == i.ID && f.ProjectID != nil && projIDs[*f.ProjectID] {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}

		if fileIDs != nil {
			found := false
			for _, f := range s.files.ListFiles() {
				if f.RepositoryID != nil && *f.RepositoryID == i.ID && fileIDs[f.ID] {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}

		if personIDs != nil {
			found := false
			for _, f := range s.files.ListFiles() {
				if f.RepositoryID != nil && *f.RepositoryID == i.ID {
					ps := s.peopleRelations.ListPeopleByFile(f.ID)
					if utils.MapKeysHaveIntersection(ps, personIDs) {
						found = true
						break
					}
				}
			}
			if !found {
				return false
			}
		}

		return true
	}), nil
}

func (s *server) createRepoFilter(repo string) (func(*model.Repository) bool, error) {
	repo = prepareToSearch(repo)

	switch {
	case repo != "":
		f, err := filters.ParseStringFilter(repo)
		if err != nil {
			return nil, err
		}

		return func(r *model.Repository) bool {
			return f(r.Name)
		}, nil

	default:
		return func(_ *model.Repository) bool { return true }, nil
	}
}

func (s *server) listRepoIDsOrNil(repo string) (map[model.UUID]bool, error) {
	repo = prepareToSearch(repo)

	switch {
	case repo != "":
		repos, err := s.listRepos(&Filters{FilterRepo: repo})
		if err != nil {
			return nil, err
		}

		result := make(map[model.UUID]bool, len(repos))
		for _, p := range repos {
			result[p.ID] = true
		}
		return result, nil

	default:
		return nil, nil
	}
}

func (s *server) sortRepos(col []*model.Repository, field string, asc *bool) error {
	if field == "" {
		field = "name"
	}
	if asc == nil {
		asc = new(bool)
		*asc = field == "name" || field == "rootDir" || field == "vcs"
	}

	switch field {
	case "name":
		return sortBy(col, func(r *model.Repository) string { return r.Name }, *asc)
	case "rootDir":
		return sortBy(col, func(r *model.Repository) string { return r.RootDir }, *asc)
	case "vcs":
		return sortBy(col, func(r *model.Repository) string { return r.VCS }, *asc)
	case "commitsTotal":
		return sortBy(col, func(r *model.Repository) int { return r.CountCommits() }, *asc)
	case "filesTotal":
		return sortBy(col, func(r *model.Repository) int { return r.FilesTotal }, *asc)
	case "filesHead":
		return sortBy(col, func(r *model.Repository) int { return r.FilesHead }, *asc)
	case "firstSeen":
		return sortBy(col, func(r *model.Repository) int64 { return r.FirstSeen.UnixMilli() }, *asc)
	case "lastSeen":
		return sortBy(col, func(r *model.Repository) int64 { return r.LastSeen.UnixMilli() }, *asc)
	default:
		return fmt.Errorf("unknown sort field: %s", field)
	}
}

func (s *server) toRepo(r *model.Repository) gin.H {
	return gin.H{
		"id":           r.ID,
		"name":         r.Name,
		"rootDir":      r.RootDir,
		"vcs":          r.VCS,
		"commitsTotal": r.CountCommits(),
		"filesTotal":   encodeMetric(r.FilesTotal),
		"filesHead":    encodeMetric(r.FilesHead),
		"firstSeen":    encodeDate(r.FirstSeen),
		"lastSeen":     encodeDate(r.LastSeen),
	}
}

func (s *server) toRepoReference(id *model.UUID) gin.H {
	if id == nil {
		return nil
	}

	repo := s.repos.GetByID(*id)

	return gin.H{
		"id":   repo.ID,
		"name": repo.Name,
	}
}

type RepoAndCommit struct {
	Repo   *model.Repository
	Commit *model.RepositoryCommit
}

func (s *server) listReposAndCommits(file string, proj string, repo string, person string) ([]RepoAndCommit, error) {
	commits := lo.FlatMap(s.repos.List(), func(i *model.Repository, index int) []RepoAndCommit {
		return lo.Map(i.ListCommits(), func(c *model.RepositoryCommit, _ int) RepoAndCommit {
			return RepoAndCommit{
				Repo:   i,
				Commit: c,
			}
		})
	})

	return s.filterCommits(commits, file, proj, repo, person)
}

func (s *server) filterCommits(col []RepoAndCommit, file string, proj string, repo string, person string) ([]RepoAndCommit, error) {
	file = prepareToSearch(file)
	proj = prepareToSearch(proj)
	repo = prepareToSearch(repo)
	person = prepareToSearch(person)

	var ids *set.Set[model.UUID]
	if file != "" || proj != "" || repo != "" || person != "" {
		r, err := s.storage.QueryCommits(file, proj, repo, person)
		if err != nil {
			return nil, err
		}
		ids = set.From(r)
	}

	return lo.Filter(col, func(i RepoAndCommit, index int) bool {
		if i.Commit.Ignore {
			return false
		}

		if ids != nil && !ids.Contains(i.Commit.ID) {
			return false
		}

		return true
	}), nil
}

func (s *server) sortCommits(col []RepoAndCommit, field string, asc *bool) error {
	if field == "" {
		field = "date"
	}
	if asc == nil {
		asc = new(bool)
		*asc = field != "date" && field != "dateAuthored"
	}

	switch field {
	case "repo.name":
		return sortBy(col, func(r RepoAndCommit) string { return r.Repo.Name }, *asc)
	case "hash":
		return sortBy(col, func(r RepoAndCommit) string { return r.Commit.Hash }, *asc)
	case "message":
		return sortBy(col, func(r RepoAndCommit) string { return r.Commit.Message }, *asc)
	case "date":
		return sortBy(col, func(r RepoAndCommit) int64 { return r.Commit.Date.UnixMilli() }, *asc)
	case "committer.name":
		return sortBy(col, func(r RepoAndCommit) string { return s.people.GetPersonByID(r.Commit.CommitterID).Name }, *asc)
	case "dateAuthored":
		return sortBy(col, func(r RepoAndCommit) int64 { return r.Commit.DateAuthored.UnixMilli() }, *asc)
	case "authors.name":
		return sortBy(col, func(r RepoAndCommit) string { return s.people.GetPersonByID(r.Commit.AuthorIDs[0]).Name }, *asc)
	case "modifiedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.LinesModified }, *asc)
	case "addedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.LinesAdded }, *asc)
	case "deletedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.LinesDeleted }, *asc)
	case "blame":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.Blame.Total() }, *asc)
	default:
		return fmt.Errorf("unknown sort field: %s", field)
	}
}

func (s *server) toCommit(commit *model.RepositoryCommit, repo *model.Repository) gin.H {
	return gin.H{
		"id":            commit.ID,
		"repo":          s.toRepoReference(&repo.ID),
		"hash":          commit.Hash,
		"message":       commit.Message,
		"date":          commit.Date,
		"parents":       commit.Parents,
		"children":      commit.Children,
		"committer":     s.toPersonReference(&commit.CommitterID),
		"dateAuthored":  commit.DateAuthored,
		"authors":       lo.Map(commit.AuthorIDs, func(a model.UUID, _ int) gin.H { return s.toPersonReference(&a) }),
		"modifiedLines": encodeMetric(commit.LinesModified),
		"addedLines":    encodeMetric(commit.LinesAdded),
		"deletedLines":  encodeMetric(commit.LinesDeleted),
		"blame":         encodeMetric(commit.Blame.Total()),
	}
}
