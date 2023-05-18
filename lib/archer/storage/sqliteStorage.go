package storage

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/samber/lo"
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

func (s *sqliteStorage) WriteProjNames(projRoot string, projNames []string) error {
	var old []*sqlProject

	err := s.db.Select("id", "name").Where("root = ?", projRoot).Find(&old).Error
	if err != nil {
		return err
	}

	oldByName := lo.Associate(old, func(p *sqlProject) (string, *sqlProject) { return p.Name, p })

	for _, name := range projNames {
		delete(oldByName, name)
	}

	toDelete := lo.Values(oldByName)
	if len(toDelete) > 0 {
		err = s.db.Delete(&toDelete).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *sqliteStorage) ReadProjNames() ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WriteBasicInfo(proj *model.Project) error {
	size := proj.GetSize()

	sp := sqlProject{
		ID:          proj.ID,
		Root:        proj.Root,
		Name:        proj.Name,
		NameParts:   proj.NameParts,
		Type:        proj.Type,
		RootDir:     proj.RootDir,
		ProjectFile: proj.ProjectFile,
		Size:        *toSqlSize(size),
		Sizes:       map[string]*sqlSize{},
		Data:        proj.Data,
	}

	for k, v := range proj.Sizes {
		sp.Sizes[k] = toSqlSize(v)
	}

	println("Writing %v %v %v", sp.ID, sp.Root, sp.Name)

	return s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sp).Error
}

func (s *sqliteStorage) ReadBasicInfo(result *model.Projects, fileName string) error {
	// TODO implement me
	panic("implement me")
}

func (s *sqliteStorage) WriteDeps(proj *model.Project) error {
	var sds []sqlProjectDependency

	for _, dep := range proj.Dependencies {
		sd := sqlProjectDependency{
			ID:       dep.ID,
			SourceID: dep.Source.ID,
			TargetID: dep.Target.ID,
			Data:     dep.Data,
		}

		sds = append(sds, sd)
	}

	err := s.db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(sds, len(sds)).Error
	if err != nil {
		return err
	}

	err = s.deleteNotUpdated(&sqlProjectDependency{}, "source_id", proj,
		lo.Map(sds, func(s sqlProjectDependency, _ int) time.Time { return s.UpdatedAt }))

	return nil
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

func (s *sqliteStorage) WriteFiles(proj *model.Project) error {
	var sds []sqlProjectDirectory
	var sfs []sqlProjectFile

	for _, dir := range proj.Dirs {
		sd := sqlProjectDirectory{
			ID:           dir.ID,
			ProjectID:    proj.ID,
			RelativePath: dir.RelativePath,
			Type:         dir.Type,
			Size:         *toSqlSize(dir.Size),
		}

		sds = append(sds, sd)

		for _, file := range dir.Files {
			sf := sqlProjectFile{
				ID:           file.ID,
				DirectoryID:  dir.ID,
				ProjectID:    proj.ID,
				RelativePath: file.RelativePath,
				Size:         *toSqlSize(file.Size),
			}

			sfs = append(sfs, sf)
		}
	}

	err := s.db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(sds, len(sds)).Error
	if err != nil {
		return err
	}

	err = s.db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(sfs, len(sfs)).Error
	if err != nil {
		return err
	}

	err = s.deleteNotUpdated(&sqlProjectFile{}, "project_id", proj,
		lo.Map(sfs, func(s sqlProjectFile, _ int) time.Time { return s.UpdatedAt }))
	if err != nil {
		return err
	}

	err = s.deleteNotUpdated(&sqlProjectDirectory{}, "project_id", proj,
		lo.Map(sds, func(s sqlProjectDirectory, _ int) time.Time { return s.UpdatedAt }))
	if err != nil {
		return err
	}

	return nil
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

func (s *sqliteStorage) deleteNotUpdated(value interface{}, projectIDField string, proj *model.Project, updatedAts []time.Time) error {
	var updatedAt time.Time
	if len(updatedAts) != 0 {
		updatedAt = updatedAts[0]
	}

	result := s.db.Where(projectIDField+" = ? and updated_at != ?", proj.ID, updatedAt).Delete(value)

	return result.Error
}
