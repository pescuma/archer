package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
)

type sqliteStorage struct {
	mainMutex      sync.RWMutex
	mainDB         *gorm.DB
	fileLinesMutex sync.RWMutex
	fileLinesDB    *gorm.DB

	configs     map[string]*sqlConfig
	projs       map[model.UUID]*sqlProject
	projDeps    map[model.UUID]*sqlProjectDependency
	projDirs    map[model.UUID]*sqlProjectDirectory
	files       map[model.UUID]*sqlFile
	people      map[model.UUID]*sqlPerson
	peopleRepos map[string]*sqlPersonRepository
	peopleFiles map[string]*sqlPersonFile
	area        map[model.UUID]*sqlProductArea
	repos       map[model.UUID]*sqlRepository
	repoCommits map[model.UUID]*sqlRepositoryCommit
	monthLines  map[model.UUID]*sqlMonthLines
}

func NewSqliteStorage(path string) (archer.Storage, error) {
	if _, err := os.Stat(path); err != nil {
		fmt.Printf("Creating workspace at %v\n", path)
		err = os.MkdirAll(path, 0o700)
		if err != nil {
			return nil, err
		}
	}

	mainFile := filepath.Join(path, "archer_main.db")
	fileLinesFile := filepath.Join(path, "archer_file_lines.db")

	return newFrom(mainFile+"?_pragma=journal_mode(WAL)", fileLinesFile+"?_pragma=journal_mode(WAL)")
}

func NewSqliteMemoryStorage(_ string) (archer.Storage, error) {
	return newFrom(":memory:", ":memory:")
}

func newFrom(mainDSN string, fileLinesDSN string) (archer.Storage, error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	)

	mainDB, err := gorm.Open(sqlite.Open(mainDSN), &gorm.Config{
		NamingStrategy: &NamingStrategy{},
		Logger:         newLogger,
	})
	if err != nil {
		return nil, err
	}

	err = mainDB.AutoMigrate(
		&sqlConfig{},
		&sqlProject{}, &sqlProjectDependency{}, &sqlProjectDirectory{},
		&sqlFile{},
		&sqlPerson{}, &sqlPersonRepository{}, &sqlPersonFile{}, &sqlProductArea{},
		&sqlRepository{}, &sqlRepositoryCommit{}, &sqlRepositoryCommitFile{},
		&sqlMonthLines{},
	)
	if err != nil {
		return nil, err
	}

	fileLinesDB, err := gorm.Open(sqlite.Open(fileLinesDSN), &gorm.Config{
		NamingStrategy: &NamingStrategy{},
		Logger:         newLogger,
	})
	if err != nil {
		return nil, err
	}

	err = fileLinesDB.AutoMigrate(
		&sqlFileLine{},
	)
	if err != nil {
		return nil, err
	}

	return &sqliteStorage{
		mainDB:      mainDB,
		fileLinesDB: fileLinesDB,
		configs:     map[string]*sqlConfig{},
		projs:       map[model.UUID]*sqlProject{},
		projDeps:    map[model.UUID]*sqlProjectDependency{},
		projDirs:    map[model.UUID]*sqlProjectDirectory{},
		files:       map[model.UUID]*sqlFile{},
		people:      map[model.UUID]*sqlPerson{},
		peopleRepos: map[string]*sqlPersonRepository{},
		peopleFiles: map[string]*sqlPersonFile{},
		repos:       map[model.UUID]*sqlRepository{},
		repoCommits: map[model.UUID]*sqlRepositoryCommit{},
	}, nil
}

func (s *sqliteStorage) LoadProjects() (*model.Projects, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	result := model.NewProjects()

	var projs []*sqlProject
	err := s.mainDB.Find(&projs).Error
	if err != nil {
		return nil, err
	}

	s.projs = lo.Associate(projs, func(i *sqlProject) (model.UUID, *sqlProject) {
		return i.ID, i
	})

	var deps []*sqlProjectDependency
	err = s.mainDB.Find(&deps).Error
	if err != nil {
		return nil, err
	}

	s.projDeps = lo.Associate(deps, func(i *sqlProjectDependency) (model.UUID, *sqlProjectDependency) {
		return i.ID, i
	})

	var dirs []*sqlProjectDirectory
	err = s.mainDB.Find(&dirs).Error
	if err != nil {
		return nil, err
	}

	s.projDirs = lo.Associate(dirs, func(i *sqlProjectDirectory) (model.UUID, *sqlProjectDirectory) {
		return i.ID, i
	})

	for _, sp := range projs {
		p := result.GetOrCreateEx(sp.Root, sp.ProjectName, &sp.ID)
		p.NameParts = sp.NameParts
		p.Type = sp.Type

		p.RootDir = sp.RootDir
		p.ProjectFile = sp.ProjectFile
		p.RepositoryID = sp.RepositoryID

		for k, v := range sp.Sizes {
			p.Sizes[k] = toModelSize(v)
		}
		p.Size = toModelSize(sp.Size)
		p.Changes = toModelChanges(sp.Changes)
		p.Metrics = toModelMetricsAggregate(sp.Metrics)
		p.Data = decodeMap(sp.Data)
		p.FirstSeen = sp.FirstSeen
		p.LastSeen = sp.LastSeen
	}

	for _, sd := range deps {
		source := result.GetByID(sd.SourceID)
		target := result.GetByID(sd.TargetID)

		d := source.GetOrCreateDependency(target)
		d.ID = sd.ID
		d.Versions.InsertSlice(sd.Versions)
		d.Data = decodeMap(sd.Data)
	}

	for _, sd := range dirs {
		p := result.GetByID(sd.ProjectID)

		d := p.GetDirectory(sd.Name)
		d.ID = sd.ID
		d.Type = sd.Type
		d.Size = toModelSize(sd.Size)
		d.Changes = toModelChanges(sd.Changes)
		d.Metrics = toModelMetricsAggregate(sd.Metrics)
		d.Data = decodeMap(sd.Data)
		d.FirstSeen = sd.FirstSeen
		d.LastSeen = sd.LastSeen
	}

	return result, nil
}

