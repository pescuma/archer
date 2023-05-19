package model

type RepositoryCommitFile struct {
	FileID UUID

	AddedLines   int
	DeletedLines int
}

func NewRepositoryCommitFile(fileID UUID) *RepositoryCommitFile {
	return &RepositoryCommitFile{
		FileID: fileID,
	}
}
