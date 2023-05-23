package model

type File struct {
	Path string
	ID   UUID

	ProjectID          *UUID
	ProjectDirectoryID *UUID

	RepositoryID *UUID

	Exists  bool
	Size    *Size
	Metrics *Metrics
	Data    map[string]string
}

func NewFile(path string) *File {
	return &File{
		Path:    path,
		ID:      NewUUID("f"),
		Exists:  true,
		Size:    NewSize(),
		Metrics: NewMetrics(),
		Data:    map[string]string{},
	}
}