func (s *sqliteStorage) WriteProjects(projs *model.Projects) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	all := projs.ListProjects(model.FilterAll)

	return s.writeProjects(all)
}

func (s *sqliteStorage) WriteProject(proj *model.Project) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	projs := []*model.Project{proj}

	return s.writeProjects(projs)
}

func (s *sqliteStorage) writeProjects(projs []*model.Project) error {
	var sqlProjs []*sqlProject
	for _, p := range projs {
		sp := toSqlProject(p)
		if prepareChange(&s.projs, sp.ID, sp) {
			sqlProjs = append(sqlProjs, sp)
		}
	}

	var sqlDeps []*sqlProjectDependency
	for _, p := range projs {
		for _, d := range p.Dependencies {
			sd := toSqlProjectDependency(d)
			if prepareChange(&s.projDeps, sd.ID, sd) {
				sqlDeps = append(sqlDeps, sd)
			}
		}
	}

	var sqlDirs []*sqlProjectDirectory
	for _, p := range projs {
		for _, d := range p.Dirs {
			sd := toSqlProjectDirectory(d, p)
			if prepareChange(&s.projDirs, sd.ID, sd) {
				sqlDirs = append(sqlDirs, sd)
			}
		}
	}

	now := time.Now().Local()
	db := s.mainDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlProjs).Error
	if err != nil {
		return err
	}

	addList(&s.projs, sqlProjs, func(s *sqlProject) model.UUID { return s.ID })

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDeps).Error
	if err != nil {
		return err
	}

	addList(&s.projDeps, sqlDeps, func(s *sqlProjectDependency) model.UUID { return s.ID })

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDirs).Error
	if err != nil {
		return err
	}

	addList(&s.projDirs, sqlDirs, func(s *sqlProjectDirectory) model.UUID { return s.ID })

	// TODO delete

	return nil
}

func (s *sqliteStorage) LoadFiles() (*model.Files, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	result := model.NewFiles()

	var files []*sqlFile
	err := s.mainDB.Find(&files).Error
	if err != nil {
		return nil, err
	}

	s.files = lo.Associate(files, func(i *sqlFile) (model.UUID, *sqlFile) {
		return i.ID, i
	})

	for _, sf := range files {
		f := result.GetOrCreateFileEx(sf.Name, &sf.ID)
		f.ProjectID = sf.ProjectID
		f.ProjectDirectoryID = sf.ProjectDirectoryID
		f.RepositoryID = sf.RepositoryID
		f.ProductAreaID = sf.ProductAreaID
		f.Exists = sf.Exists
		f.Size = toModelSize(sf.Size)
		f.Changes = toModelChanges(sf.Changes)
		f.Metrics = toModelMetrics(sf.Metrics)
		f.Data = decodeMap(sf.Data)
		f.FirstSeen = sf.FirstSeen
		f.LastSeen = sf.LastSeen
	}

	return result, nil
}

func (s *sqliteStorage) WriteFiles(files *model.Files) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	all := files.ListFiles()

	err := s.writeFiles(all)
	if err != nil {
		return err
	}

	// TODO delete

	return nil
}

func (s *sqliteStorage) WriteFile(file *model.File) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	return s.writeFiles([]*model.File{file})
}

func (s *sqliteStorage) writeFiles(all []*model.File) error {
	var sqlFiles []*sqlFile
	for _, f := range all {
		sf := toSqlFile(f)
		if prepareChange(&s.files, sf.ID, sf) {
			sqlFiles = append(sqlFiles, sf)
		}
	}

	now := time.Now().Local()
	db := s.mainDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlFiles).Error
	if err != nil {
		return err
	}

	addList(&s.files, sqlFiles, func(s *sqlFile) model.UUID { return s.ID })

	return nil
}

