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

	RepositoryID *model.UUID `gorm:"index"`

	Sizes     map[string]*sqlSize  `gorm:"serializer:json"`
	Size      *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
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
	People      []sqlPersonFile           `gorm:"foreignKey:FileID"`
}

type sqlFileLine struct {
	FileID model.UUID `gorm:"primaryKey"`
	Line   int        `gorm:"primaryKey"`

	ProjectID    *model.UUID
	RepositoryID *model.UUID
	CommitID     *model.UUID
	AuthorID     *model.UUID
	CommitterID  *model.UUID
	Date         time.Time

	Type model.FileLineType
	Text string
}

type sqlMonthLines struct {
	ID model.UUID `gorm:"primaryKey"`

	Month        string      `gorm:"index"`
	RepositoryID model.UUID  `gorm:"index"`
	AuthorID     model.UUID  `gorm:"index"`
	CommitterID  model.UUID  `gorm:"index"`
	ProjectID    *model.UUID `gorm:"index"`

	Changes *sqlChanges `gorm:"embedded;embeddedPrefix:changes_"`
	Blame   *sqlBlame   `gorm:"embedded;embeddedPrefix:blame_"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlPerson struct {
	ID   model.UUID
	Name string

	Names     []string          `gorm:"serializer:json"`
	Emails    []string          `gorm:"serializer:json"`
	Changes   *sqlChanges       `gorm:"embedded;embeddedPrefix:changes_"`
	Blame     *sqlBlame         `gorm:"embedded;embeddedPrefix:blame_"`
	Data      map[string]string `gorm:"serializer:json"`
	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitAuthors    []sqlRepositoryCommit `gorm:"foreignKey:AuthorID"`
	CommitCommitters []sqlRepositoryCommit `gorm:"foreignKey:CommitterID"`
	Repositories     []sqlPersonRepository `gorm:"foreignKey:PersonID"`
}

type sqlPersonRepository struct {
	PersonID     model.UUID `gorm:"primaryKey"`
	RepositoryID model.UUID `gorm:"primaryKey"`

	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlPersonFile struct {
	PersonID model.UUID `gorm:"primaryKey"`
	FileID   model.UUID `gorm:"primaryKey"`

	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
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

	Commits []sqlRepositoryCommit `gorm:"foreignKey:RepositoryID"`
	Files   []sqlFile             `gorm:"foreignKey:RepositoryID"`
	People  []sqlPersonRepository `gorm:"foreignKey:RepositoryID"`
}

type sqlRepositoryCommit struct {
	ID           model.UUID
	RepositoryID model.UUID `gorm:"index"`
	Name         string
	Message      string
	Parents      []model.UUID `gorm:"serializer:json"`
	Children     []model.UUID `gorm:"serializer:json"`
	Date         time.Time    `gorm:"index"`
	CommitterID  model.UUID   `gorm:"index"`
	DateAuthored time.Time
	AuthorID     model.UUID
	Ignore       bool

	FilesModified *int
	FilesCreated  *int
	FilesDeleted  *int
	LinesModified *int
	LinesAdded    *int
	LinesDeleted  *int
	Blame         *sqlBlame `gorm:"embedded;embeddedPrefix:blame_"`

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitFiles []sqlRepositoryCommitFile `gorm:"foreignKey:CommitID"`
}

type sqlRepositoryCommitFile struct {
	CommitID      model.UUID `gorm:"primaryKey"`
	FileID        model.UUID `gorm:"primaryKey"`
	Hash          string
	Change        model.FileChangeType
	OldFileIDs    string
	LinesModified *int
	LinesAdded    *int
	LinesDeleted  *int
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

type sqlBlame struct {
	Code    *int
	Comment *int
	Blank   *int
}
