package model

type RepositoryCommitFile struct {
	FileID UUID
	ID     UUID

	AddedLines   int
	DeletedLines int
}

func NewRepositoryCommitFile(fileID UUID) *RepositoryCommitFile {
	return &RepositoryCommitFile{
		FileID: fileID,
		ID:     NewUUID("b"),
	}
}
