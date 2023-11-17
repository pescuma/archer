package server

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) filterRepos(col []*model.Repository, search string, person string) []*model.Repository {
	search = prepareToSearch(search)
	person = prepareToSearch(person)

	return lo.Filter(col, func(i *model.Repository, index int) bool {
		if search != "" && !strings.Contains(strings.ToLower(i.Name), search) {
			return false
		}

		if person != "" && !s.repoHasPerson(i, person) {
			return false
		}

		return true
	})
}

func (s *server) repoHasPerson(i *model.Repository, person string) bool {
	for _, c := range i.ListCommits() {
		if s.filterCommit(RepoAndCommit{Repo: i, Commit: c}, "", person) {
			return true
		}
	}

	return false
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

func (s *server) toRepoReference(repo *model.Repository) gin.H {
	return gin.H{
		"id":   repo.ID,
		"name": repo.Name,
	}
}

type RepoAndCommit struct {
	Repo   *model.Repository
	Commit *model.RepositoryCommit
}

func (s *server) filterCommits(col []RepoAndCommit, repo string, person string) []RepoAndCommit {
	repo = prepareToSearch(repo)
	person = prepareToSearch(person)

	return lo.Filter(col, func(i RepoAndCommit, index int) bool {
		return s.filterCommit(i, repo, person)
	})
}

func (s *server) filterCommit(i RepoAndCommit, repo string, person string) bool {
	if i.Commit.Ignore {
		return false
	}

	if repo != "" && !strings.Contains(strings.ToLower(i.Repo.Name), repo) {
		return false
	}

	if person != "" {
		committer := s.people.GetPersonByID(i.Commit.CommitterID)
		hasCommitter := s.filterPerson(committer, person)

		author := s.people.GetPersonByID(i.Commit.AuthorID)
		hasAuthor := s.filterPerson(author, person)

		if !hasCommitter && !hasAuthor {
			return false
		}
	}

	return true
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
	case "author.name":
		return sortBy(col, func(r RepoAndCommit) string { return s.people.GetPersonByID(r.Commit.AuthorID).Name }, *asc)
	case "modifiedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.LinesModified }, *asc)
	case "addedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.LinesAdded }, *asc)
	case "deletedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.LinesDeleted }, *asc)
	case "survivedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.LinesSurvived }, *asc)
	default:
		return fmt.Errorf("unknown sort field: %s", field)
	}
}

func (s *server) toCommit(commit *model.RepositoryCommit, repo *model.Repository) gin.H {
	author := s.people.GetPersonByID(commit.AuthorID)
	committer := s.people.GetPersonByID(commit.CommitterID)

	return gin.H{
		"repo":          s.toRepoReference(repo),
		"id":            commit.ID,
		"hash":          commit.Hash,
		"message":       commit.Message,
		"date":          commit.Date,
		"parents":       commit.Parents,
		"children":      commit.Children,
		"committer":     s.toPersonReference(committer),
		"dateAuthored":  commit.DateAuthored,
		"author":        s.toPersonReference(author),
		"modifiedLines": encodeMetric(commit.LinesModified),
		"addedLines":    encodeMetric(commit.LinesAdded),
		"deletedLines":  encodeMetric(commit.LinesDeleted),
		"survivedLines": encodeMetric(commit.LinesSurvived),
	}
}
