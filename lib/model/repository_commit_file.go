package model

type RepositoryCommitFile struct {
	FileID ID
	Hash   string

	Change    FileChangeType
	OldIDs    map[UUID]ID
	OldHashes map[UUID]string

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
		OldIDs:        make(map[UUID]ID),
		OldHashes:     make(map[UUID]string),
		LinesModified: -1,
		LinesAdded:    -1,
		LinesDeleted:  -1,
	}
}
