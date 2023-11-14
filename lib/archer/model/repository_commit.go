package model

import (
	"time"
)

type RepositoryCommit struct {
	Hash    string
	Message string
	Parents []UUID
	ID      UUID

	Date         time.Time
	CommitterID  UUID
	DateAuthored time.Time
	AuthorID     UUID

	ModifiedLines int
	AddedLines    int
	DeletedLines  int
	SurvivedLines int

	Ignore bool

	Files map[UUID]*RepositoryCommitFile
}

func NewRepositoryCommit(hash string, id *UUID) *RepositoryCommit {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("c")
	} else {
		uuid = *id
	}

	result := &RepositoryCommit{
		Hash:          hash,
		ID:            uuid,
		ModifiedLines: -1,
		AddedLines:    -1,
		DeletedLines:  -1,
		SurvivedLines: -1,
		Files:         make(map[UUID]*RepositoryCommitFile),
	}

	return result
}

func (c *RepositoryCommit) GetOrCreateFile(fileID UUID) *RepositoryCommitFile {
	file, ok := c.Files[fileID]

	if !ok {
		file = NewRepositoryCommitFile(fileID)
		c.Files[fileID] = file
	}

	return file
}
