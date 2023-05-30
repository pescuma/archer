package model

type ProjectDirectory struct {
	RelativePath string
	Type         ProjectDirectoryType
	ID           UUID

	Size    *Size
	Changes *Changes
	Metrics *Metrics
	Data    map[string]string
}

func NewProjectDirectory(relativePath string) *ProjectDirectory {
	return &ProjectDirectory{
		RelativePath: relativePath,
		ID:           NewUUID("d"),
		Size:         NewSize(),
		Changes:      NewChanges(),
		Metrics:      NewMetrics(),
		Data:         map[string]string{},
	}
}