func (s *sqliteStorage) LoadFileContents(fileID model.UUID) (*model.FileContents, error) {
	s.fileLinesMutex.RLock()
	defer s.fileLinesMutex.RUnlock()

	result := model.NewFileContents(fileID)

	var lines []*sqlFileLine
	err := s.fileLinesDB.Where("file_id = ?", fileID).Find(&lines).Error
	if err != nil {
		return nil, err
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Line <= lines[j].Line
	})

	for _, sf := range lines {
		line := result.AppendLine()

		if sf.Line != line.Line {
			return nil, fmt.Errorf("invalid line number: %v (should be %v)", line.Line, sf.Line)
		}

		line.ProjectID = sf.ProjectID
		line.RepositoryID = sf.RepositoryID
		line.CommitID = sf.CommitID
		line.AuthorID = sf.AuthorID
		line.CommitterID = sf.CommitterID
		line.Date = sf.Date
		line.Type = sf.Type
		line.Text = sf.Text
	}

	return result, nil
}

func (s *sqliteStorage) WriteFileContents(contents *model.FileContents) error {
	s.fileLinesMutex.Lock()
	defer s.fileLinesMutex.Unlock()

	var sqlLines []*sqlFileLine
	for _, f := range contents.Lines {
		sf := toSqlFileLine(contents.FileID, f)
		sqlLines = append(sqlLines, sf)
	}

	now := time.Now().Local()
	db := s.fileLinesDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlLines).Error
	if err != nil {
		return err
	}

	err = db.Where("file_id = ? and line > ?", contents.FileID, len(contents.Lines)).Delete(&sqlFileLine{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *sqliteStorage) QueryBlamePerAuthor() ([]*archer.BlamePerAuthor, error) {
	s.fileLinesMutex.RLock()
	defer s.fileLinesMutex.RUnlock()

	var result []*archer.BlamePerAuthor

	err := s.fileLinesDB.Raw(`
select author_id, committer_id, repository_id, commit_id, file_id, type line_type, count(*) lines
from file_lines l
group by author_id, committer_id, repository_id, commit_id, file_id, type
	`).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *sqliteStorage) LoadPeople() (*model.People, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	result := model.NewPeople()

	var people []*sqlPerson
	err := s.mainDB.Find(&people).Error
	if err != nil {
		return nil, err
	}

	s.people = lo.Associate(people, func(i *sqlPerson) (model.UUID, *sqlPerson) {
		return i.ID, i
	})

	var areas []*sqlProductArea
	err = s.mainDB.Find(&areas).Error
	if err != nil {
		return nil, err
	}

	s.area = lo.Associate(areas, func(i *sqlProductArea) (model.UUID, *sqlProductArea) {
		return i.ID, i
	})

	for _, sp := range people {
		p := result.GetOrCreatePersonEx(sp.Name, &sp.ID)
		for _, name := range sp.Names {
			p.AddName(name)
		}
		for _, email := range sp.Emails {
			p.AddEmail(email)
		}
		p.Blame = toModelBlame(sp.Blame)
		p.Changes = toModelChanges(sp.Changes)
		p.Data = decodeMap(sp.Data)
		p.FirstSeen = sp.FirstSeen
		p.LastSeen = sp.LastSeen
	}

	for _, sa := range areas {
		a := result.GetOrCreateProductAreaEx(sa.Name, &sa.ID)
		a.Size = toModelSize(sa.Size)
		a.Changes = toModelChanges(sa.Changes)
		a.Metrics = toModelMetricsAggregate(sa.Metrics)
		a.Data = decodeMap(sa.Data)
	}

	return result, nil
}

func (s *sqliteStorage) WritePeople(peopleDB *model.People) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	var sqlPeople []*sqlPerson
	people := peopleDB.ListPeople()
	for _, p := range people {
		sp := toSqlPerson(p)
		if prepareChange(&s.people, sp.ID, sp) {
			sqlPeople = append(sqlPeople, sp)
		}
	}

	var sqlAreas []*sqlProductArea
	area := peopleDB.ListProductAreas()
	for _, p := range area {
		sp := toSqlProductArea(p)
		if prepareChange(&s.area, sp.ID, sp) {
			sqlAreas = append(sqlAreas, sp)
		}
	}

	now := time.Now().Local()
	db := s.mainDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlPeople).Error
	if err != nil {
		return err
	}

	addList(&s.people, sqlPeople, func(s *sqlPerson) model.UUID { return s.ID })

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlAreas).Error
	if err != nil {
		return err
	}

	addList(&s.area, sqlAreas, func(s *sqlProductArea) model.UUID { return s.ID })

	// TODO delete

	return nil
}

func compositeKey(ids ...model.UUID) string {
	return strings.Join(lo.Map(ids, func(i model.UUID, _ int) string { return string(i) }), "\n")
}

func (s *sqliteStorage) LoadPeopleRelations() (*model.PeopleRelations, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	result := model.NewPeopleRelations()

	var rs []*sqlPersonRepository
	err := s.mainDB.Find(&rs).Error
	if err != nil {
		return nil, err
	}

	s.peopleRepos = lo.Associate(rs, func(i *sqlPersonRepository) (string, *sqlPersonRepository) {
		return compositeKey(i.PersonID, i.RepositoryID), i
	})

	var fs []*sqlPersonFile
	err = s.mainDB.Find(&fs).Error
	if err != nil {
		return nil, err
	}

	s.peopleFiles = lo.Associate(fs, func(i *sqlPersonFile) (string, *sqlPersonFile) {
		return compositeKey(i.PersonID, i.FileID), i
	})

	for _, r := range rs {
		pr := result.GetOrCreatePersonRepo(r.PersonID, r.RepositoryID)
		pr.FirstSeen = r.FirstSeen
		pr.LastSeen = r.LastSeen
	}

	for _, f := range fs {
		pr := result.GetOrCreatePersonFile(f.PersonID, f.FileID)
		pr.FirstSeen = f.FirstSeen
		pr.LastSeen = f.LastSeen
	}

	return result, nil
}

