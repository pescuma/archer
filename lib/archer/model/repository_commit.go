package model

import (
	"time"
)

type RepositoryCommit struct {
	Hash    string
	Message string
	Parents []string
	ID      UUID

	Date         time.Time
	CommitterID  UUID
	DateAuthored time.Time
	AuthorID     UUID

	ModifiedLines int
	AddedLines    int
	DeletedLines  int

	Files []*RepositoryCommitFile
}

func NewRepositoryCommit(hash string) *RepositoryCommit {
	return &RepositoryCommit{
		Hash: hash,
		ID:   NewUUID("c"),
	}
}

func (c *RepositoryCommit) AddFile(fileID UUID, modifiedLines, addedLines, deletedLines int) {
	c.Files = append(c.Files, &RepositoryCommitFile{
		FileID:        fileID,
		ModifiedLines: modifiedLines,
		AddedLines:    addedLines,
		DeletedLines:  deletedLines,
	})
}
