package sqlite

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
	"github.com/Faire/archer/lib/archer/utils"
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
		&sqlPerson{},
		&sqlRepository{}, &sqlRepositoryCommit{}, &sqlRepositoryCommitFile{},
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

	for _, sp := range projs {
		p := result.GetOrCreate(sp.Root, sp.Name)
		result.ChangeID(p, sp.ID)
		p.NameParts = sp.NameParts
		p.Type = sp.Type

		p.RootDir = sp.RootDir
		p.ProjectFile = sp.ProjectFile

		for k, v := range sp.Sizes {
			p.Sizes[k] = toModelSize(v)
		}
		p.Metrics = toModelMetrics(sp.Metrics)
		p.Data = sp.Data
	}

	for _, sd := range deps {
		source := result.GetByID(sd.SourceID)
		target := result.GetByID(sd.TargetID)

		d := source.GetDependency(target)
		d.ID = sd.ID
		d.Data = sd.Data
	}

	for _, sd := range dirs {
		p := result.GetByID(sd.ProjectID)

		d := p.GetDirectory(sd.RelativePath)
		d.ID = sd.ID
		d.Type = sd.Type
		d.Size = toModelSize(sd.Size)
		d.Metrics = toModelMetrics(sd.Metrics)
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
	changedProjs := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedData != 0 || changes&archer.ChangedSize != 0 || changes&archer.ChangedMetrics != 0
	changedDeps := changes&archer.ChangedDependencies != 0 || changes&archer.ChangedData != 0
	changedDirs := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedSize != 0 || changes&archer.ChangedMetrics != 0

	sqlProjs := make([]*sqlProject, 0, len(projs))
	if changedProjs {
		for _, p := range projs {
			sqlProjs = append(sqlProjs, toSqlProject(p))
		}
	}

	var sqlDeps []*sqlProjectDependency
	if changedDeps {
		for _, p := range projs {
			for _, d := range p.Dependencies {
				sqlDeps = append(sqlDeps, toSqlProjectDependency(d))
			}
		}
	}

	var sqlDirs []*sqlProjectDirectory
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
		f.RepositoryID = sf.RepositoryID
		f.Exists = sf.Exists
		f.Size = toModelSize(sf.Size)
		f.Metrics = toModelMetrics(sf.Metrics)
		f.Data = sf.Data
	}

	return result, nil
}

