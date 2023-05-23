package model

type ProjectDirectory struct {
	RelativePath string
	Type         ProjectDirectoryType
	ID           UUID

	Size    *Size
	Metrics *Metrics
	Data    map[string]string
}

func NewProjectDirectory(relativePath string) *ProjectDirectory {
	return &ProjectDirectory{
		RelativePath: relativePath,
		ID:           NewUUID("d"),
		Size:         NewSize(),
		Metrics:      NewMetrics(),
		Data:         map[string]string{},
	}
}
