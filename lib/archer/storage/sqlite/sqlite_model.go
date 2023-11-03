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

	Size    *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Sizes   map[string]*sqlSize  `gorm:"serializer:json"`
	Changes *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetricsAggregate `gorm:"embedded"`
	Data    map[string]string    `gorm:"serializer:json"`

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

	Size    *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetricsAggregate `gorm:"embedded"`
	Data    map[string]string    `gorm:"serializer:json"`

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
	OrgID         *model.UUID `gorm:"index"`
	OrgGroupID    *model.UUID `gorm:"index"`
	OrgTeamID     *model.UUID `gorm:"index"`

	Exists  bool
	Size    *sqlSize          `gorm:"embedded;embeddedPrefix:size_"`
	Changes *sqlChanges       `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetrics       `gorm:"embedded"`
	Data    map[string]string `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitFiles    []sqlRepositoryCommitFile `gorm:"foreignKey:FileID"`
	CommitOldFiles []sqlRepositoryCommitFile `gorm:"foreignKey:OldFileID"`
}

type sqlPerson struct {
	ID   model.UUID
	Name string

	Names   []string             `gorm:"serializer:json"`
	Emails  []string             `gorm:"serializer:json"`
	Size    *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetricsAggregate `gorm:"embedded"`
	Data    map[string]string    `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitAuthored []sqlRepositoryCommit `gorm:"foreignKey:AuthorID"`
	CommitCommited []sqlRepositoryCommit `gorm:"foreignKey:CommitterID"`
	TeamMembers    []sqlTeamMember       `gorm:"foreignKey:PersonID"`
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

	TeamAreas []sqlTeamArea `gorm:"foreignKey:CodeAreaID"`
}

type sqlOrg struct {
	ID   model.UUID
	Name string

	Size    *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetricsAggregate `gorm:"embedded"`
	Data    map[string]string    `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Groups      []sqlOrgGroup   `gorm:"foreignKey:OrgID"`
	Teams       []sqlOrgTeam    `gorm:"foreignKey:OrgID"`
	TeamMembers []sqlTeamMember `gorm:"foreignKey:OrgID"`
	TeamAreas   []sqlTeamArea   `gorm:"foreignKey:OrgID"`
}

type sqlOrgGroup struct {
	ID    model.UUID
	Name  string
	OrgID model.UUID

	Size    *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetricsAggregate `gorm:"embedded"`
	Data    map[string]string    `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Teams       []sqlOrgTeam    `gorm:"foreignKey:OrgGroupID"`
	TeamMembers []sqlTeamMember `gorm:"foreignKey:OrgGroupID"`
	TeamAreas   []sqlTeamArea   `gorm:"foreignKey:OrgGroupID"`
}

type sqlOrgTeam struct {
	ID         model.UUID
	Name       string
	OrgGroupID model.UUID
	OrgID      model.UUID

	Size    *sqlSize             `gorm:"embedded;embeddedPrefix:size_"`
	Changes *sqlChanges          `gorm:"embedded;embeddedPrefix:changes_"`
	Metrics *sqlMetricsAggregate `gorm:"embedded"`
	Data    map[string]string    `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time

	TeamMembers []sqlTeamMember `gorm:"foreignKey:OrgTeamID"`
	TeamAreas   []sqlTeamArea   `gorm:"foreignKey:OrgTeamID"`
}

type sqlTeamMember struct {
	ID         model.UUID
	PersonID   model.UUID `gorm:"index"`
	OrgTeamID  model.UUID `gorm:"index"`
	OrgGroupID model.UUID `gorm:"index"`
	OrgID      model.UUID `gorm:"index"`

	Start *time.Time
	End   *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlTeamArea struct {
	ID         model.UUID
	CodeAreaID model.UUID `gorm:"index"`
	OrgTeamID  model.UUID `gorm:"index"`
	OrgGroupID model.UUID `gorm:"index"`
	OrgID      model.UUID `gorm:"index"`

	Start *time.Time
	End   *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
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
	Files       []sqlFile                 `gorm:"foreignKey:RepositoryID"`
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
	AddedLines    int
	ModifiedLines int
	DeletedLines  int

	CreatedAt time.Time
	UpdatedAt time.Time

	CommitFiles []sqlRepositoryCommitFile `gorm:"foreignKey:CommitID"`
}

type sqlRepositoryCommitFile struct {
	CommitID      model.UUID `gorm:"primaryKey"`
	FileID        model.UUID `gorm:"primaryKey"`
	OldFileID     *model.UUID
	RepositoryID  model.UUID `gorm:"index"`
	AddedLines    int        // TODO *
	ModifiedLines int
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
	ModifiedLines *int
	AddedLines    *int
	DeletedLines  *int
}
