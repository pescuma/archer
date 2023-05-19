package model

import (
	"time"
)

type RepositoryCommit struct {
	Hash string
	ID   UUID

	Date         time.Time
	CommitterID  UUID
	DateAuthored time.Time
	AuthorID     UUID

	AddedLines   int
	DeletedLines int

	Files []*RepositoryCommitFile
}

func NewRepositoryCommit(hash string) *RepositoryCommit {
	return &RepositoryCommit{
		Hash: hash,
		ID:   NewUUID("c"),
	}
}

func (c RepositoryCommit) AddFile(fileID UUID, addedLines int, deletedLines int) {
	c.Files = append(c.Files, &RepositoryCommitFile{
		FileID:       fileID,
		AddedLines:   addedLines,
		DeletedLines: deletedLines,
	})
}
