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

	LoadFiles() (*model.Files, error)
	WriteFiles(files *model.Files) error
	WriteFile(file *model.File) error

	LoadFileContents(fileID model.UUID) (*model.FileContents, error)
	WriteFileContents(contents *model.FileContents) error
	QueryBlamePerAuthor() ([]*BlamePerAuthor, error)

	LoadPeople() (*model.People, error)
	WritePeople(people *model.People) error
	LoadPeopleRelations() (*model.PeopleRelations, error)
	WritePeopleRelations(prs *model.PeopleRelations) error

	LoadRepositories() (*model.Repositories, error)
	LoadRepository(rootDir string) (*model.Repository, error)
	WriteRepositories(repos *model.Repositories) error
	WriteRepository(repo *model.Repository) error
	WriteCommit(repo *model.Repository, commit *model.RepositoryCommit) error
	LoadRepositoryCommitFiles(repo *model.Repository, commit *model.RepositoryCommit) (*model.RepositoryCommitFiles, error)
	WriteRepositoryCommitFiles(files []*model.RepositoryCommitFiles) error
	QueryCommits(file string, proj string, repo string, person string) ([]model.UUID, error)

	LoadMonthlyStats() (*model.MonthlyStats, error)
	WriteMonthlyStats(stats *model.MonthlyStats) error

	LoadConfig() (*map[string]string, error)
	WriteConfig(*map[string]string) error
}

type StorageFactory = func(path string) (Storage, error)

type BlamePerAuthor struct {
	AuthorID     model.UUID
	CommitterID  model.UUID
	RepositoryID model.UUID
	CommitID     model.UUID
	FileID       model.UUID
	LineType     model.FileLineType
	Lines        int
}
