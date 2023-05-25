package model

type RepositoryCommitFile struct {
	FileID    UUID
	OldFileID *UUID

	ModifiedLines int
	AddedLines    int
	DeletedLines  int
}

func NewRepositoryCommitFile(fileID UUID) *RepositoryCommitFile {
	return &RepositoryCommitFile{
		FileID: fileID,
	}
}
