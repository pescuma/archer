package sqlite

import (
	"time"

	"github.com/Faire/archer/lib/archer/model"
)

type sqlProject struct {
	ID        model.UUID
	Root      string   `gorm:"index:idx_projects_name"`
	Name      string   `gorm:"index:idx_projects_name"`
	NameParts []string `gorm:"serializer:json"`
	Type      model.ProjectType

	RootDir     string
	ProjectFile string

	Size  *sqlSize            `gorm:"embedded;embeddedPrefix:size_"`
	Sizes map[string]*sqlSize `gorm:"serializer:json"`
	Data  map[string]string   `gorm:"serializer:json"`

	Dependencies []sqlProjectDependency `gorm:"foreignKey:SourceID;foreignKey:TargetID"`
	Dirs         []sqlProjectDirectory  `gorm:"foreignKey:ProjectID"`
	Files        []sqlFile              `gorm:"foreignKey:ProjectID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlProjectDependency struct {
	ID       model.UUID
	SourceID model.UUID `gorm:"index"`
	TargetID model.UUID `gorm:"index"`

	Data map[string]string `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlProjectDirectory struct {
	ID           model.UUID
	ProjectID    model.UUID `gorm:"index"`
	RelativePath string
	Type         model.ProjectDirectoryType

	Size *sqlSize          `gorm:"embedded;embeddedPrefix:size_"`
	Data map[string]string `gorm:"serializer:json"`

	Files []sqlFile `gorm:"foreignKey:ProjectDirectoryID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlFile struct {
	ID   model.UUID
	Path string

	ProjectID          *model.UUID `gorm:"index"`
	ProjectDirectoryID *model.UUID `gorm:"index"`

	Size *sqlSize          `gorm:"embedded;embeddedPrefix:size_"`
	Data map[string]string `gorm:"serializer:json"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type sqlSize struct {
	Lines int
	Files int
	Bytes int
	Other map[string]int `gorm:"serializer:json"`
}
