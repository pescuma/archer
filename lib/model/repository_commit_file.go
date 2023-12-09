package model

type RepositoryCommitFile struct {
	FileID UUID
	Hash   string

	Change    FileChangeType
	OldIDs    map[UUID]UUID
	OldHashes map[UUID]string

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
		Change:        FileChangeUnknown,
		OldIDs:        make(map[UUID]UUID),
		OldHashes:     make(map[UUID]string),
		LinesModified: -1,
		LinesAdded:    -1,
		LinesDeleted:  -1,
	}
}
