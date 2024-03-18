package model

import "time"

type FileLineType int

const (
	CodeFileLine FileLineType = iota
	CommentFileLine
	BlankFileLine
)

type FileLine struct {
	Line         int
	ProjectID    *ID
	RepositoryID *UUID
	CommitID     *UUID
	AuthorID     *ID
	CommitterID  *ID
	Date         time.Time
	Type         FileLineType
	Text         string
}