func (s *sqliteStorage) WritePeopleRelations(prs *model.PeopleRelations) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	var rs []*sqlPersonRepository
	for _, r := range prs.ListRepositories() {
		pr := toSqlPersonRepository(r)
		if prepareChange(&s.peopleRepos, compositeKey(r.PersonID, r.RepositoryID), pr) {
			rs = append(rs, pr)
		}
	}

	var fs []*sqlPersonFile
	for _, f := range prs.ListFiles() {
		pf := toSqlPersonFile(f)
		if prepareChange(&s.peopleFiles, compositeKey(f.PersonID, f.FileID), pf) {
			fs = append(fs, pf)
		}
	}

	now := time.Now().Local()
	db := s.mainDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&rs).Error
	if err != nil {
		return err
	}

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&fs).Error
	if err != nil {
		return err
	}

	// TODO delete

	addList(&s.peopleRepos, rs, func(pr *sqlPersonRepository) string { return compositeKey(pr.PersonID, pr.RepositoryID) })
	addList(&s.peopleFiles, fs, func(pr *sqlPersonFile) string { return compositeKey(pr.PersonID, pr.FileID) })

	return nil
}

func (s *sqliteStorage) LoadRepositories() (*model.Repositories, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	return s.loadRepositories(
		func([]*sqlRepository) func(db *gorm.DB) *gorm.DB {
			return func(db *gorm.DB) *gorm.DB {
				return db
			}
		})
}

func (s *sqliteStorage) LoadRepository(rootDir string) (*model.Repository, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	reposDB, err := s.loadRepositories(
		func(repos []*sqlRepository) func(db *gorm.DB) *gorm.DB {
			if repos == nil {
				return func(db *gorm.DB) *gorm.DB {
					return db.Where("root_dir = ?", rootDir)
				}
			} else {
				return func(db *gorm.DB) *gorm.DB {
					return db.Where("repository_id = ?", repos[0].ID)
				}
			}
		})
	if err != nil {
		return nil, err
	}

	return reposDB.Get(rootDir), nil
}

func (s *sqliteStorage) loadRepositories(scope func([]*sqlRepository) func(*gorm.DB) *gorm.DB) (*model.Repositories, error) {
	result := model.NewRepositories()

	var repos []*sqlRepository
	err := s.mainDB.Scopes(scope(repos)).Find(&repos).Error
	if err != nil {
		return nil, err
	}

	addMap(&s.repos, lo.Associate(repos, func(i *sqlRepository) (model.UUID, *sqlRepository) {
		return i.ID, i
	}))

	if len(repos) == 0 {
		return result, nil
	}

	var commits []*sqlRepositoryCommit
	err = s.mainDB.Scopes(scope(repos)).Find(&commits).Error
	if err != nil {
		return nil, err
	}

	addMap(&s.repoCommits, lo.Associate(commits, func(i *sqlRepositoryCommit) (model.UUID, *sqlRepositoryCommit) {
		return i.ID, i
	}))

	for _, sr := range repos {
		r := result.GetOrCreateEx(sr.RootDir, &sr.ID)
		r.Name = sr.Name
		r.VCS = sr.VCS
		r.Data = decodeMap(sr.Data)
		r.FirstSeen = sr.FirstSeen
		r.LastSeen = sr.LastSeen
		r.FilesTotal = decodeMetric(sr.FilesTotal)
		r.FilesHead = decodeMetric(sr.FilesHead)
	}

	commitsById := map[model.UUID]*model.RepositoryCommit{}
	for _, sc := range commits {
		repo := result.GetByID(sc.RepositoryID)

		c := repo.GetOrCreateCommitEx(sc.Name, &sc.ID)
		c.Message = sc.Message
		c.Parents = sc.Parents
		c.Children = sc.Children
		c.Date = sc.Date
		c.CommitterID = sc.CommitterID
		c.DateAuthored = sc.DateAuthored
		c.AuthorID = sc.AuthorID
		c.Ignore = sc.Ignore
		c.FilesModified = decodeMetric(sc.FilesModified)
		c.FilesCreated = decodeMetric(sc.FilesCreated)
		c.FilesDeleted = decodeMetric(sc.FilesDeleted)
		c.LinesModified = decodeMetric(sc.LinesModified)
		c.LinesAdded = decodeMetric(sc.LinesAdded)
		c.LinesDeleted = decodeMetric(sc.LinesDeleted)
		c.Blame = toModelBlame(sc.Blame)

		commitsById[c.ID] = c
	}

	return result, nil
}

