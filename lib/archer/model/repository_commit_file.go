package model

type RepositoryCommitFile struct {
	FileID     UUID
	OldFileIDs map[UUID]UUID

	LinesModified int
	LinesAdded    int
	LinesDeleted  int
	LinesSurvived int
}

func NewRepositoryCommitFile(fileID UUID) *RepositoryCommitFile {
	return &RepositoryCommitFile{
		FileID:        fileID,
		OldFileIDs:    make(map[UUID]UUID),
		LinesModified: -1,
		LinesAdded:    -1,
		LinesDeleted:  -1,
		LinesSurvived: -1,
	}
}
