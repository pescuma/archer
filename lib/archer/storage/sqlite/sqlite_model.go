package sqlite

import (
	"time"

	"github.com/Faire/archer/lib/archer/model"
)

type sqlProject struct {
	ID          model.UUID
	Name        string
	Root        string   `gorm:"index:idx_projects_name"`
	ProjectName string   `gorm:"index:idx_projects_name"`
	NameParts   []string `gorm:"serializer:json"`
	Type        model.ProjectType

	RootDir     string
	ProjectFile string

	Size    *sqlSize            `gorm:"embedded;embeddedPrefix:size_"`
	Sizes   map[string]*sqlSize `gorm:"serializer:json"`
	Metrics *sqlMetrics         `gorm:"embedded"`
	Data    map[string]string   `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time

	DependencySources []sqlProjectDependency `gorm:"foreignKey:SourceID"`
	DependencyTargets []sqlProjectDependency `gorm:"foreignKey:TargetID"`
	Dirs              []sqlProjectDirectory  `gorm:"foreignKey:ProjectID"`
	Files             []sqlFile              `gorm:"foreignKey:ProjectID"`
}

type sqlProjectDependency struct {
	ID       model.UUID
	Name     string
	SourceID model.UUID `gorm:"index"`
	TargetID model.UUID `gorm:"index"`

	Data map[string]string `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlProjectDirectory struct {
	ID        model.UUID
	ProjectID model.UUID `gorm:"index"`
	Name      string
	Type      model.ProjectDirectoryType

	Size    *sqlSize          `gorm:"embedded;embeddedPrefix:size_"`
	Metrics *sqlMetrics       `gorm:"embedded"`
	Data    map[string]string `gorm:"serializer:json"`

	Files []sqlFile `gorm:"foreignKey:ProjectDirectoryID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlFile struct {
	ID   model.UUID
	Name string

	ProjectID          *model.UUID `gorm:"index"`
	ProjectDirectoryID *model.UUID `gorm:"index"`
	RepositoryID       *model.UUID `gorm:"index"`

	Exists  bool
	Size    *sqlSize          `gorm:"embedded;embeddedPrefix:size_"`
	Metrics *sqlMetrics       `gorm:"embedded"`
	Data    map[string]string `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitFiles []sqlRepositoryCommitFile `gorm:"foreignKey:FileID"`
}

type sqlPerson struct {
	ID   model.UUID
	Name string

	Names  []string          `gorm:"serializer:json"`
	Emails []string          `gorm:"serializer:json"`
	Data   map[string]string `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitAuthored []sqlRepositoryCommit `gorm:"foreignKey:AuthorID"`
	CommitCommited []sqlRepositoryCommit `gorm:"foreignKey:CommitterID"`
}

type sqlRepository struct {
	ID      model.UUID
	Name    string
	RootDir string `gorm:"uniqueIndex"`
	VCS     string

	Data map[string]string `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Commits     []sqlRepositoryCommit     `gorm:"foreignKey:RepositoryID"`
	CommitFiles []sqlRepositoryCommitFile `gorm:"foreignKey:RepositoryID"`
}

type sqlRepositoryCommit struct {
	ID            model.UUID
	RepositoryID  model.UUID `gorm:"index"`
	Name          string
	Message       string
	Parents       []string   `gorm:"serializer:json"`
	Date          time.Time  `gorm:"index"`
	CommitterID   model.UUID `gorm:"index"`
	DateAuthored  time.Time
	AuthorID      model.UUID
	ModifiedLines int
	AddedLines    int
	DeletedLines  int

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitFiles []sqlRepositoryCommitFile `gorm:"foreignKey:CommitID"`
}

type sqlRepositoryCommitFile struct {
	CommitID      model.UUID `gorm:"primaryKey"`
	FileID        model.UUID `gorm:"primaryKey"`
	RepositoryID  model.UUID `gorm:"index"`
	ModifiedLines int
	AddedLines    int
	DeletedLines  int

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlSize struct {
	Lines int
	Files int
	Bytes int
	Other map[string]int `gorm:"serializer:json"`
}

type sqlMetrics struct {
	DependenciesGuice    *int
	ComplexityCyclomatic *int
	ComplexityCognitive  *int
	Changes6Months       *int
	ChangesTotal         *int
}
