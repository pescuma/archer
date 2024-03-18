package orm

import "github.com/pescuma/archer/lib/model"

type sqlRepositoryCommitFile struct {
	CommitID      model.ID `gorm:"primaryKey"`
	FileID        model.ID `gorm:"primaryKey"`
	Hash          string
	Change        model.FileChangeType
	OldIDs        string
	OldHashes     string
	LinesModified *int
	LinesAdded    *int
	LinesDeleted  *int
}

func newSqlRepositoryCommitFile(c model.ID, f *model.RepositoryCommitFile) *sqlRepositoryCommitFile {
	return &sqlRepositoryCommitFile{
		CommitID:      c,
		FileID:        f.FileID,
		Hash:          f.Hash,
		Change:        f.Change,
		OldIDs:        encodeOldFileIDs(f.OldIDs),
		OldHashes:     encodeOldFileHashes(f.OldHashes),
		LinesModified: encodeMetric(f.LinesModified),
		LinesAdded:    encodeMetric(f.LinesAdded),
		LinesDeleted:  encodeMetric(f.LinesDeleted),
	}
}
