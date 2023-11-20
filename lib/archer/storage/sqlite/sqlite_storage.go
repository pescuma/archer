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
	db *gorm.DB

	configs     map[string]*sqlConfig
	projs       map[model.UUID]*sqlProject
	projDeps    map[model.UUID]*sqlProjectDependency
	projDirs    map[model.UUID]*sqlProjectDirectory
	files       map[model.UUID]*sqlFile
	people      map[model.UUID]*sqlPerson
	peopleRepos map[string]*sqlPeopleRepository
	area        map[model.UUID]*sqlProductArea
	repos       map[model.UUID]*sqlRepository
	repoCommits map[model.UUID]*sqlRepositoryCommit
}

func NewSqliteStorage(file string) (archer.Storage, error) {
	if strings.HasSuffix(file, string(filepath.Separator)) {
		file = file + "archer.db"
	}

	if _, err := os.Stat(file); err != nil {
		fmt.Printf("Creating workspace at %v\n", file)
		root := filepath.Dir(file)
		err = os.MkdirAll(root, 0o700)
		if err != nil {
			return nil, err
		}
	}

	return newFrom(file + "?_pragma=journal_mode(WAL)")
}

func NewSqliteMemoryStorage(_ string) (archer.Storage, error) {
	return newFrom(":memory:")
}

func newFrom(dsn string) (archer.Storage, error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		NamingStrategy: &NamingStrategy{},
		Logger:         newLogger,
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&sqlConfig{},
		&sqlProject{}, &sqlProjectDependency{}, &sqlProjectDirectory{},
		&sqlFile{}, &sqlFileLine{},
		&sqlPerson{}, &sqlPeopleRepository{}, &sqlProductArea{},
		&sqlRepository{}, &sqlRepositoryCommit{}, &sqlRepositoryCommitFile{},
	)
	if err != nil {
		return nil, err
	}

	return &sqliteStorage{
		db:          db,
		configs:     map[string]*sqlConfig{},
		projs:       map[model.UUID]*sqlProject{},
		projDeps:    map[model.UUID]*sqlProjectDependency{},
		projDirs:    map[model.UUID]*sqlProjectDirectory{},
		files:       map[model.UUID]*sqlFile{},
		people:      map[model.UUID]*sqlPerson{},
		peopleRepos: map[string]*sqlPeopleRepository{},
		repos:       map[model.UUID]*sqlRepository{},
		repoCommits: map[model.UUID]*sqlRepositoryCommit{},
	}, nil
}

func (s *sqliteStorage) LoadProjects() (*model.Projects, error) {
	result := model.NewProjects()

	var projs []*sqlProject
	err := s.db.Find(&projs).Error
	if err != nil {
		return nil, err
	}

	s.projs = lo.Associate(projs, func(i *sqlProject) (model.UUID, *sqlProject) {
		return i.ID, i
	})

	var deps []*sqlProjectDependency
	err = s.db.Find(&deps).Error
	if err != nil {
		return nil, err
	}

	s.projDeps = lo.Associate(deps, func(i *sqlProjectDependency) (model.UUID, *sqlProjectDependency) {
		return i.ID, i
	})

	var dirs []*sqlProjectDirectory
	err = s.db.Find(&dirs).Error
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
	all := projs.ListProjects(model.FilterAll)

	return s.writeProjects(all)
}

