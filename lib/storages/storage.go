package storages

import (
	"github.com/pescuma/archer/lib/model"
)

type Storage interface {
	LoadConfig() (*map[string]string, error)
	WriteConfig() error

	LoadProjects() (*model.Projects, error)
	WriteProjects() error
	WriteProject(proj *model.Project) error

	LoadFiles() (*model.Files, error)
	WriteFiles() error
	WriteFile(file *model.File) error

	LoadFileContents(fileID model.ID) (*model.FileContents, error)
	WriteFileContents(contents *model.FileContents) error
	QueryBlamePerAuthor() ([]*BlamePerAuthor, error)

	LoadPeople() (*model.People, error)
	WritePeople() error
	LoadPeopleRelations() (*model.PeopleRelations, error)
	WritePeopleRelations() error

	LoadRepositories() (*model.Repositories, error)
	WriteRepositories() error
	WriteRepository(repo *model.Repository) error
	WriteCommit(repo *model.Repository, commit *model.RepositoryCommit) error
	LoadRepositoryCommitFiles(repo *model.Repository, commit *model.RepositoryCommit) (*model.RepositoryCommitFiles, error)
	WriteRepositoryCommitFiles(files []*model.RepositoryCommitFiles) error
	QueryCommits(file string, proj string, repo string, person string) ([]model.UUID, error)

	LoadMonthlyStats() (*model.MonthlyStats, error)
	WriteMonthlyStats() error

	LoadIgnoreRules() (*model.IgnoreRules, error)
	WriteIgnoreRules() error

	Close() error
}

type Factory = func(path string) (Storage, error)

type BlamePerAuthor struct {
	AuthorID     model.UUID
	CommitterID  model.UUID
	RepositoryID model.UUID
	CommitID     model.UUID
	FileID       model.ID
	LineType     model.FileLineType
	Lines        int
}
