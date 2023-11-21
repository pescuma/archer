package model

type RepositoryCommitFile struct {
	FileID     UUID
	OldFileIDs map[UUID]UUID

	Change FileChangeType

	LinesModified int
	LinesAdded    int
	LinesDeleted  int
}

type FileChangeType int

const (
	Unknown  FileChangeType = -1
	Modified FileChangeType = iota
	Renamed
	Created
	Deleted
)

func NewRepositoryCommitFile(fileID UUID) *RepositoryCommitFile {
	return &RepositoryCommitFile{
		FileID:        fileID,
		OldFileIDs:    make(map[UUID]UUID),
		Change:        Unknown,
		LinesModified: -1,
		LinesAdded:    -1,
		LinesDeleted:  -1,
	}
}