func (s *sqliteStorage) WriteProject(proj *model.Project) error {
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
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
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

func (s *sqliteStorage) QueryProjects(file string, proj string, repo string, person string) ([]model.UUID, error) {
	var result []model.UUID

	err := s.db.Raw(`
select distinct p.id
from projects p
         left join files f
                   on f.project_id = p.id
         left join repositories r
                   on r.id = p.repository_id
         left join people_repositories pr
                   on pr.repository_id = r.id
         left join people pe
                   on pe.id = pr.person_id
where (@proj = '' or p.name like @proj)
  and (@file = '' or f.name like @file)
  and (@repo = '' or r.name like @repo)
  and (@person = '' or pe.names like @person or pe.emails like @person)
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

func (s *sqliteStorage) LoadFiles() (*model.Files, error) {
	result := model.NewFiles()

	var files []*sqlFile
	err := s.db.Find(&files).Error
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
	all := files.ListFiles()

	err := s.writeFiles(all)
	if err != nil {
		return err
	}

	// TODO delete

	return nil
}

func (s *sqliteStorage) WriteFile(file *model.File) error {
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
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlFiles).Error
	if err != nil {
		return err
	}

	addList(&s.files, sqlFiles, func(s *sqlFile) model.UUID { return s.ID })

	return nil
}

func (s *sqliteStorage) LoadFileContents(fileID model.UUID) (*model.FileContents, error) {
	result := model.NewFileContents(fileID)

	var lines []*sqlFileLine
	err := s.db.Where("file_id = ?", fileID).Find(&lines).Error
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

		line.AuthorID = sf.AuthorID
		line.CommitID = sf.CommitID
		line.Type = sf.Type
		line.Text = sf.Text
	}

	return result, nil
}

func (s *sqliteStorage) WriteFileContents(contents *model.FileContents) error {
	var sqlLines []*sqlFileLine
	for _, f := range contents.Lines {
		sf := toSqlFileLine(contents.FileID, f)
		sqlLines = append(sqlLines, sf)
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
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
	var result []*archer.BlamePerAuthor

	err := s.db.Raw(`
select author_id, commit_id, file_id, type line_type, count(*) lines
from file_lines l
group by author_id, commit_id, file_id, type
	`).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *sqliteStorage) QueryFiles(file string, proj string, repo string, person string) ([]model.UUID, error) {
	var result []model.UUID

	err := s.db.Raw(`
select distinct f.id
from files f
         left join projects p
                   on p.id = f.project_id
         left join repositories r
                   on r.id = f.repository_id
         left join people_repositories pr
                   on pr.repository_id = r.id
         left join people pe
                   on pe.id = pr.person_id
where (@proj = '' or p.name like @proj)
  and (@file = '' or f.name like @file)
  and (@repo = '' or r.name like @repo)
  and (@person = '' or pe.names like @person or pe.emails like @person)
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

func (s *sqliteStorage) LoadPeople() (*model.People, error) {
	result := model.NewPeople()

	var people []*sqlPerson
	err := s.db.Find(&people).Error
	if err != nil {
		return nil, err
	}

	s.people = lo.Associate(people, func(i *sqlPerson) (model.UUID, *sqlPerson) {
		return i.ID, i
	})

	var areas []*sqlProductArea
	err = s.db.Find(&areas).Error
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
		p.Blame = toModelSize(sp.Blame)
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
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
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

func (s *sqliteStorage) LoadPeopleRepositories() (*model.PeopleRepositories, error) {
	result := model.NewPeopleRepositories()

	var sprs []*sqlPeopleRepository
	err := s.db.Find(&sprs).Error
	if err != nil {
		return nil, err
	}

	s.peopleRepos = lo.Associate(sprs, func(i *sqlPeopleRepository) (string, *sqlPeopleRepository) {
		return string(i.PersonID) + "\n" + string(i.RepositoryID), i
	})

	for _, spr := range sprs {
		pr := result.GetOrCreatePerson(spr.PersonID).GetOrCreateRepository(spr.RepositoryID)
		pr.FirstSeen = spr.FirstSeen
		pr.LastSeen = spr.LastSeen
	}

	return result, nil
}

func (s *sqliteStorage) WritePeopleRepositories(prs *model.PeopleRepositories) error {
	key := func(pr *sqlPeopleRepository) string { return string(pr.PersonID) + "\n" + string(pr.RepositoryID) }

	var sqlPeopleRepositories []*sqlPeopleRepository
	for _, p := range prs.List() {
		for _, r := range p.List() {
			pr := toSqlPersonRepository(p, r)
			if prepareChange(&s.peopleRepos, key(pr), pr) {
				sqlPeopleRepositories = append(sqlPeopleRepositories, pr)
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlPeopleRepositories).Error
	if err != nil {
		return err
	}

	// TODO delete

	addList(&s.peopleRepos, sqlPeopleRepositories, key)

	return nil
}

func (s *sqliteStorage) QueryPeople(file string, proj string, repo string, person string) ([]model.UUID, error) {
	var result []model.UUID

	err := s.db.Raw(`
select distinct pe.id
from people pe
         left join people_repositories pr
                   on pr.person_id = pe.id
         left join repositories r
                   on r.id = pr.repository_id
         left join repository_commits c
                   on c.repository_id = r.id
                       and (c.author_id = pe.id or c.committer_id = pe.id)
         left join repository_commit_files cf
                   on cf.commit_id = c.id
         left join files f
                   on f.id = cf.file_id
         left join projects p
                   on p.id = f.project_id
where (c.ignore is null or c.ignore = 0)
  and (@proj = '' or p.name like @proj)
  and (@file = '' or f.name like @file)
  and (@repo = '' or r.name like @repo)
  and (@person = '' or pe.names like @person or pe.emails)
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

func (s *sqliteStorage) LoadRepositories() (*model.Repositories, error) {
	return s.loadRepositories(
		func([]*sqlRepository) func(db *gorm.DB) *gorm.DB {
			return func(db *gorm.DB) *gorm.DB {
				return db
			}
		})
}

func (s *sqliteStorage) LoadRepository(rootDir string) (*model.Repository, error) {
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
	err := s.db.Scopes(scope(repos)).Find(&repos).Error
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
	err = s.db.Scopes(scope(repos)).Find(&commits).Error
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
		c.FilesModified = decodeMetric(sc.FilesModified)
		c.FilesCreated = decodeMetric(sc.FilesCreated)
		c.FilesDeleted = decodeMetric(sc.FilesDeleted)
		c.LinesModified = decodeMetric(sc.LinesModified)
		c.LinesAdded = decodeMetric(sc.LinesAdded)
		c.LinesDeleted = decodeMetric(sc.LinesDeleted)
		c.LinesSurvived = decodeMetric(sc.LinesSurvived)
		c.Ignore = sc.Ignore

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
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
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
	var sqlCommits []*sqlRepositoryCommit

	sc := toSqlRepositoryCommit(repo, commit)
	if prepareChange(&s.repoCommits, sc.ID, sc) {
		sqlCommits = append(sqlCommits, sc)
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
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
	var commitFiles []*sqlRepositoryCommitFile
	err := s.db.Where("commit_id = ?", commit.ID).Find(&commitFiles).Error
	if err != nil {
		return nil, err
	}

	result := model.NewRepositoryCommitFiles(repo.ID, commit.ID)
	for _, sf := range commitFiles {
		file := result.GetOrCreate(sf.FileID)
		file.OldFileIDs = decodeOldFileIDs(sf.OldFileIDs)
		file.LinesModified = decodeMetric(sf.LinesModified)
		file.LinesAdded = decodeMetric(sf.LinesAdded)
		file.LinesDeleted = decodeMetric(sf.LinesDeleted)
		file.LinesSurvived = decodeMetric(sf.LinesSurvived)
	}
	return result, nil
}

func (s *sqliteStorage) WriteRepositoryCommitFiles(files []*model.RepositoryCommitFiles) error {
	var sqlCommitFiles []*sqlRepositoryCommitFile
	for _, fs := range files {
		for _, f := range fs.List() {
			sf := toSqlRepositoryCommitFile(fs.RepositoryID, fs.CommitID, f)
			sqlCommitFiles = append(sqlCommitFiles, sf)
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommitFiles).Error
	if err != nil {
		return err
	}

	// TODO delete

	return nil
}

func (s *sqliteStorage) QueryRepositories(file string, proj string, repo string, person string) ([]model.UUID, error) {
	var result []model.UUID

	err := s.db.Raw(`
select distinct r.id
from repositories r
         left join people_repositories pr
                   on pr.repository_id = r.id
         left join people pe
                   on pe.id = pr.person_id
         left join files f
                   on f.repository_id = r.id
         left join projects p
                   on p.repository_id = r.id
where (@proj = '' or p.name like @proj)
  and (@file = '' or f.name like @file)
  and (@repo = '' or r.name like @repo)
  and (@person = '' or pe.names like @person or pe.emails like @person)
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

func (s *sqliteStorage) QueryCommits(file string, proj string, repo string, person string) ([]model.UUID, error) {
	var result []model.UUID

	err := s.db.Raw(`
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

func (s *sqliteStorage) QuerySurvivedLines(file string, proj string, repo string, person string) ([]*archer.SurvivedLineCount, error) {
	var result []*archer.SurvivedLineCount

	if file == "" {
		err := s.db.Raw(`
select l.month, l.type line_type, sum(l.lines) lines
from file_lines_month l
         join repositories r
              on r.id = l.repository_id
         join people pa
              on pa.id = l.author_id
         join people pc
              on pc.id = l.committer_id
         left join projects p
                   on p.id = l.project_id
where (@proj = '' or p.name like @proj)
  and (@repo = '' or r.name like @repo)
  and (@person = '' or pc.names like @person or pc.emails like @person or pa.names like @person or pa.emails like @person)
group by 1, 2
		`,
			sql.Named("proj", "%"+proj+"%"),
			sql.Named("repo", "%"+repo+"%"),
			sql.Named("person", "%"+person+"%"),
		).Scan(&result).Error
		if err != nil {
			return nil, err
		}

	} else {
		err := s.db.Raw(`
select strftime('%Y-%m', c.date) month, l.type line_type, count(1) lines
from files f
         join file_lines l
              on l.file_id = f.id
         left join projects p
                   on p.id = f.project_id
         join repository_commits c
              on c.id = l.commit_id
         join repositories r
              on r.id = c.repository_id
         join people pa
              on pa.id = c.author_id
         join people pc
              on pc.id = c.committer_id
where c.ignore = 0
  and f.name like @file
  and (@proj = '' or p.name like @proj)
  and (@repo = '' or r.name like @repo)
  and (@person = '' or pc.names like @person or pc.emails like @person or pa.names like @person or pa.emails like @person)
group by 1, 2
		`,
			sql.Named("proj", "%"+proj+"%"),
			sql.Named("file", "%"+file+"%"),
			sql.Named("repo", "%"+repo+"%"),
			sql.Named("person", "%"+person+"%"),
		).Scan(&result).Error
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *sqliteStorage) WriteSurvivedLinesCache() error {
	err := s.db.Exec(`drop table file_lines_month`).Error
	if err != nil {
		return err
	}

	err = s.db.Exec(`
create table file_lines_month as
select strftime('%Y-%m', c.date) month,
       p.id                      project_id,
       r.id                      repository_id,
       pa.id                     author_id,
       pc.id                     committer_id,
       l.type                    ,
       count(1)                  lines
from file_lines l
         join repository_commits c
              on c.id = l.commit_id
         join repositories r
              on r.id = c.repository_id
         join people pa
              on pa.id = c.author_id
         join people pc
              on pc.id = c.committer_id
         join files f
              on f.id = l.file_id
         left join projects p
                   on p.id = f.project_id
where c.ignore = 0
group by 1,
         p.id,
         r.id,
         pa.id,
         pc.id,
         l.type
`).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *sqliteStorage) LoadConfig() (*map[string]string, error) {
	result := map[string]string{}

	var sqlConfigs []*sqlConfig
	err := s.db.Find(&sqlConfigs).Error
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
	var sqlConfigs []*sqlConfig
	for k, v := range *configs {
		sc := toSqlConfig(k, v)
		if prepareChange(&s.configs, sc.Key, sc) {
			sqlConfigs = append(sqlConfigs, sc)
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlConfigs).Error
	if err != nil {
		return err
	}

	addList(&s.configs, sqlConfigs, func(s *sqlConfig) string { return s.Key })

	// TODO delete

	return nil
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
		FileID:   fileID,
		Line:     f.Line,
		AuthorID: f.AuthorID,
		CommitID: f.CommitID,
		Type:     f.Type,
		Text:     f.Text,
	}
}

func toSqlPerson(p *model.Person) *sqlPerson {
	result := &sqlPerson{
		ID:        p.ID,
		Name:      p.Name,
		Names:     p.ListNames(),
		Emails:    p.ListEmails(),
		Blame:     toSqlSize(p.Blame),
		Changes:   toSqlChanges(p.Changes),
		Data:      encodeMap(p.Data),
		FirstSeen: p.FirstSeen,
		LastSeen:  p.LastSeen,
	}

	return result
}

func toSqlPersonRepository(p *model.PersonRepositories, r *model.PersonRepository) *sqlPeopleRepository {
	return &sqlPeopleRepository{
		PersonID:     p.PersonID,
		RepositoryID: r.RepositoryID,
		FirstSeen:    r.FirstSeen,
		LastSeen:     r.LastSeen,
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
		FilesModified: encodeMetric(c.FilesModified),
		FilesCreated:  encodeMetric(c.FilesCreated),
		FilesDeleted:  encodeMetric(c.FilesDeleted),
		LinesModified: encodeMetric(c.LinesModified),
		LinesAdded:    encodeMetric(c.LinesAdded),
		LinesDeleted:  encodeMetric(c.LinesDeleted),
		LinesSurvived: encodeMetric(c.LinesSurvived),
		Ignore:        c.Ignore,
	}
}

func toSqlRepositoryCommitFile(r model.UUID, c model.UUID, f *model.RepositoryCommitFile) *sqlRepositoryCommitFile {
	return &sqlRepositoryCommitFile{
		CommitID:      c,
		FileID:        f.FileID,
		OldFileIDs:    encodeOldFileIDs(f.OldFileIDs),
		RepositoryID:  r,
		LinesModified: encodeMetric(f.LinesModified),
		LinesAdded:    encodeMetric(f.LinesAdded),
		LinesDeleted:  encodeMetric(f.LinesDeleted),
		LinesSurvived: encodeMetric(f.LinesSurvived),
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
