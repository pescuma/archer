package model

import "github.com/samber/lo"

type RepositoryCommitDetails struct {
	RepositoryID ID
	CommitID     ID
	filesByID    map[ID]*RepositoryCommitFileDetails
}

func NewRepositoryCommitDetails(repositoryID ID, commitID ID) *RepositoryCommitDetails {
	return &RepositoryCommitDetails{
		RepositoryID: repositoryID,
		CommitID:     commitID,
		filesByID:    make(map[ID]*RepositoryCommitFileDetails),
	}
}

func (l *RepositoryCommitDetails) GetOrCreateFile(fileID ID) *RepositoryCommitFileDetails {
	file, ok := l.filesByID[fileID]

	if !ok {
		file = NewRepositoryCommitFileDetails(fileID)
		l.filesByID[fileID] = file
	}

	return file
}

func (l *RepositoryCommitDetails) ListFiles() []*RepositoryCommitFileDetails {
	return lo.Values(l.filesByID)
}
