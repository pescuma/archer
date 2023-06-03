package model

type File struct {
	Path string
	ID   UUID

	ProjectID          *UUID
	ProjectDirectoryID *UUID

	RepositoryID *UUID

	ProductAreaID  *UUID
	OrganizationID *UUID
	GroupID        *UUID
	TeamID         *UUID

	Exists  bool
	Size    *Size
	Changes *Changes
	Metrics *Metrics
	Data    map[string]string
}

func NewFile(path string, id *UUID) *File {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("f")
	} else {
		uuid = *id
	}

	return &File{
		Path:    path,
		ID:      uuid,
		Exists:  true,
		Size:    NewSize(),
		Changes: NewChanges(),
		Metrics: NewMetrics(),
		Data:    map[string]string{},
	}
}
