package model

type RepositoryCommitFile struct {
	FileID        ID
	Change        FileChangeType
	LinesModified int
	LinesAdded    int
	LinesDeleted  int
}

type FileChangeType int

const (
	FileChangeUnknown FileChangeType = -1
	FileNotChanged    FileChangeType = iota
	FileModified
	FileRenamed
	FileCreated
	FileDeleted
)

func NewRepositoryCommitFile(fileID ID) *RepositoryCommitFile {
	return &RepositoryCommitFile{
		FileID:        fileID,
		Change:        FileChangeUnknown,
		LinesModified: -1,
		LinesAdded:    -1,
		LinesDeleted:  -1,
	}
}
