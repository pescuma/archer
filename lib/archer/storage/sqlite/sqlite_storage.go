package sqlite

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
)

type sqliteStorage struct {
	db *gorm.DB
}

func NewSqliteStorage(file string) (archer.Storage, error) {
	if strings.HasSuffix(file, string(filepath.Separator)) {
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

	db, err := gorm.Open(sqlite.Open(file), &gorm.Config{
		NamingStrategy: &NamingStrategy{},
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&sqlProject{}, &sqlProjectDependency{}, &sqlProjectDirectory{},
		&sqlFile{},
	)
	if err != nil {
		return nil, err
	}

	return &sqliteStorage{
		db: db,
	}, nil
}

func (s *sqliteStorage) LoadProjects() (*model.Projects, error) {
	result := model.NewProjects()

	var projs []*sqlProject
	err := s.db.Find(&projs).Error
	if err != nil {
		return nil, err
	}

	var deps []*sqlProjectDependency
	err = s.db.Find(&deps).Error
	if err != nil {
		return nil, err
	}

	var dirs []*sqlProjectDirectory
	err = s.db.Find(&dirs).Error
	if err != nil {
		return nil, err
	}

	projsByID := map[model.UUID]*model.Project{}

	for _, sp := range projs {
		p := result.Get(sp.Root, sp.Name)
		p.ID = sp.ID
		p.NameParts = sp.NameParts
		p.Type = sp.Type

		p.RootDir = sp.RootDir
		p.ProjectFile = sp.ProjectFile

		for k, v := range sp.Sizes {
			p.Sizes[k] = toModelSize(v)
		}
		p.Data = sp.Data

		projsByID[p.ID] = p
	}

	for _, sd := range deps {
		source := projsByID[sd.SourceID]
		target := projsByID[sd.TargetID]

		d := source.GetDependency(target)
		d.ID = sd.ID
		d.Data = sd.Data
	}

	for _, sd := range dirs {
		p := projsByID[sd.ProjectID]

		d := p.GetDirectory(sd.RelativePath)
		d.ID = sd.ID
		d.Type = sd.Type
		d.Size = toModelSize(sd.Size)
		d.Data = sd.Data
	}

	return result, nil
}

func (s *sqliteStorage) WriteProjects(projs *model.Projects, changes archer.StorageChanges) error {
	all := projs.ListProjects(model.FilterAll)

	return s.writeProjects(all, changes,
		func(string) func(db *gorm.DB) *gorm.DB {
			return func(db *gorm.DB) *gorm.DB {
				return db
			}
		})
}

func (s *sqliteStorage) WriteProject(proj *model.Project, changes archer.StorageChanges) error {
	projs := []*model.Project{proj}

	return s.writeProjects(projs, changes,
		func(projectIDField string) func(db *gorm.DB) *gorm.DB {
			return func(db *gorm.DB) *gorm.DB {
				return db.Where(projectIDField+" = ?", proj.ID)
			}
		})
}

func (s *sqliteStorage) writeProjects(projs []*model.Project, changes archer.StorageChanges, scope func(string) func(*gorm.DB) *gorm.DB) error {
	changedProjs := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedConfig != 0 || changes&archer.ChangedSize != 0
	changedDeps := changes&archer.ChangedDependencies != 0 || changes&archer.ChangedConfig != 0
	changedDirs := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedSize != 0

	sqlProjs := make([]sqlProject, 0, len(projs))
	if changedProjs {
		for _, p := range projs {
			sqlProjs = append(sqlProjs, toSqlProject(p))
		}
	}

	var sqlDeps []sqlProjectDependency
	if changedDeps {
		for _, p := range projs {
			for _, d := range p.Dependencies {
				sqlDeps = append(sqlDeps, toSqlProjectDependency(d))
			}
		}
	}

	var sqlDirs []sqlProjectDirectory
	if changedDirs {
		for _, p := range projs {
			for _, dir := range p.Dirs {
				sqlDirs = append(sqlDirs, toSqlProjectDirectory(dir, p))
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 100,
	})

	if changedProjs {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlProjs).Error
		if err != nil {
			return err
		}
	}

	if changedDeps {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDeps).Error
		if err != nil {
			return err
		}
	}

	if changedDirs {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDirs).Error
		if err != nil {
			return err
		}
	}

	if changedDirs {
		err := db.Scopes(scope("project_id")).Where("updated_at != ?", now).Delete(&sqlProjectDirectory{}).Error
		if err != nil {
			return err
		}
	}

	if changedDeps {
		err := db.Scopes(scope("source_id")).Where("updated_at != ?", now).Delete(&sqlProjectDependency{}).Error
		if err != nil {
			return err
		}
	}

	if changedProjs {
		err := db.Scopes(scope("project_id")).Where("updated_at != ?", now).Delete(&sqlProject{}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *sqliteStorage) LoadFiles() (*model.Files, error) {
	result := model.NewFiles()

	var files []*sqlFile
	err := s.db.Find(&files).Error
	if err != nil {
		return nil, err
	}

	for _, sf := range files {
		f := result.Get(sf.Path)
		f.ID = sf.ID
		f.ProjectID = sf.ProjectID
		f.ProjectDirectoryID = sf.ProjectDirectoryID
		f.Size = toModelSize(sf.Size)
		f.Data = sf.Data
	}

	return result, nil
}

func (s *sqliteStorage) WriteFiles(files *model.Files, changes archer.StorageChanges) error {
	changedFiles := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedConfig != 0 || changes&archer.ChangedSize != 0
	if !changedFiles {
		return nil
	}

	all := files.List()

	sqlFiles := make([]sqlFile, 0, len(all))
	for _, f := range all {
		sqlFiles = append(sqlFiles, toSqlFile(f))
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 100,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlFiles).Error
	if err != nil {
		return err
	}

	err = db.Where("updated_at != ?", now).Delete(&sqlFile{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *sqliteStorage) LoadRepositories() (*model.Repositories, error) {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WriteRepository(repo *model.Repository, changes archer.StorageChanges) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) LoadPeople() (*model.People, error) {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WritePeople(people *model.People, changes archer.StorageChanges) error {
	// TODO implement me
	panic("implement me")
}

func toSqlSize(size *model.Size) *sqlSize {
	return &sqlSize{
		Lines: size.Lines,
		Files: size.Files,
		Bytes: size.Bytes,
		Other: size.Other,
	}
}

func toModelSize(size *sqlSize) *model.Size {
	return &model.Size{
		Lines: size.Lines,
		Files: size.Files,
		Bytes: size.Bytes,
		Other: size.Other,
	}
}

func toSqlProject(p *model.Project) sqlProject {
	size := p.GetSize()

	sp := sqlProject{
		ID:          p.ID,
		Root:        p.Root,
		Name:        p.Name,
		NameParts:   p.NameParts,
		Type:        p.Type,
		RootDir:     p.RootDir,
		ProjectFile: p.ProjectFile,
		Size:        toSqlSize(size),
		Sizes:       map[string]*sqlSize{},
		Data:        p.Data,
	}

	for k, v := range p.Sizes {
		sp.Sizes[k] = toSqlSize(v)
	}

	return sp
}

func toSqlProjectDependency(d *model.ProjectDependency) sqlProjectDependency {
	return sqlProjectDependency{
		ID:       d.ID,
		SourceID: d.Source.ID,
		TargetID: d.Target.ID,
		Data:     d.Data,
	}
}

func toSqlProjectDirectory(d *model.ProjectDirectory, p *model.Project) sqlProjectDirectory {
	return sqlProjectDirectory{
		ID:           d.ID,
		ProjectID:    p.ID,
		RelativePath: d.RelativePath,
		Type:         d.Type,
		Size:         toSqlSize(d.Size),
		Data:         d.Data,
	}
}

func toSqlFile(f *model.File) sqlFile {
	return sqlFile{
		ID:                 f.ID,
		Path:               f.Path,
		ProjectID:          f.ProjectID,
		ProjectDirectoryID: f.ProjectDirectoryID,
		Size:               toSqlSize(f.Size),
		Data:               f.Data,
	}
}

type NamingStrategy struct {
	inner schema.NamingStrategy
}

func (n *NamingStrategy) TableName(table string) string {
	return strings.TrimPrefix(n.inner.TableName(table), "sql_")
}

func (n *NamingStrategy) SchemaName(table string) string {
	return n.inner.SchemaName(table)
}

func (n *NamingStrategy) ColumnName(table, column string) string {
	return n.inner.ColumnName(table, column)
}

func (n *NamingStrategy) JoinTableName(joinTable string) string {
	return n.inner.JoinTableName(joinTable)
}

func (n *NamingStrategy) RelationshipFKName(relationship schema.Relationship) string {
	return strings.ReplaceAll(n.inner.RelationshipFKName(relationship), "_sql_", "_")
}

func (n *NamingStrategy) CheckerName(table, column string) string {
	return strings.ReplaceAll(n.inner.CheckerName(table, column), "_sql_", "_")
}

func (n *NamingStrategy) IndexName(table, column string) string {
	return strings.ReplaceAll(n.inner.IndexName(table, column), "_sql_", "_")
}
