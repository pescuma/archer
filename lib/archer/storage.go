package archer

import (
	"github.com/Faire/archer/lib/archer/model"
)

type StorageChanges uint16

const (
	ChangedBasicInfo StorageChanges = 1 << iota
	ChangedConfig
	ChangedDependencies
	ChangedSize

	ChangedAll = 0xffff
)

type Storage interface {
	LoadProjects(result *model.Projects) error
	WriteProjects(projs *model.Projects, changes StorageChanges) error
	WriteProject(proj *model.Project, changes StorageChanges) error

	LoadFiles(result *model.Files) error
	WriteFiles(files *model.Files, changes StorageChanges) error
}

type StorageFactory = func(path string) (Storage, error)