func (s *sqliteStorage) WriteRepositories(repos *model.Repositories) error {
	for _, repo := range repos.List() {
		err := s.WriteRepository(repo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *sqliteStorage) WriteRepository(repo *model.Repository) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	var sqlRepos []*sqlRepository
	sr := toSqlRepository(repo)
	if prepareChange(&s.repos, sr.ID, sr) {
		sqlRepos = append(sqlRepos, sr)
	}

	var sqlCommits []*sqlRepositoryCommit
	for _, c := range repo.ListCommits() {
		sc := toSqlRepositoryCommit(repo, c)
		if prepareChange(&s.repoCommits, sc.ID, sc) {
			sqlCommits = append(sqlCommits, sc)
		}
	}

	now := time.Now().Local()
	db := s.mainDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlRepos).Error
	if err != nil {
		return err
	}

	addList(&s.repos, sqlRepos, func(s *sqlRepository) model.UUID { return s.ID })

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommits).Error
	if err != nil {
		return err
	}

	addList(&s.repoCommits, sqlCommits, func(s *sqlRepositoryCommit) model.UUID { return s.ID })

	// TODO delete

	return nil
}

func (s *sqliteStorage) WriteCommit(repo *model.Repository, commit *model.RepositoryCommit) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	var sqlCommits []*sqlRepositoryCommit

	sc := toSqlRepositoryCommit(repo, commit)
	if prepareChange(&s.repoCommits, sc.ID, sc) {
		sqlCommits = append(sqlCommits, sc)
	}

	now := time.Now().Local()
	db := s.mainDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommits).Error
	if err != nil {
		return err
	}

	addList(&s.repoCommits, sqlCommits, func(s *sqlRepositoryCommit) model.UUID { return s.ID })

	// TODO delete

	return nil
}

func (s *sqliteStorage) LoadRepositoryCommitFiles(repo *model.Repository, commit *model.RepositoryCommit) (*model.RepositoryCommitFiles, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	var commitFiles []*sqlRepositoryCommitFile
	err := s.mainDB.Where("commit_id = ?", commit.ID).Find(&commitFiles).Error
	if err != nil {
		return nil, err
	}

	result := model.NewRepositoryCommitFiles(repo.ID, commit.ID)
	for _, sf := range commitFiles {
		file := result.GetOrCreate(sf.FileID)
		file.Hash = sf.Hash
		file.OldFileIDs = decodeOldFileIDs(sf.OldFileIDs)
		file.Change = sf.Change
		file.LinesModified = decodeMetric(sf.LinesModified)
		file.LinesAdded = decodeMetric(sf.LinesAdded)
		file.LinesDeleted = decodeMetric(sf.LinesDeleted)
	}
	return result, nil
}

func (s *sqliteStorage) WriteRepositoryCommitFiles(files []*model.RepositoryCommitFiles) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	var sqlCommitFiles []*sqlRepositoryCommitFile
	for _, fs := range files {
		for _, f := range fs.List() {
			sf := toSqlRepositoryCommitFile(fs.CommitID, f)
			sqlCommitFiles = append(sqlCommitFiles, sf)
		}
	}

	now := time.Now().Local()
	db := s.mainDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommitFiles).Error
	if err != nil {
		return err
	}

	// TODO delete

	return nil
}

func (s *sqliteStorage) QueryCommits(file string, proj string, repo string, person string) ([]model.UUID, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	var result []model.UUID

	err := s.mainDB.Raw(`
select distinct c.id
from repository_commits c
         join repositories r
              on r.id = c.repository_id
         join people pa
              on pa.id = c.author_id
         join people pc
              on pc.id = c.committer_id
         join repository_commit_files cf
              on cf.commit_id = c.id
         join files f
              on f.id = cf.file_id
         left join projects p
              on p.id = f.project_id
where c.ignore = 0
  and (@proj = '' or p.name like @proj)
  and (@file = '' or f.name like @file)
  and (@repo = '' or r.name like @repo)
  and (@person = '' or pc.names like @person or pc.emails like @person or pa.names like @person or pa.emails like @person)
		`,
		sql.Named("proj", "%"+proj+"%"),
		sql.Named("file", "%"+file+"%"),
		sql.Named("repo", "%"+repo+"%"),
		sql.Named("person", "%"+person+"%"),
	).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *sqliteStorage) LoadMonthlyStats() (*model.MonthlyStats, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	result := model.NewMonthlyStats()

	var sqlLines []*sqlMonthLines
	err := s.mainDB.Find(&sqlLines).Error
	if err != nil {
		return nil, err
	}

	s.monthLines = lo.Associate(sqlLines, func(i *sqlMonthLines) (model.UUID, *sqlMonthLines) {
		return i.ID, i
	})

	for _, sl := range sqlLines {
		l := result.GetOrCreateLines(sl.Month, sl.RepositoryID, sl.AuthorID, sl.CommitterID, sl.FileID, sl.ProjectID)
		l.ID = sl.ID
		l.Changes = toModelChanges(sl.Changes)
		l.Blame = toModelBlame(sl.Blame)
	}

	return result, nil
}

