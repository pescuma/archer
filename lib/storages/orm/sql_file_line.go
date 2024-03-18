package orm

import (
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlFileLine struct {
	FileID model.ID `gorm:"primaryKey;autoIncrement:false"`
	Line   int      `gorm:"primaryKey"`

	ProjectID    *model.ID
	RepositoryID *model.UUID
	CommitID     *model.UUID
	AuthorID     *model.ID
	CommitterID  *model.ID
	Date         time.Time

	Type model.FileLineType
	Text string
}

func newSqlFileLine(fileID model.ID, f *model.FileLine) *sqlFileLine {
	return &sqlFileLine{
		FileID:       fileID,
		Line:         f.Line,
		ProjectID:    f.ProjectID,
		RepositoryID: f.RepositoryID,
		CommitID:     f.CommitID,
		AuthorID:     f.AuthorID,
		Date:         f.Date,
		CommitterID:  f.CommitterID,
		Type:         f.Type,
		Text:         f.Text,
	}
}