func (s *sqliteStorage) WriteFiles(files *model.Files, changes archer.StorageChanges) error {
	changed := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedData != 0 || changes&archer.ChangedSize != 0 || changes&archer.ChangedMetrics != 0
	if !changed {
		return nil
	}

	all := files.List()

	sqlFiles := make([]*sqlFile, 0, len(all))
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

func (s *sqliteStorage) LoadPeople() (*model.People, error) {
	result := model.NewPeople()

	var people []*sqlPerson
	err := s.db.Find(&people).Error
	if err != nil {
		return nil, err
	}

	for _, sp := range people {
		p := result.Get(sp.Name)
		p.ID = sp.ID

		for _, name := range sp.Names {
			p.AddName(name)
		}
		for _, email := range sp.Emails {
			p.AddEmail(email)
		}
		p.Data = sp.Data
	}

	return result, nil
}

func (s *sqliteStorage) WritePeople(people *model.People, changes archer.StorageChanges) error {
	changed := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedData != 0
	if !changed {
		return nil
	}

	all := people.List()

	sqlPeople := make([]*sqlPerson, 0, len(all))
	for _, p := range all {
		sqlPeople = append(sqlPeople, toSqlPerson(p))
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 100,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlPeople).Error
	if err != nil {
		return err
	}

	err = db.Where("updated_at != ?", now).Delete(&sqlPerson{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *sqliteStorage) LoadRepository(rootDir string) (*model.Repository, error) {
	var sqlRepo *sqlRepository
	err := s.db.Where("root_dir = ?", rootDir).Limit(1).Find(&sqlRepo).Error
	if err != nil {
		return nil, err
	}
	if sqlRepo.ID == "" {
		return nil, nil
	}

	var sqlCommits []*sqlRepositoryCommit
	err = s.db.Where("repository_id = ?", sqlRepo.ID).Find(&sqlCommits).Error
	if err != nil {
		return nil, err
	}

	var sqlFiles []*sqlRepositoryCommitFile
	err = s.db.Where("repository_id = ?", sqlRepo.ID).Find(&sqlFiles).Error
	if err != nil {
		return nil, err
	}

	filesByCommit := lo.GroupBy(sqlFiles, func(f *sqlRepositoryCommitFile) model.UUID { return f.CommitID })

	result := model.NewRepository(rootDir)
	result.VCS = sqlRepo.VCS
	result.ID = sqlRepo.ID
	result.Data = sqlRepo.Data

	for _, sc := range sqlCommits {
		c := result.GetCommit(sc.Hash)
		c.ID = sc.ID
		c.Hash = sc.Hash
		c.Parents = sc.Parents
		c.Date = sc.Date
		c.CommitterID = sc.CommitterID
		c.DateAuthored = sc.DateAuthored
		c.AuthorID = sc.AuthorID
		c.AddedLines = sc.AddedLines
		c.DeletedLines = sc.DeletedLines

		if sfs, ok := filesByCommit[sc.ID]; ok {
			for _, sf := range sfs {
				c.AddFile(sf.FileID, sf.AddedLines, sf.DeletedLines)
			}
		}
	}

	return result, nil
}

func (s *sqliteStorage) WriteRepository(repo *model.Repository, changes archer.StorageChanges) error {
	changedRepo := changes&archer.ChangedBasicInfo != 0
	changedCommits := changes&archer.ChangedHistory != 0
	changedFiles := changes&archer.ChangedHistory != 0

	var sqlRepo *sqlRepository
	if changedRepo {
		sqlRepo = toSqlRepository(repo)
	}

	var sqlCommits []*sqlRepositoryCommit
	if changedCommits {
		for _, c := range repo.ListCommits() {
			sqlCommits = append(sqlCommits, toSqlRepositoryCommit(repo, c))
		}
	}

	var sqlFiles []*sqlRepositoryCommitFile
	if changedFiles {
		for _, c := range repo.ListCommits() {
			for _, f := range c.Files {
				sqlFiles = append(sqlFiles, toSqlRepositoryCommitFile(repo, c, f))
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 100,
	})

	if changedRepo {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlRepo).Error
		if err != nil {
			return err
		}
	}

	if changedCommits {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommits).Error
		if err != nil {
			return err
		}
	}

	if changedFiles {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlFiles).Error
		if err != nil {
			return err
		}
	}

	if changedFiles {
		err := db.Where("repository_id = ? and updated_at != ?", repo.ID, now).Delete(&sqlRepositoryCommitFile{}).Error
		if err != nil {
			return err
		}
	}

	if changedCommits {
		err := db.Where("repository_id = ? and updated_at != ?", repo.ID, now).Delete(&sqlRepositoryCommit{}).Error
		if err != nil {
			return err
		}
	}

	return nil
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

func toSqlMetrics(metrics *model.Metrics) *sqlMetrics {
	return &sqlMetrics{
		DependenciesGuice: encodeMetric(metrics.GuiceDependencies),
	}
}

func toModelMetrics(metrics *sqlMetrics) *model.Metrics {
	return &model.Metrics{
		GuiceDependencies: decodeMetric(metrics.DependenciesGuice),
	}
}

func encodeMetric(v int) *int {
	return utils.IIf(v == -1, nil, &v)
}
func decodeMetric(v *int) int {
	if v == nil {
		return -1
	} else {
		return *v
	}
}

func toSqlProject(p *model.Project) *sqlProject {
	size := p.GetSize()

	sp := &sqlProject{
		ID:          p.ID,
		Root:        p.Root,
		Name:        p.Name,
		NameParts:   p.NameParts,
		Type:        p.Type,
		RootDir:     p.RootDir,
		ProjectFile: p.ProjectFile,
		Size:        toSqlSize(size),
		Sizes:       map[string]*sqlSize{},
		Metrics:     toSqlMetrics(p.Metrics),
		Data:        p.Data,
	}

	for k, v := range p.Sizes {
		sp.Sizes[k] = toSqlSize(v)
	}

	return sp
}

func toSqlProjectDependency(d *model.ProjectDependency) *sqlProjectDependency {
	return &sqlProjectDependency{
		ID:       d.ID,
		SourceID: d.Source.ID,
		TargetID: d.Target.ID,
		Data:     d.Data,
	}
}

func toSqlProjectDirectory(d *model.ProjectDirectory, p *model.Project) *sqlProjectDirectory {
	return &sqlProjectDirectory{
		ID:           d.ID,
		ProjectID:    p.ID,
		RelativePath: d.RelativePath,
		Type:         d.Type,
		Size:         toSqlSize(d.Size),
		Metrics:      toSqlMetrics(d.Metrics),
		Data:         d.Data,
	}
}

func toSqlFile(f *model.File) *sqlFile {
	return &sqlFile{
		ID:                 f.ID,
		Path:               f.Path,
		ProjectID:          f.ProjectID,
		ProjectDirectoryID: f.ProjectDirectoryID,
		RepositoryID:       f.RepositoryID,
		Exists:             f.Exists,
		Size:               toSqlSize(f.Size),
		Metrics:            toSqlMetrics(f.Metrics),
		Data:               f.Data,
	}
}

func toSqlPerson(p *model.Person) *sqlPerson {
	return &sqlPerson{
		ID:     p.ID,
		Name:   p.Name,
		Names:  p.ListNames(),
		Emails: p.ListEmails(),
		Data:   p.Data,
	}
}

func toSqlRepository(r *model.Repository) *sqlRepository {
	return &sqlRepository{
		ID:      r.ID,
		RootDir: r.RootDir,
		VCS:     r.VCS,
		Data:    r.Data,
	}
}

func toSqlRepositoryCommit(r *model.Repository, c *model.RepositoryCommit) *sqlRepositoryCommit {
	return &sqlRepositoryCommit{
		ID:           c.ID,
		RepositoryID: r.ID,
		Hash:         c.Hash,
		Parents:      c.Parents,
		Date:         c.Date,
		CommitterID:  c.CommitterID,
		DateAuthored: c.DateAuthored,
		AuthorID:     c.AuthorID,
		AddedLines:   c.AddedLines,
		DeletedLines: c.DeletedLines,
	}
}

func toSqlRepositoryCommitFile(r *model.Repository, c *model.RepositoryCommit, f *model.RepositoryCommitFile) *sqlRepositoryCommitFile {
	return &sqlRepositoryCommitFile{
		CommitID:     c.ID,
		FileID:       f.FileID,
		RepositoryID: r.ID,
		AddedLines:   f.AddedLines,
		DeletedLines: f.DeletedLines,
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
