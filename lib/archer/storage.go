package archer

import (
	"github.com/Faire/archer/lib/archer/model"
)

type StorageChanges uint16

const (
	ChangedProjectBasicInfo StorageChanges = 1 << iota
	ChangedProjectConfig
	ChangedProjectDependencies
	ChangedProjectFiles
	ChangedProjectSize

	ChangedAll = 0xffff
)

type Storage interface {
	LoadProjects(result *model.Projects) error
	WriteProjects(projs *model.Projects, changes StorageChanges) error
	WriteProject(proj *model.Project, changes StorageChanges) error
}

type StorageFactory = func(path string) (Storage, error)
