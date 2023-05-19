package model

import (
	"github.com/samber/lo"
)

type Repository struct {
	RootDir string
	VCS     string
	ID      UUID

	Data map[string]string

	Commits map[string]*RepositoryCommit
}

func NewRepository(rootDir string) *Repository {
	return &Repository{
		RootDir: rootDir,
		ID:      NewUUID("r"),
		Data:    map[string]string{},
		Commits: map[string]*RepositoryCommit{},
	}
}

func (r *Repository) GetCommit(hash string) *RepositoryCommit {
	result, ok := r.Commits[hash]

	if !ok {
		result = NewRepositoryCommit(hash)
		r.Commits[hash] = result
	}

	return result
}

func (r *Repository) ContainsCommit(hash string) bool {
	_, ok := r.Commits[hash]
	return ok
}

func (r *Repository) ListCommits() []*RepositoryCommit {
	return lo.Values(r.Commits)
}