func (s *sqliteStorage) WriteMonthlyStats(stats *model.MonthlyStats) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	var sqlLines []*sqlMonthLines
	for _, f := range stats.ListLines() {
		sf := toSqlMonthLines(f)
		if prepareChange(&s.monthLines, sf.ID, sf) {
			sqlLines = append(sqlLines, sf)
		}
	}

	now := time.Now().Local()
	db := s.mainDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlLines).Error
	if err != nil {
		return err
	}

	addList(&s.monthLines, sqlLines, func(s *sqlMonthLines) model.UUID { return s.ID })

	// TODO delete

	return nil
}

func (s *sqliteStorage) LoadConfig() (*map[string]string, error) {
	s.mainMutex.RLock()
	defer s.mainMutex.RUnlock()

	result := map[string]string{}

	var sqlConfigs []*sqlConfig
	err := s.mainDB.Find(&sqlConfigs).Error
	if err != nil {
		return nil, err
	}

	s.configs = lo.Associate(sqlConfigs, func(i *sqlConfig) (string, *sqlConfig) {
		return i.Key, i
	})

	for _, sc := range sqlConfigs {
		result[sc.Key] = sc.Value
	}

	return &result, nil
}

func (s *sqliteStorage) WriteConfig(configs *map[string]string) error {
	s.mainMutex.Lock()
	defer s.mainMutex.Unlock()

	var sqlConfigs []*sqlConfig
	for k, v := range *configs {
		sc := toSqlConfig(k, v)
		if prepareChange(&s.configs, sc.Key, sc) {
			sqlConfigs = append(sqlConfigs, sc)
		}
	}

	now := time.Now().Local()
	db := s.mainDB.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlConfigs).Error
	if err != nil {
		return err
	}

	addList(&s.configs, sqlConfigs, func(s *sqlConfig) string { return s.Key })

	// TODO delete

	return nil
}

func toSqlMonthLines(l *model.MonthlyStatsLine) *sqlMonthLines {
	return &sqlMonthLines{
		ID:           l.ID,
		Month:        l.Month,
		RepositoryID: l.RepositoryID,
		AuthorID:     l.AuthorID,
		CommitterID:  l.CommitterID,
		FileID:       l.FileID,
		ProjectID:    l.ProjectID,
		Changes:      toSqlChanges(l.Changes),
		Blame:        toSqlBlame(l.Blame),
	}
}

func toSqlConfig(k string, v string) *sqlConfig {
	return &sqlConfig{
		Key:   k,
		Value: v,
	}
}

func toSqlSize(size *model.Size) *sqlSize {
	return &sqlSize{
		Lines: encodeMetric(size.Lines),
		Files: encodeMetric(size.Files),
		Bytes: encodeMetric(size.Bytes),
		Other: encodeMap(size.Other),
	}
}

func toModelSize(size *sqlSize) *model.Size {
	return &model.Size{
		Lines: decodeMetric(size.Lines),
		Files: decodeMetric(size.Files),
		Bytes: decodeMetric(size.Bytes),
		Other: decodeMap(size.Other),
	}
}

func toSqlBlame(blame *model.Blame) *sqlBlame {
	return &sqlBlame{
		Code:    encodeMetric(blame.Code),
		Comment: encodeMetric(blame.Comment),
		Blank:   encodeMetric(blame.Blank),
	}
}

func toModelBlame(blame *sqlBlame) *model.Blame {
	return &model.Blame{
		Code:    decodeMetric(blame.Code),
		Comment: decodeMetric(blame.Comment),
		Blank:   decodeMetric(blame.Blank),
	}
}

func toSqlChanges(c *model.Changes) *sqlChanges {
	return &sqlChanges{
		Semester:      encodeMetric(c.In6Months),
		Total:         encodeMetric(c.Total),
		LinesModified: encodeMetric(c.LinesModified),
		LinesAdded:    encodeMetric(c.LinesAdded),
		LinesDeleted:  encodeMetric(c.LinesDeleted),
	}
}

func toModelChanges(sc *sqlChanges) *model.Changes {
	return &model.Changes{
		In6Months:     decodeMetric(sc.Semester),
		Total:         decodeMetric(sc.Total),
		LinesModified: decodeMetric(sc.LinesModified),
		LinesAdded:    decodeMetric(sc.LinesAdded),
		LinesDeleted:  decodeMetric(sc.LinesDeleted),
	}
}

func toSqlMetrics(metrics *model.Metrics) *sqlMetrics {
	return &sqlMetrics{
		DependenciesGuice:    encodeMetric(metrics.GuiceDependencies),
		Abstracts:            encodeMetric(metrics.Abstracts),
		ComplexityCyclomatic: encodeMetric(metrics.CyclomaticComplexity),
		ComplexityCognitive:  encodeMetric(metrics.CognitiveComplexity),
		ComplexityFocus:      encodeMetric(metrics.FocusedComplexity),
	}
}

func toModelMetrics(metrics *sqlMetrics) *model.Metrics {
	return &model.Metrics{
		GuiceDependencies:    decodeMetric(metrics.DependenciesGuice),
		Abstracts:            decodeMetric(metrics.Abstracts),
		CyclomaticComplexity: decodeMetric(metrics.ComplexityCyclomatic),
		CognitiveComplexity:  decodeMetric(metrics.ComplexityCognitive),
		FocusedComplexity:    decodeMetric(metrics.ComplexityFocus),
	}
}

