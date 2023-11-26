package model

import (
	"time"
)

type RepositoryCommit struct {
	Hash     string
	Message  string
	Parents  []UUID
	Children []UUID
	ID       UUID

	Date         time.Time
	CommitterID  UUID
	DateAuthored time.Time
	AuthorID     UUID

	FilesModified int
	FilesCreated  int
	FilesDeleted  int

	LinesModified int
	LinesAdded    int
	LinesDeleted  int

	Blame *Blame

	Ignore bool
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
		FilesModified: -1,
		FilesCreated:  -1,
		FilesDeleted:  -1,
		LinesModified: -1,
		LinesAdded:    -1,
		LinesDeleted:  -1,
		Blame:         NewBlame(),
	}

	return result
}
