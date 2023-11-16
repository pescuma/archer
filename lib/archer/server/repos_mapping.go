package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
)

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

func (s *server) toRepo(repo *model.Repository) gin.H {
	return gin.H{
		"id":           repo.ID,
		"name":         repo.Name,
		"rootDir":      repo.RootDir,
		"vcs":          repo.VCS,
		"commitsTotal": repo.CountCommits(),
		"filesTotal":   repo.FilesTotal,
		"filesHead":    repo.FilesHead,
		"firstSeen":    repo.FirstSeen,
		"lastSeen":     repo.LastSeen,
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
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.ModifiedLines }, *asc)
	case "addedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.AddedLines }, *asc)
	case "deletedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.DeletedLines }, *asc)
	case "survivedLines":
		return sortBy(col, func(r RepoAndCommit) int { return r.Commit.SurvivedLines }, *asc)
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
		"committer":     s.toPersonReference(committer),
		"dateAuthored":  commit.DateAuthored,
		"author":        s.toPersonReference(author),
		"modifiedLines": encodeMetric(commit.ModifiedLines),
		"addedLines":    encodeMetric(commit.AddedLines),
		"deletedLines":  encodeMetric(commit.DeletedLines),
		"survivedLines": encodeMetric(commit.SurvivedLines),
	}
}
