package model

import (
	"time"
)

type RepositoryCommit struct {
	ID       ID
	Hash     string
	Message  string
	Parents  []ID
	Children []ID

	Date         time.Time
	CommitterID  ID
	DateAuthored time.Time
	AuthorIDs    []ID

	FilesModified int
	FilesCreated  int
	FilesDeleted  int

	LinesModified int
	LinesAdded    int
	LinesDeleted  int

	Blame *Blame

	Ignore bool

	Files map[ID]*RepositoryCommitFile
}

func NewRepositoryCommit(id ID, hash string) *RepositoryCommit {
	result := &RepositoryCommit{
		Hash:          hash,
		ID:            id,
		FilesModified: -1,
		FilesCreated:  -1,
		FilesDeleted:  -1,
		LinesModified: -1,
		LinesAdded:    -1,
		LinesDeleted:  -1,
		Blame:         NewBlame(),
		Ignore:        false,
		Files:         map[ID]*RepositoryCommitFile{},
	}

	return result
}

func (c *RepositoryCommit) GetOrCreateFile(fileID ID) *RepositoryCommitFile {
	file, ok := c.Files[fileID]

	if !ok {
		file = NewRepositoryCommitFile(fileID)
		c.Files[fileID] = file
	}

	return file
}
