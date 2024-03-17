package orm

import (
	"strconv"
	"strings"
	"time"

	"github.com/pescuma/archer/lib/model"
)

type sqlTable interface {
	CacheKey() string
}

type sqlConfig struct {
	Key   string `gorm:"primaryKey"`
	Value string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *sqlConfig) CacheKey() string {
	return s.Key
}

type sqlProject struct {
	ID          model.UUID
	Name        string
	ProjectName string   `gorm:"index"`
	Groups      []string `gorm:"serializer:json"`
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

func (s *sqlProject) CacheKey() string {
	return string(s.ID)
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

func (s *sqlProjectDependency) CacheKey() string {
	return string(s.ID)
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

func (s *sqlProjectDirectory) CacheKey() string {
	return string(s.ID)
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

func (s *sqlFile) CacheKey() string {
	return string(s.ID)
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

	Month        string
	RepositoryID model.UUID
	AuthorID     model.UUID
	CommitterID  model.UUID
	ProjectID    *model.UUID

	Changes *sqlChanges `gorm:"embedded;embeddedPrefix:changes_"`
	Blame   *sqlBlame   `gorm:"embedded;embeddedPrefix:blame_"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *sqlMonthLines) CacheKey() string {
	return string(s.ID)
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

	Commits      []sqlRepositoryCommitPerson `gorm:"foreignKey:PersonID"`
	Repositories []sqlPersonRepository       `gorm:"foreignKey:PersonID"`
}

func (s *sqlPerson) CacheKey() string {
	return string(s.ID)
}

type sqlPersonRepository struct {
	PersonID     model.UUID `gorm:"primaryKey"`
	RepositoryID model.UUID `gorm:"primaryKey"`

	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *sqlPersonRepository) CacheKey() string {
	return compositeKey(string(s.PersonID), string(s.RepositoryID))
}

type sqlPersonFile struct {
	PersonID model.UUID `gorm:"primaryKey"`
	FileID   model.UUID `gorm:"primaryKey"`

	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *sqlPersonFile) CacheKey() string {
	return compositeKey(string(s.PersonID), string(s.FileID))
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

func (s *sqlProductArea) CacheKey() string {
	return string(s.ID)
}

type sqlRepository struct {
	ID      model.UUID
	Name    string
	RootDir string `gorm:"uniqueIndex"`
	VCS     string
	Branch  string

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

func (s *sqlRepository) CacheKey() string {
	return string(s.ID)
}

type sqlRepositoryCommit struct {
	ID           model.UUID
	RepositoryID model.UUID `gorm:"index"`
	Name         string
	Message      string
	Parents      []model.UUID `gorm:"serializer:json"`
	Children     []model.UUID `gorm:"serializer:json"`
	Date         time.Time    `gorm:"index"`
	DateAuthored time.Time
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

	People []sqlRepositoryCommitPerson `gorm:"foreignKey:CommitID"`
	Files  []sqlRepositoryCommitFile   `gorm:"foreignKey:CommitID"`
}

func (s *sqlRepositoryCommit) CacheKey() string {
	return string(s.ID)
}

type CommitRole int

const (
	CommitRoleAuthor    CommitRole = iota
	CommitRoleCommitter CommitRole = iota
)

func (r CommitRole) String() string {
	return strconv.Itoa(int(r))
}

type sqlRepositoryCommitPerson struct {
	CommitID model.UUID `gorm:"primaryKey"`
	PersonID model.UUID `gorm:"primaryKey"`
	Role     CommitRole `gorm:"primaryKey"`
	Order    int

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *sqlRepositoryCommitPerson) CacheKey() string {
	return compositeKey(string(s.CommitID), string(s.PersonID), s.Role.String())
}

type sqlRepositoryCommitFile struct {
	CommitID      model.UUID `gorm:"primaryKey"`
	FileID        model.UUID `gorm:"primaryKey"`
	Hash          string
	Change        model.FileChangeType
	OldIDs        string
	OldHashes     string
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

func compositeKey(ids ...string) string {
	return strings.Join(ids, "\n")
}
