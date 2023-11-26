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
	ProjectID    *UUID
	RepositoryID *UUID
	CommitID     *UUID
	AuthorID     *UUID
	CommitterID  *UUID
	Date         time.Time
	Type         FileLineType
	Text         string
}
