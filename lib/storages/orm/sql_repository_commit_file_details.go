package orm

import "github.com/pescuma/archer/lib/model"

type sqlRepositoryCommitFileDetails struct {
	CommitID  model.ID `gorm:"primaryKey"`
	FileID    model.ID `gorm:"primaryKey"`
	Hash      string
	OldIDs    string
	OldHashes string
}

func newSqlRepositoryCommitFileDetails(c model.ID, f *model.RepositoryCommitFileDetails) *sqlRepositoryCommitFileDetails {
	return &sqlRepositoryCommitFileDetails{
		CommitID:  c,
		FileID:    f.FileID,
		Hash:      f.Hash,
		OldIDs:    encodeOldFileIDs(f.OldIDs),
		OldHashes: encodeOldFileHashes(f.OldHashes),
	}
}
