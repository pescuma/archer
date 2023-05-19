package model

type ProjectDirectory struct {
	RelativePath string
	Type         ProjectDirectoryType
	ID           UUID

	Size *Size
	Data map[string]string
}

func NewProjectDirectory(relativePath string) *ProjectDirectory {
	return &ProjectDirectory{
		RelativePath: relativePath,
		ID:           NewUUID("d"),
		Size:         NewSize(),
		Data:         map[string]string{},
	}
}
