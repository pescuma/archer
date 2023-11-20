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
	WriteProjects(projs *model.Projects) error
	WriteProject(proj *model.Project) error
	QueryProjects(file string, proj string, repo string, person string) ([]model.UUID, error)

	LoadFiles() (*model.Files, error)
	WriteFiles(files *model.Files) error
	WriteFile(file *model.File) error

	LoadFileContents(fileID model.UUID) (*model.FileContents, error)
	WriteFileContents(contents *model.FileContents) error
	QueryBlamePerAuthor() ([]*BlamePerAuthor, error)
	QueryFiles(file string, proj string, repo string, person string) ([]model.UUID, error)

	LoadPeople() (*model.People, error)
	WritePeople(people *model.People) error
	LoadPeopleRepositories() (*model.PeopleRepositories, error)
	WritePeopleRepositories(prs *model.PeopleRepositories) error
	QueryPeople(file string, proj string, repo string, person string) ([]model.UUID, error)

	LoadRepositories() (*model.Repositories, error)
	LoadRepository(rootDir string) (*model.Repository, error)
	WriteRepositories(repos *model.Repositories) error
	WriteRepository(repo *model.Repository) error
	WriteCommit(repo *model.Repository, commit *model.RepositoryCommit) error
	LoadRepositoryCommitFiles(repo *model.Repository, commit *model.RepositoryCommit) (*model.RepositoryCommitFiles, error)
	WriteRepositoryCommitFiles(files []*model.RepositoryCommitFiles) error
	QueryRepositories(file string, proj string, repo string, person string) ([]model.UUID, error)
	QueryCommits(file string, proj string, repo string, person string) ([]model.UUID, error)
	QuerySurvivedLines(file string, proj string, repo string, person string) ([]*SurvivedLineCount, error)
	WriteSurvivedLinesCache() error

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

type SurvivedLineCount struct {
	Month    string
	LineType model.FileLineType
	Lines    int
}
