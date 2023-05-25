package archer

import (
	"github.com/Faire/archer/lib/archer/model"
)

type StorageChanges uint16

const (
	ChangedBasicInfo StorageChanges = 1 << iota
	ChangedData
	ChangedDependencies
	ChangedSize
	ChangedHistory
	ChangedMetrics

	ChangedAll = 0xffff
)

type Storage interface {
	LoadProjects() (*model.Projects, error)
	WriteProjects(projs *model.Projects, changes StorageChanges) error
	WriteProject(proj *model.Project, changes StorageChanges) error

	LoadFiles() (*model.Files, error)
	WriteFiles(files *model.Files, changes StorageChanges) error

	LoadPeople() (*model.People, error)
	WritePeople(people *model.People, changes StorageChanges) error

	LoadRepositories() (repos *model.Repositories, err error)
	LoadRepository(rootDir string) (*model.Repository, error)
	WriteRepository(repo *model.Repository, changes StorageChanges) error
}

type StorageFactory = func(path string) (Storage, error)
