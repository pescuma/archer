package storage

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
)

type sqliteStorage struct {
	db *gorm.DB
}

func NewSqliteStorage(file string) (archer.Storage, error) {
	if strings.HasSuffix(file, "/") {
		file = file + "archer.db"
	}

	if _, err := os.Stat(file); err != nil {
		fmt.Printf("Creating workspace at %v\n", file)
		root := path.Base(file)
		err = os.MkdirAll(root, 0o700)
		if err != nil {
			return nil, err
		}
	}

	db, err := gorm.Open(sqlite.Open(file), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &sqliteStorage{
		db: db,
	}, nil
}

func (s *sqliteStorage) LoadProjects(result *model.Projects) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WriteProjNames(projRoot string, projNames []string) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) ReadProjNames() ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WriteDeps(proj *model.Project) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) ReadDeps(result *model.Projects, fileName string) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WriteSize(proj *model.Project) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) ReadSize(result *model.Projects, fileName string) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WriteBasicInfo(proj *model.Project) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) ReadBasicInfo(result *model.Projects, fileName string) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WriteFiles(proj *model.Project) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) ReadFiles(result *model.Projects, fileName string) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WriteConfig(proj *model.Project) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) ReadConfig(result *model.Projects, fileName string) error {
	// TODO implement me
	panic("implement me")
}
