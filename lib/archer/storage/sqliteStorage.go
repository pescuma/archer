package storage

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

	err = db.AutoMigrate(&sqlProject{}, &sqlProjectDependency{}, &sqlProjectDirectory{}, &sqlProjectFile{})
	if err != nil {
		return nil, err
	}

	return &sqliteStorage{
		db: db,
	}, nil
}

func (s *sqliteStorage) LoadProjects(result *model.Projects) error {
	var projs []*sqlProject
	err := s.db.Find(&projs).Error
	if err != nil {
		return err
	}

	var deps []*sqlProjectDependency
	err = s.db.Find(&deps).Error
	if err != nil {
		return err
	}

	var dirs []*sqlProjectDirectory
	err = s.db.Find(&dirs).Error
	if err != nil {
		return err
	}

	var files []*sqlProjectFile
	err = s.db.Find(&files).Error
	if err != nil {
		return err
	}

	projsByID := map[model.UUID]*model.Project{}
	dirsByID := map[model.UUID]*model.ProjectDirectory{}

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
		d.Size = toModelSize(&sd.Size)

		dirsByID[d.ID] = d
	}

	for _, sf := range files {
		d := dirsByID[sf.DirectoryID]

		f := d.GetFile(sf.RelativePath)
		f.Size = toModelSize(&sf.Size)
	}

	return nil
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
	changedBasicInfo := changes&archer.ChangedProjectBasicInfo != 0 || changes&archer.ChangedProjectConfig != 0 || changes&archer.ChangedProjectSize != 0
	changedDependencies := changes&archer.ChangedProjectDependencies != 0 || changes&archer.ChangedProjectConfig != 0
	changedFiles := changes&archer.ChangedProjectFiles != 0 || changes&archer.ChangedProjectSize != 0

	sqlProjs := make([]sqlProject, len(projs))
	if changedBasicInfo {
		for _, p := range projs {
			sqlProjs = append(sqlProjs, toSqlProject(p))
		}
	}

	var sqlDeps []sqlProjectDependency
	if changedDependencies {
		for _, p := range projs {
			for _, d := range p.Dependencies {
				sqlDeps = append(sqlDeps, toSqlProjectDependency(d))
			}
		}
	}

	var sqlDirs []sqlProjectDirectory
	var sqlFiles []sqlProjectFile
	if changedFiles {
		for _, p := range projs {
			for _, dir := range p.Dirs {
				sqlDirs = append(sqlDirs, toSqlProjectDirectory(dir, p))

				for _, file := range dir.Files {
					sqlFiles = append(sqlFiles, toSqlProjectFile(file, dir, p))
				}
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 100,
	})

	if changedBasicInfo {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlProjs).Error
		if err != nil {
			return err
		}
	}

	if changedDependencies {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDeps).Error
		if err != nil {
			return err
		}
	}

	if changedFiles {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDirs).Error
		if err != nil {
			return err
		}

		err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlFiles).Error
		if err != nil {
			return err
		}
	}

	if changedFiles {
		err := db.Scopes(scope("project_id")).Where("updated_at != ?", now).Delete(&sqlProjectFile{}).Error
		if err != nil {
			return err
		}

		err = db.Scopes(scope("project_id")).Where("updated_at != ?", now).Delete(&sqlProjectDirectory{}).Error
		if err != nil {
			return err
		}
	}

	if changedDependencies {
		err := db.Scopes(scope("source_id")).Where("updated_at != ?", now).Delete(&sqlProjectDependency{}).Error
		if err != nil {
			return err
		}
	}

	if changedBasicInfo {
		err := db.Scopes(scope("project_id")).Where("updated_at != ?", now).Delete(&sqlProject{}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *sqliteStorage) deleteNotUpdated(value interface{}, projectIDField string, proj *model.Project, updatedAts []time.Time) error {
	var updatedAt time.Time
	if len(updatedAts) != 0 {
		updatedAt = updatedAts[0]
	}

	result := s.db.Where(projectIDField+" = ? and updated_at != ?", proj.ID, updatedAt).Delete(value)

	return result.Error
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
		Size:        *toSqlSize(size),
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

func toSqlProjectDirectory(dir *model.ProjectDirectory, p *model.Project) sqlProjectDirectory {
	sd := sqlProjectDirectory{
		ID:           dir.ID,
		ProjectID:    p.ID,
		RelativePath: dir.RelativePath,
		Type:         dir.Type,
		Size:         *toSqlSize(dir.Size),
	}
	return sd
}

func toSqlProjectFile(file *model.ProjectFile, dir *model.ProjectDirectory, p *model.Project) sqlProjectFile {
	sf := sqlProjectFile{
		ID:           file.ID,
		DirectoryID:  dir.ID,
		ProjectID:    p.ID,
		RelativePath: file.RelativePath,
		Size:         *toSqlSize(file.Size),
	}
	return sf
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
