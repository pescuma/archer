package model

type File struct {
	Path string
	ID   UUID

	ProjectID          *UUID
	ProjectDirectoryID *UUID

	Size *Size
	Data map[string]string
}

func NewFile(path string) *File {
	return &File{
		Path: path,
		ID:   NewUUID("f"),
		Size: NewSize(),
		Data: map[string]string{},
	}
}