func toSqlMetricsAggregate(metrics *model.Metrics, size *model.Size) *sqlMetricsAggregate {
	return &sqlMetricsAggregate{
		DependenciesGuiceTotal:    encodeMetric(metrics.GuiceDependencies),
		DependenciesGuiceAvg:      encodeMetricAggregate(metrics.GuiceDependencies, size.Files),
		ComplexityCyclomaticTotal: encodeMetric(metrics.CyclomaticComplexity),
		ComplexityCyclomaticAvg:   encodeMetricAggregate(metrics.CyclomaticComplexity, size.Files),
		ComplexityCognitiveTotal:  encodeMetric(metrics.CognitiveComplexity),
		ComplexityCognitiveAvg:    encodeMetricAggregate(metrics.CognitiveComplexity, size.Files),
		ComplexityFocusTotal:      encodeMetric(metrics.FocusedComplexity),
		ComplexityFocusAvg:        encodeMetricAggregate(metrics.FocusedComplexity, size.Files),
	}
}

func toModelMetricsAggregate(metrics *sqlMetricsAggregate) *model.Metrics {
	return &model.Metrics{
		GuiceDependencies:    decodeMetric(metrics.DependenciesGuiceTotal),
		CyclomaticComplexity: decodeMetric(metrics.ComplexityCyclomaticTotal),
		CognitiveComplexity:  decodeMetric(metrics.ComplexityCognitiveTotal),
		FocusedComplexity:    decodeMetric(metrics.ComplexityFocusTotal),
	}
}

