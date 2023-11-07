package model

type RepositoryCommitFile struct {
	FileID     UUID
	OldFileIDs map[string]UUID

	ModifiedLines int
	AddedLines    int
	DeletedLines  int
	SurvivedLines int
}

func NewRepositoryCommitFile(fileID UUID) *RepositoryCommitFile {
	return &RepositoryCommitFile{
		FileID:        fileID,
		OldFileIDs:    make(map[string]UUID),
		ModifiedLines: -1,
		AddedLines:    -1,
		DeletedLines:  -1,
		SurvivedLines: -1,
	}
}
