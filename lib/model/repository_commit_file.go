package model

type RepositoryCommitFile struct {
	FileID UUID
	Hash   string

	Change     FileChangeType
	OldFileIDs map[UUID]UUID

	LinesModified int
	LinesAdded    int
	LinesDeleted  int
}

type FileChangeType int

const (
	FileChangeUnknown FileChangeType = -1
	FileModified      FileChangeType = iota
	FileRenamed
	FileCreated
	FileDeleted
)

func NewRepositoryCommitFile(fileID UUID) *RepositoryCommitFile {
	return &RepositoryCommitFile{
		FileID:        fileID,
		OldFileIDs:    make(map[UUID]UUID),
		Change:        FileChangeUnknown,
		LinesModified: -1,
		LinesAdded:    -1,
		LinesDeleted:  -1,
	}
}