func encodeMetricAggregate(v int, t int) *float32 {
	if v == -1 {
		return nil
	}
	if t == 0 {
		return nil
	}
	a := float32(math.Round(float64(v)*10/float64(t)) / 10)
	return &a
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

func encodeMap[K comparable, V any](m map[K]V) map[K]V {
	if len(m) == 0 {
		return nil
	}

	return cloneMap(m)
}

func decodeMap[K comparable, V any](m map[K]V) map[K]V {
	return cloneMap(m)
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	result := make(map[K]V, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func toSqlProject(p *model.Project) *sqlProject {
	sp := &sqlProject{
		ID:           p.ID,
		Name:         p.String(),
		Root:         p.Root,
		ProjectName:  p.Name,
		NameParts:    p.NameParts,
		Type:         p.Type,
		RootDir:      p.RootDir,
		ProjectFile:  p.ProjectFile,
		RepositoryID: p.RepositoryID,
		Sizes:        map[string]*sqlSize{},
		Size:         toSqlSize(p.Size),
		Changes:      toSqlChanges(p.Changes),
		Metrics:      toSqlMetricsAggregate(p.Metrics, p.Size),
		Data:         encodeMap(p.Data),
		FirstSeen:    p.FirstSeen,
		LastSeen:     p.LastSeen,
	}

	for k, v := range p.Sizes {
		sp.Sizes[k] = toSqlSize(v)
	}

	if len(sp.Sizes) == 0 {
		sp.Sizes = nil
	}

	return sp
}

func toSqlProjectDependency(d *model.ProjectDependency) *sqlProjectDependency {
	return &sqlProjectDependency{
		ID:       d.ID,
		Name:     d.String(),
		SourceID: d.Source.ID,
		TargetID: d.Target.ID,
		Versions: d.Versions.Slice(),
		Data:     encodeMap(d.Data),
	}
}

func toSqlProjectDirectory(d *model.ProjectDirectory, p *model.Project) *sqlProjectDirectory {
	return &sqlProjectDirectory{
		ID:        d.ID,
		ProjectID: p.ID,
		Name:      d.RelativePath,
		Type:      d.Type,
		Size:      toSqlSize(d.Size),
		Changes:   toSqlChanges(d.Changes),
		Metrics:   toSqlMetricsAggregate(d.Metrics, d.Size),
		Data:      encodeMap(d.Data),
		FirstSeen: d.FirstSeen,
		LastSeen:  d.LastSeen,
	}
}

func toSqlFile(f *model.File) *sqlFile {
	return &sqlFile{
		ID:                 f.ID,
		Name:               f.Path,
		ProjectID:          f.ProjectID,
		ProjectDirectoryID: f.ProjectDirectoryID,
		RepositoryID:       f.RepositoryID,
		ProductAreaID:      f.ProductAreaID,
		Exists:             f.Exists,
		Size:               toSqlSize(f.Size),
		Changes:            toSqlChanges(f.Changes),
		Metrics:            toSqlMetrics(f.Metrics),
		Data:               encodeMap(f.Data),
		FirstSeen:          f.FirstSeen,
		LastSeen:           f.LastSeen,
	}
}

func toSqlFileLine(fileID model.UUID, f *model.FileLine) *sqlFileLine {
	return &sqlFileLine{
		FileID:       fileID,
		Line:         f.Line,
		ProjectID:    f.ProjectID,
		RepositoryID: f.RepositoryID,
		CommitID:     f.CommitID,
		AuthorID:     f.AuthorID,
		Date:         f.Date,
		CommitterID:  f.CommitterID,
		Type:         f.Type,
		Text:         f.Text,
	}
}

func toSqlPerson(p *model.Person) *sqlPerson {
	result := &sqlPerson{
		ID:        p.ID,
		Name:      p.Name,
		Names:     p.ListNames(),
		Emails:    p.ListEmails(),
		Changes:   toSqlChanges(p.Changes),
		Blame:     toSqlBlame(p.Blame),
		Data:      encodeMap(p.Data),
		FirstSeen: p.FirstSeen,
		LastSeen:  p.LastSeen,
	}

	return result
}

func toSqlPersonRepository(r *model.PersonRepository) *sqlPersonRepository {
	return &sqlPersonRepository{
		PersonID:     r.PersonID,
		RepositoryID: r.RepositoryID,
		FirstSeen:    r.FirstSeen,
		LastSeen:     r.LastSeen,
	}
}

func toSqlPersonFile(f *model.PersonFile) *sqlPersonFile {
	return &sqlPersonFile{
		PersonID:  f.PersonID,
		FileID:    f.FileID,
		FirstSeen: f.FirstSeen,
		LastSeen:  f.LastSeen,
	}
}

func toSqlProductArea(a *model.ProductArea) *sqlProductArea {
	return &sqlProductArea{
		ID:      a.ID,
		Name:    a.Name,
		Size:    toSqlSize(a.Size),
		Changes: toSqlChanges(a.Changes),
		Metrics: toSqlMetricsAggregate(a.Metrics, a.Size),
		Data:    encodeMap(a.Data),
	}
}

func toSqlRepository(r *model.Repository) *sqlRepository {
	return &sqlRepository{
		ID:           r.ID,
		Name:         r.Name,
		RootDir:      r.RootDir,
		VCS:          r.VCS,
		Data:         encodeMap(r.Data),
		FirstSeen:    r.FirstSeen,
		LastSeen:     r.LastSeen,
		CommitsTotal: r.CountCommits(),
		FilesTotal:   encodeMetric(r.FilesTotal),
		FilesHead:    encodeMetric(r.FilesHead),
	}
}

func toSqlRepositoryCommit(r *model.Repository, c *model.RepositoryCommit) *sqlRepositoryCommit {
	return &sqlRepositoryCommit{
		ID:            c.ID,
		RepositoryID:  r.ID,
		Name:          c.Hash,
		Message:       c.Message,
		Parents:       c.Parents,
		Children:      c.Children,
		Date:          c.Date,
		CommitterID:   c.CommitterID,
		DateAuthored:  c.DateAuthored,
		AuthorID:      c.AuthorID,
		Ignore:        c.Ignore,
		FilesModified: encodeMetric(c.FilesModified),
		FilesCreated:  encodeMetric(c.FilesCreated),
		FilesDeleted:  encodeMetric(c.FilesDeleted),
		LinesModified: encodeMetric(c.LinesModified),
		LinesAdded:    encodeMetric(c.LinesAdded),
		LinesDeleted:  encodeMetric(c.LinesDeleted),
		Blame:         toSqlBlame(c.Blame),
	}
}

func toSqlRepositoryCommitFile(c model.UUID, f *model.RepositoryCommitFile) *sqlRepositoryCommitFile {
	return &sqlRepositoryCommitFile{
		CommitID:      c,
		FileID:        f.FileID,
		Hash:          f.Hash,
		Change:        f.Change,
		OldFileIDs:    encodeOldFileIDs(f.OldFileIDs),
		LinesModified: encodeMetric(f.LinesModified),
		LinesAdded:    encodeMetric(f.LinesAdded),
		LinesDeleted:  encodeMetric(f.LinesDeleted),
	}
}

func encodeOldFileIDs(v map[model.UUID]model.UUID) string {
	var sb strings.Builder

	for k, v := range v {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(string(k))
		sb.WriteString(":")
		sb.WriteString(string(v))
	}

	return sb.String()
}
func decodeOldFileIDs(v string) map[model.UUID]model.UUID {
	result := make(map[model.UUID]model.UUID)
	if v == "" {
		return result
	}

	for _, line := range strings.Split(v, "\n") {
		cols := strings.Split(line, ":")
		result[model.UUID(cols[0])] = model.UUID(cols[1])
	}

	return result
}

func addMap[K comparable, V any](target *map[K]V, toAdd map[K]V) {
	for k, v := range toAdd {
		(*target)[k] = v
	}
}
func addList[K comparable, V any](target *map[K]V, toAdd []V, key func(V) K) {
	for _, v := range toAdd {
		(*target)[key(v)] = v
	}
}

func prepareChange[K comparable, V any](byID *map[K]V, id K, n V) bool {
	o, ok := (*byID)[id]
	if ok {
		ro := reflect.Indirect(reflect.ValueOf(o))
		rn := reflect.Indirect(reflect.ValueOf(n))

		rn.FieldByName("CreatedAt").Set(ro.FieldByName("CreatedAt"))
		rn.FieldByName("UpdatedAt").Set(ro.FieldByName("UpdatedAt"))
	}

	if reflect.DeepEqual(n, o) {
		return false
	} else {
		(*byID)[id] = n
		return true
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
