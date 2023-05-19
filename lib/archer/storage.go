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
	ChangedHistory

	ChangedAll = 0xffff
)

type Storage interface {
	LoadProjects() (*model.Projects, error)
	WriteProjects(projs *model.Projects, changes StorageChanges) error
	WriteProject(proj *model.Project, changes StorageChanges) error

	LoadFiles() (*model.Files, error)
	WriteFiles(files *model.Files, changes StorageChanges) error

	LoadRepositories() (*model.Repositories, error)
	WriteRepository(repo *model.Repository, changes StorageChanges) error

	LoadPeople() (*model.People, error)
	WritePeople(people *model.People, changes StorageChanges) error
}

type StorageFactory = func(path string) (Storage, error)
