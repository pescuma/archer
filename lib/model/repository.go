package model

import (
	"time"

	"github.com/samber/lo"
)

type Repository struct {
	Name    string
	RootDir string
	VCS     string
	ID      UUID

	Data map[string]string

	FirstSeen time.Time
	LastSeen  time.Time

	FilesTotal int
	FilesHead  int

	commitsByHash map[string]*RepositoryCommit
	commitsByID   map[UUID]*RepositoryCommit
}

func NewRepository(rootDir string, id *UUID) *Repository {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("r")
	} else {
		uuid = *id
	}

	return &Repository{
		RootDir:       rootDir,
		ID:            uuid,
		Data:          map[string]string{},
		FilesTotal:    -1,
		FilesHead:     -1,
		commitsByHash: map[string]*RepositoryCommit{},
		commitsByID:   map[UUID]*RepositoryCommit{},
	}
}

func (r *Repository) GetOrCreateCommit(hash string) *RepositoryCommit {
	return r.GetOrCreateCommitEx(hash, nil)
}

func (r *Repository) GetOrCreateCommitEx(hash string, id *UUID) *RepositoryCommit {
	result, ok := r.commitsByHash[hash]

	if !ok {
		result = NewRepositoryCommit(hash, id)
		r.commitsByHash[hash] = result
		r.commitsByID[result.ID] = result
	}

	return result
}

func (r *Repository) GetCommit(hash string) *RepositoryCommit {
	return r.commitsByHash[hash]
}

func (r *Repository) GetCommitByID(id UUID) *RepositoryCommit {
	return r.commitsByID[id]
}

func (r *Repository) ContainsCommit(hash string) bool {
	_, ok := r.commitsByHash[hash]
	return ok
}

func (r *Repository) ListCommits() []*RepositoryCommit {
	return lo.Values(r.commitsByHash)
}

func (r *Repository) CountCommits() int {
	return len(r.commitsByHash)
}

func (r *Repository) SeenAt(ts ...time.Time) {
	empty := time.Time{}

	for _, t := range ts {
		t = t.UTC().Round(time.Second)

		if r.FirstSeen == empty || t.Before(r.FirstSeen) {
			r.FirstSeen = t
		}
		if r.LastSeen == empty || t.After(r.LastSeen) {
			r.LastSeen = t
		}
	}
}