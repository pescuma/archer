package model

type FileLineType int

const (
	CodeFileLine FileLineType = iota
	CommentFileLine
	BlankFileLine
)

type FileLine struct {
	Line     int
	AuthorID *UUID
	CommitID *UUID
	Type     FileLineType
	Text     string
}
