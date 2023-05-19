package model

type ProjectFile struct {
	RelativePath string
	ID           UUID

	Size *Size
}

func NewProjectFile(relativePath string) *ProjectFile {
	return &ProjectFile{
		RelativePath: relativePath,
		ID:           NewUUID("f"),
		Size:         NewSize(),
	}
}
