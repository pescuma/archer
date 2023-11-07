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
	SurvivedLines int

	Files map[UUID]*RepositoryCommitFile
}

func NewRepositoryCommit(hash string) *RepositoryCommit {
	result := &RepositoryCommit{
		Hash:          hash,
		ID:            NewUUID("c"),
		ModifiedLines: -1,
		AddedLines:    -1,
		DeletedLines:  -1,
		SurvivedLines: -1,
		Files:         make(map[UUID]*RepositoryCommitFile),
	}

	return result
}

func (c *RepositoryCommit) AddFile(fileID UUID) *RepositoryCommitFile {
	file := NewRepositoryCommitFile(fileID)

	c.Files[fileID] = file

	return file
}
