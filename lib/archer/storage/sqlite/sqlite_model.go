package sqlite

import (
	"time"

	"github.com/pescuma/archer/lib/archer/model"
)

type sqlConfig struct {
	Key   string `gorm:"primaryKey"`
	Value string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlProject struct {
	ID          model.UUID
	Name        string
	Root        string   `gorm:"index:idx_projects_name"`
	ProjectName string   `gorm:"index:idx_projects_name"`
	NameParts   []string `gorm:"serializer:json"`
	Type        model.ProjectType

	RootDir     string
	ProjectFile string

	Size      *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Sizes     map[string]*sqlSize  `gorm:"serializer:json"`
	Changes   *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics   *sqlMetricsAggregate `gorm:"embedded"`
	Data      map[string]string    `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

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

	Versions []string          `gorm:"serializer:json"`
	Data     map[string]string `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlProjectDirectory struct {
	ID        model.UUID
	ProjectID model.UUID `gorm:"index"`
	Name      string
	Type      model.ProjectDirectoryType

	Size      *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes   *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics   *sqlMetricsAggregate `gorm:"embedded"`
	Data      map[string]string    `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

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

	ProductAreaID *model.UUID `gorm:"index"`

	Exists    bool
	Size      *sqlSize          `gorm:"embedded;embeddedPrefix:size_"`
	Changes   *sqlChanges       `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics   *sqlMetrics       `gorm:"embedded"`
	Data      map[string]string `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitFiles []sqlRepositoryCommitFile `gorm:"foreignKey:FileID"`
	Lines       []sqlFileLine             `gorm:"foreignKey:FileID"`
}

type sqlFileLine struct {
	FileID model.UUID `gorm:"primaryKey;index:idx_blame"`
	Line   int        `gorm:"primaryKey"`

	AuthorID *model.UUID        `gorm:"index;index:idx_blame"`
	CommitID *model.UUID        `gorm:"index;index:idx_blame"`
	Type     model.FileLineType `gorm:"index:idx_blame"`
	Text     string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlPerson struct {
	ID   model.UUID
	Name string

	Names     []string          `gorm:"serializer:json"`
	Emails    []string          `gorm:"serializer:json"`
	Blame     *sqlSize          `gorm:"embedded;embeddedPrefix:blame_"`
	Changes   *sqlChanges       `gorm:"embedded;embeddedPrefix:changes_"`
	Data      map[string]string `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitAuthors    []sqlRepositoryCommit `gorm:"foreignKey:AuthorID"`
	CommitCommitters []sqlRepositoryCommit `gorm:"foreignKey:CommitterID"`
	FileLineAuthors  []sqlFileLine         `gorm:"foreignKey:AuthorID"`
}

type sqlProductArea struct {
	ID   model.UUID
	Name string

	Size    *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetricsAggregate `gorm:"embedded"`
	Data    map[string]string    `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlRepository struct {
	ID      model.UUID
	Name    string
	RootDir string `gorm:"uniqueIndex"`
	VCS     string

	CommitsTotal int
	FilesTotal   *int
	FilesHead    *int

	Data      map[string]string `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	Commits     []sqlRepositoryCommit     `gorm:"foreignKey:RepositoryID"`
	CommitFiles []sqlRepositoryCommitFile `gorm:"foreignKey:RepositoryID"`
	Files       []sqlFile                 `gorm:"foreignKey:RepositoryID"`
}

type sqlRepositoryCommit struct {
	ID            model.UUID
	RepositoryID  model.UUID `gorm:"index"`
	Name          string
	Message       string
	Parents       []model.UUID `gorm:"serializer:json"`
	Children      []model.UUID `gorm:"serializer:json"`
	Date          time.Time    `gorm:"index"`
	CommitterID   model.UUID   `gorm:"index"`
	DateAuthored  time.Time
	AuthorID      model.UUID
	FilesModified *int
	FilesCreated  *int
	FilesDeleted  *int
	LinesModified *int
	LinesAdded    *int
	LinesDeleted  *int
	LinesSurvived *int
	Ignore        bool

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitFiles []sqlRepositoryCommitFile `gorm:"foreignKey:CommitID"`
	Lines       []sqlFileLine             `gorm:"foreignKey:CommitID"`
}

type sqlRepositoryCommitFile struct {
	CommitID      model.UUID `gorm:"primaryKey"`
	FileID        model.UUID `gorm:"primaryKey"`
	OldFileIDs    string
	RepositoryID  model.UUID `gorm:"index"`
	LinesAdded    *int
	LinesModified *int
	LinesDeleted  *int
	LinesSurvived *int

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlSize struct {
	Lines *int
	Files *int
	Bytes *int
	Other map[string]int `gorm:"serializer:json"`
}

type sqlMetrics struct {
	DependenciesGuice    *int
	Abstracts            *int
	ComplexityCyclomatic *int
	ComplexityCognitive  *int
	ComplexityFocus      *int
}

type sqlMetricsAggregate struct {
	DependenciesGuiceTotal    *int
	DependenciesGuiceAvg      *float32
	ComplexityCyclomaticTotal *int
	ComplexityCyclomaticAvg   *float32
	ComplexityCognitiveTotal  *int
	ComplexityCognitiveAvg    *float32
	ComplexityFocusTotal      *int
	ComplexityFocusAvg        *float32
}

type sqlChanges struct {
	Semester      *int
	Total         *int
	LinesModified *int
	LinesAdded    *int
	LinesDeleted  *int
}
