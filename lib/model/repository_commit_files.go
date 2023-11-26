package model

import "github.com/samber/lo"

type RepositoryCommitFiles struct {
	RepositoryID UUID
	CommitID     UUID
	byID         map[UUID]*RepositoryCommitFile
}

func NewRepositoryCommitFiles(repositoryID UUID, commitID UUID) *RepositoryCommitFiles {
	return &RepositoryCommitFiles{
		RepositoryID: repositoryID,
		CommitID:     commitID,
		byID:         make(map[UUID]*RepositoryCommitFile),
	}
}

func (l *RepositoryCommitFiles) GetOrCreate(fileID UUID) *RepositoryCommitFile {
	file, ok := l.byID[fileID]

	if !ok {
		file = NewRepositoryCommitFile(fileID)
		l.byID[fileID] = file
	}

	return file
}

func (l *RepositoryCommitFiles) List() []*RepositoryCommitFile {
	return lo.Values(l.byID)
}
