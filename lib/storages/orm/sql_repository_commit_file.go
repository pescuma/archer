package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlRepositoryCommitFile struct {
	CommitID model.ID `gorm:"primaryKey"`
	FileID   model.ID `gorm:"primaryKey"`

	Change        model.FileChangeType
	LinesModified *int
	LinesAdded    *int
	LinesDeleted  *int

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newSqlRepositoryCommitFile(commit *model.RepositoryCommit, f *model.RepositoryCommitFile) *sqlRepositoryCommitFile {
	return &sqlRepositoryCommitFile{
		CommitID:      commit.ID,
		FileID:        f.FileID,
		Change:        f.Change,
		LinesModified: encodeMetric(f.LinesModified),
		LinesAdded:    encodeMetric(f.LinesAdded),
		LinesDeleted:  encodeMetric(f.LinesDeleted),
	}
}

func (s *sqlRepositoryCommitFile) CacheKey() string {
	return compositeKey(s.CommitID.String(), s.FileID.String())
}
