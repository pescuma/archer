package archer

import (
	"github.com/pescuma/archer/lib/archer/model"
)

type StorageChanges uint16

const (
	ChangedBasicInfo StorageChanges = 1 << iota
	ChangedData
	ChangedDependencies
	ChangedHistory
	ChangedTeams
	ChangedSize
	ChangedMetrics
	ChangedChanges

	ChangedAll = 0xffff
)

type Storage interface {
	LoadProjects() (*model.Projects, error)
	WriteProjects(projs *model.Projects, changes StorageChanges) error
	WriteProject(proj *model.Project, changes StorageChanges) error

	LoadFiles() (*model.Files, error)
	WriteFiles(files *model.Files, changes StorageChanges) error
	WriteFile(file *model.File, changes StorageChanges) error

	LoadFileContents(fileID model.UUID) (*model.FileContents, error)
	WriteFileContents(contents *model.FileContents, changes StorageChanges) error
	ComputeBlamePerAuthor() ([]*BlamePerAuthor, error)

	LoadPeople() (*model.People, error)
	WritePeople(people *model.People, changes StorageChanges) error

	LoadRepositories() (repos *model.Repositories, err error)
	LoadRepository(rootDir string) (*model.Repository, error)
	WriteRepository(repo *model.Repository, changes StorageChanges) error

	LoadConfig() (*map[string]string, error)
	WriteConfig(*map[string]string) error
}

type StorageFactory = func(path string) (Storage, error)

type BlamePerAuthor struct {
	AuthorID model.UUID
	CommitID model.UUID
	FileID   model.UUID
	LineType model.FileLineType
	Lines    int
}
