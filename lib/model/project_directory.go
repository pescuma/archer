package model

type ProjectDirectory struct {
	RelativePath string
	Type         ProjectDirectoryType
	ID           ID

	Size    *Size
	Changes *Changes
	Metrics *Metrics
	SeenAt  *SeenAt
	Data    map[string]string
}

func NewProjectDirectory(id ID, relativePath string) *ProjectDirectory {
	return &ProjectDirectory{
		RelativePath: relativePath,
		ID:           id,
		Size:         NewSize(),
		Changes:      NewChanges(),
		Metrics:      NewMetrics(),
		SeenAt:       NewSeenAt(),
		Data:         map[string]string{},
	}
}
