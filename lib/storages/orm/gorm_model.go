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

type sqlProjectDirectory struct {
	ID        model.UUID
	ProjectID model.ID `gorm:"index"`
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

type sqlPersonRepository struct {
	PersonID     model.ID   `gorm:"primaryKey"`
	RepositoryID model.UUID `gorm:"primaryKey"`

	FirstSeen time.Time
	LastSeen  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *sqlPersonRepository) CacheKey() string {
	return compositeKey(s.PersonID.String(), string(s.RepositoryID))
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
	PersonID model.ID   `gorm:"primaryKey"`
	Role     CommitRole `gorm:"primaryKey"`
	Order    int

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *sqlRepositoryCommitPerson) CacheKey() string {
	return compositeKey(string(s.CommitID), s.PersonID.String(), s.Role.String())
}

type sqlRepositoryCommitFile struct {
	CommitID      model.UUID `gorm:"primaryKey"`
	FileID        model.ID   `gorm:"primaryKey"`
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
