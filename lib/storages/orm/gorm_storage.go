package orm

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
)

type gormStorage struct {
	mutex   sync.RWMutex
	db      *gorm.DB
	console consoles.Console

	projects        *model.Projects
	files           *model.Files
	people          *model.People
	peopleRelations *model.PeopleRelations
	repos           *model.Repositories
	stats           *model.MonthlyStats
	config          *map[string]string
	ignoreRules     *model.IgnoreRules

	sqlConfigs          map[string]*sqlConfig
	sqlProjs            map[string]*sqlProject
	sqlProjDeps         map[string]*sqlProjectDependency
	sqlProjDirs         map[string]*sqlProjectDirectory
	sqlFiles            map[string]*sqlFile
	sqlPeople           map[string]*sqlPerson
	sqlPeopleRepos      map[string]*sqlPersonRepository
	sqlPeopleFiles      map[string]*sqlPersonFile
	sqlAreas            map[string]*sqlProductArea
	sqlRepos            map[string]*sqlRepository
	sqlRepoCommits      map[string]*sqlRepositoryCommit
	sqlRepoCommitPeople map[string]*sqlRepositoryCommitPerson
	monthLines          map[string]*sqlMonthLines
	sqlIgnoreRules      map[string]*sqlIgnoreRule
}

func NewGormStorage(d gorm.Dialector, console consoles.Console) (storages.Storage, error) {
	l := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(d, &gorm.Config{
		NamingStrategy: &NamingStrategy{},
		Logger:         l,
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&sqlConfig{},
		&sqlProject{}, &sqlProjectDependency{}, &sqlProjectDirectory{},
		&sqlFile{},
		&sqlPerson{}, &sqlPersonRepository{}, &sqlPersonFile{}, &sqlProductArea{},
		&sqlRepository{}, &sqlRepositoryCommit{}, &sqlRepositoryCommitFile{}, &sqlRepositoryCommitPerson{},
		&sqlMonthLines{},
		&sqlFileLine{},
		&sqlIgnoreRule{},
	)
	if err != nil {
		return nil, err
	}

	return &gormStorage{
		db:      db,
		console: console,
	}, nil
}

func (s *gormStorage) Close() error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}

	return db.Close()
}

func createCache[T sqlTable](rows []T) map[string]T {
	return lo.Associate(rows, func(i T) (string, T) {
		return i.CacheKey(), i
	})
}

func (s *gormStorage) LoadProjects() (*model.Projects, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.projects != nil {
		return s.projects, nil
	}

	s.console.Printf("Loading projects...\n")

	result := model.NewProjects()

	var projs []*sqlProject
	err := s.db.Find(&projs).Error
	if err != nil {
		return nil, err
	}

	s.sqlProjs = createCache(projs)

	var deps []*sqlProjectDependency
	err = s.db.Find(&deps).Error
	if err != nil {
		return nil, err
	}

	s.sqlProjDeps = createCache(deps)

	var dirs []*sqlProjectDirectory
	err = s.db.Find(&dirs).Error
	if err != nil {
		return nil, err
	}

	s.sqlProjDirs = createCache(dirs)

	for _, sp := range projs {
		p := result.GetOrCreateEx(sp.ProjectName, &sp.ID)
		p.Groups = sp.Groups
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

	s.projects = result
	return result, nil
}

func (s *gormStorage) WriteProjects() error {
	if s.projects == nil {
		return nil
	}

	return s.writeProjects(s.projects.ListProjects(model.FilterAll))
}

func (s *gormStorage) WriteProject(proj *model.Project) error {
	if s.projects == nil {
		return errors.New("projects not loaded")
	}

	return s.writeProjects([]*model.Project{proj})
}

func (s *gormStorage) writeProjects(projs []*model.Project) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sqlProjs []*sqlProject
	for _, p := range projs {
		sp := toSqlProject(p)
		if prepareChange(&s.sqlProjs, sp) {
			sqlProjs = append(sqlProjs, sp)
		}
	}

	var sqlDeps []*sqlProjectDependency
	for _, p := range projs {
		for _, d := range p.Dependencies {
			sd := toSqlProjectDependency(d)
			if prepareChange(&s.sqlProjDeps, sd) {
				sqlDeps = append(sqlDeps, sd)
			}
		}
	}

	var sqlDirs []*sqlProjectDirectory
	for _, p := range projs {
		for _, d := range p.Dirs {
			sd := toSqlProjectDirectory(d, p)
			if prepareChange(&s.sqlProjDirs, sd) {
				sqlDirs = append(sqlDirs, sd)
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlProjs).Error
	if err != nil {
		return err
	}

	addList(&s.sqlProjs, sqlProjs)

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDeps).Error
	if err != nil {
		return err
	}

	addList(&s.sqlProjDeps, sqlDeps)

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDirs).Error
	if err != nil {
		return err
	}

	addList(&s.sqlProjDirs, sqlDirs)

	// TODO delete

	return nil
}

func (s *gormStorage) LoadFiles() (*model.Files, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.files != nil {
		return s.files, nil
	}

	s.console.Printf("Loading files...\n")

	result := model.NewFiles()

	var files []*sqlFile
	err := s.db.Find(&files).Error
	if err != nil {
		return nil, err
	}

	s.sqlFiles = createCache(files)

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

	s.files = result
	return result, nil
}

func (s *gormStorage) WriteFiles() error {
	if s.files == nil {
		return nil
	}

	return s.writeFiles(s.files.ListFiles())
}

func (s *gormStorage) WriteFile(file *model.File) error {
	if s.files == nil {
		return errors.New("files not loaded")
	}

	return s.writeFiles([]*model.File{file})
}

func (s *gormStorage) writeFiles(all []*model.File) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	sqlFiles := prepareChanges(all, newSqlFile, &s.sqlFiles)

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlFiles).Error
	if err != nil {
		return err
	}

	addList(&s.sqlFiles, sqlFiles)

	// TODO delete

	return nil
}

func (s *gormStorage) LoadFileContents(fileID model.ID) (*model.FileContents, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

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

func (s *gormStorage) WriteFileContents(contents *model.FileContents) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sqlLines []*sqlFileLine
	for _, f := range contents.Lines {
		sf := newSqlFileLine(contents.FileID, f)
		sqlLines = append(sqlLines, sf)
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
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

func (s *gormStorage) QueryBlamePerAuthor() ([]*storages.BlamePerAuthor, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []*storages.BlamePerAuthor

	err := s.db.Raw(`
		select author_id, committer_id, repository_id, commit_id, file_id, type line_type, count(*) lines
		from file_lines
		group by author_id, committer_id, repository_id, commit_id, file_id, type
	`).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *gormStorage) LoadPeople() (*model.People, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.people != nil {
		return s.people, nil
	}

	s.console.Printf("Loading people...\n")

	result := model.NewPeople()

	var people []*sqlPerson
	err := s.db.Find(&people).Error
	if err != nil {
		return nil, err
	}

	s.sqlPeople = createCache(people)

	var areas []*sqlProductArea
	err = s.db.Find(&areas).Error
	if err != nil {
		return nil, err
	}

	s.sqlAreas = createCache(areas)

	for _, sp := range people {
		p := result.GetOrCreatePerson(&sp.ID)
		p.Name = sp.Name
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

	s.people = result
	return result, nil
}

func (s *gormStorage) WritePeople() error {
	if s.people == nil {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sqlPeople []*sqlPerson
	people := s.people.ListPeople()
	for _, p := range people {
		sp := toSqlPerson(p)
		if prepareChange(&s.sqlPeople, sp) {
			sqlPeople = append(sqlPeople, sp)
		}
	}

	var sqlAreas []*sqlProductArea
	area := s.people.ListProductAreas()
	for _, p := range area {
		sp := toSqlProductArea(p)
		if prepareChange(&s.sqlAreas, sp) {
			sqlAreas = append(sqlAreas, sp)
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlPeople).Error
	if err != nil {
		return err
	}

	addList(&s.sqlPeople, sqlPeople)

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlAreas).Error
	if err != nil {
		return err
	}

	addList(&s.sqlAreas, sqlAreas)

	// TODO delete

	return nil
}

func (s *gormStorage) LoadPeopleRelations() (*model.PeopleRelations, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.peopleRelations != nil {
		return s.peopleRelations, nil
	}

	result := model.NewPeopleRelations()

	var rs []*sqlPersonRepository
	err := s.db.Find(&rs).Error
	if err != nil {
		return nil, err
	}

	s.sqlPeopleRepos = createCache(rs)

	var fs []*sqlPersonFile
	err = s.db.Find(&fs).Error
	if err != nil {
		return nil, err
	}

	s.sqlPeopleFiles = createCache(fs)

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

	s.peopleRelations = result
	return result, nil
}

func (s *gormStorage) WritePeopleRelations() error {
	if s.peopleRelations == nil {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var rs []*sqlPersonRepository
	for _, r := range s.peopleRelations.ListRepositories() {
		pr := toSqlPersonRepository(r)
		if prepareChange(&s.sqlPeopleRepos, pr) {
			rs = append(rs, pr)
		}
	}

	var fs []*sqlPersonFile
	for _, f := range s.peopleRelations.ListFiles() {
		pf := toSqlPersonFile(f)
		if prepareChange(&s.sqlPeopleFiles, pf) {
			fs = append(fs, pf)
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
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

	addList(&s.sqlPeopleRepos, rs)
	addList(&s.sqlPeopleFiles, fs)

	return nil
}

func (s *gormStorage) LoadRepositories() (*model.Repositories, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.repos != nil {
		return s.repos, nil
	}

	s.console.Printf("Loading repositories...\n")

	result := model.NewRepositories()

	var repos []*sqlRepository
	err := s.db.Find(&repos).Error
	if err != nil {
		return nil, err
	}

	s.sqlRepos = createCache(repos)

	var commits []*sqlRepositoryCommit
	err = s.db.Find(&commits).Error
	if err != nil {
		return nil, err
	}

	s.sqlRepoCommits = createCache(commits)

	var cps []*sqlRepositoryCommitPerson
	err = s.db.Find(&cps).Error
	if err != nil {
		return nil, err
	}

	s.sqlRepoCommitPeople = createCache(cps)

	for _, sr := range repos {
		r := result.GetOrCreateEx(sr.RootDir, &sr.ID)
		r.Name = sr.Name
		r.VCS = sr.VCS
		r.Branch = sr.Branch
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
		c.DateAuthored = sc.DateAuthored
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

	sort.Slice(cps, func(i, j int) bool {
		return cps[i].Order < cps[j].Order
	})
	for _, cp := range cps {
		commit := commitsById[cp.CommitID]

		switch cp.Role {
		case CommitRoleAuthor:
			commit.AuthorIDs = append(commit.AuthorIDs, cp.PersonID)
		case CommitRoleCommitter:
			commit.CommitterID = cp.PersonID
		default:
			panic(fmt.Sprintf("invalid role: %v", cp.Role))
		}
	}

	s.repos = result
	return result, nil
}

func (s *gormStorage) WriteRepositories() error {
	if s.repos == nil {
		return nil
	}

	return s.writeRepositories(s.repos.List())
}

func (s *gormStorage) WriteRepository(repo *model.Repository) error {
	if s.repos == nil {
		return errors.New("repos not loaded")
	}

	return s.writeRepositories([]*model.Repository{repo})
}

func (s *gormStorage) writeRepositories(repos []*model.Repository) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sqlRepos []*sqlRepository
	for _, repo := range repos {
		sr := toSqlRepository(repo)
		if prepareChange(&s.sqlRepos, sr) {
			sqlRepos = append(sqlRepos, sr)
		}
	}

	var sqlCommits []*sqlRepositoryCommit
	var sqlCommitPeople []*sqlRepositoryCommitPerson
	for _, repo := range repos {
		for _, c := range repo.ListCommits() {
			sc := toSqlRepositoryCommit(repo, c)
			if prepareChange(&s.sqlRepoCommits, sc) {
				sqlCommits = append(sqlCommits, sc)
			}

			cp := toSqlRepositoryCommitPerson(c, c.CommitterID, CommitRoleCommitter, 1)
			if prepareChange(&s.sqlRepoCommitPeople, cp) {
				sqlCommitPeople = append(sqlCommitPeople, cp)
			}

			for i, a := range c.AuthorIDs {
				cp = toSqlRepositoryCommitPerson(c, a, CommitRoleAuthor, i+1)
				if prepareChange(&s.sqlRepoCommitPeople, cp) {
					sqlCommitPeople = append(sqlCommitPeople, cp)
				}
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlRepos).Error
	if err != nil {
		return err
	}

	addList(&s.sqlRepos, sqlRepos)

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommits).Error
	if err != nil {
		return err
	}

	addList(&s.sqlRepoCommits, sqlCommits)

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommitPeople).Error
	if err != nil {
		return err
	}

	addList(&s.sqlRepoCommitPeople, sqlCommitPeople)

	// TODO delete

	return nil
}

func (s *gormStorage) WriteCommit(repo *model.Repository, commit *model.RepositoryCommit) error {
	if s.repos == nil {
		return errors.New("repos not loaded")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sqlCommits []*sqlRepositoryCommit
	var sqlCommitPeople []*sqlRepositoryCommitPerson

	sc := toSqlRepositoryCommit(repo, commit)
	if prepareChange(&s.sqlRepoCommits, sc) {
		sqlCommits = append(sqlCommits, sc)
	}

	cp := toSqlRepositoryCommitPerson(commit, commit.CommitterID, CommitRoleCommitter, 1)
	if prepareChange(&s.sqlRepoCommitPeople, cp) {
		sqlCommitPeople = append(sqlCommitPeople, cp)
	}

	for i, a := range commit.AuthorIDs {
		cp = toSqlRepositoryCommitPerson(commit, a, CommitRoleAuthor, i+1)
		if prepareChange(&s.sqlRepoCommitPeople, cp) {
			sqlCommitPeople = append(sqlCommitPeople, cp)
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommits).Error
	if err != nil {
		return err
	}

	addList(&s.sqlRepoCommits, sqlCommits)

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommitPeople).Error
	if err != nil {
		return err
	}

	addList(&s.sqlRepoCommitPeople, sqlCommitPeople)

	// TODO delete

	return nil
}

func (s *gormStorage) LoadRepositoryCommitFiles(repo *model.Repository, commit *model.RepositoryCommit) (*model.RepositoryCommitFiles, error) {
	if s.repos == nil {
		return nil, errors.New("repos not loaded")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var commitFiles []*sqlRepositoryCommitFile
	err := s.db.Where("commit_id = ?", commit.ID).Find(&commitFiles).Error
	if err != nil {
		return nil, err
	}

	result := model.NewRepositoryCommitFiles(repo.ID, commit.ID)
	for _, sf := range commitFiles {
		file := result.GetOrCreate(sf.FileID)
		file.Hash = sf.Hash
		file.Change = sf.Change
		file.OldIDs = decodeOldFileIDs(sf.OldIDs)
		file.OldHashes = decodeOldFileHashes(sf.OldHashes)
		file.LinesModified = decodeMetric(sf.LinesModified)
		file.LinesAdded = decodeMetric(sf.LinesAdded)
		file.LinesDeleted = decodeMetric(sf.LinesDeleted)
	}
	return result, nil
}

func (s *gormStorage) WriteRepositoryCommitFiles(files []*model.RepositoryCommitFiles) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sqlCommitFiles []*sqlRepositoryCommitFile
	for _, fs := range files {
		for _, f := range fs.List() {
			sf := toSqlRepositoryCommitFile(fs.CommitID, f)
			sqlCommitFiles = append(sqlCommitFiles, sf)
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
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

func (s *gormStorage) QueryCommits(file string, proj string, repo string, person string) ([]model.UUID, error) {
	if s.repos == nil {
		return nil, errors.New("repos not loaded")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

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

func (s *gormStorage) LoadMonthlyStats() (*model.MonthlyStats, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.stats != nil {
		return s.stats, nil
	}

	s.console.Printf("Loading monthly stats...\n")

	result := model.NewMonthlyStats()

	var sqlLines []*sqlMonthLines
	err := s.db.Find(&sqlLines).Error
	if err != nil {
		return nil, err
	}

	s.monthLines = createCache(sqlLines)

	for _, sl := range sqlLines {
		l := result.GetOrCreateLinesEx(&sl.ID, sl.Month, sl.RepositoryID, sl.AuthorID, sl.CommitterID, sl.ProjectID)
		l.Changes = toModelChanges(sl.Changes)
		l.Blame = toModelBlame(sl.Blame)
	}

	s.stats = result
	return result, nil
}

func (s *gormStorage) WriteMonthlyStats() error {
	if s.stats == nil {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sqlLines []*sqlMonthLines
	for _, f := range s.stats.ListLines() {
		sf := toSqlMonthLines(f)
		if prepareChange(&s.monthLines, sf) {
			sqlLines = append(sqlLines, sf)
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlLines).Error
	if err != nil {
		return err
	}

	addList(&s.monthLines, sqlLines)

	// TODO delete

	return nil
}

func (s *gormStorage) LoadConfig() (*map[string]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.config != nil {
		return s.config, nil
	}

	s.console.Printf("Loading config...\n")

	result := map[string]string{}

	var sqlConfigs []*sqlConfig
	err := s.db.Find(&sqlConfigs).Error
	if err != nil {
		return nil, err
	}

	s.sqlConfigs = createCache(sqlConfigs)

	for _, sc := range sqlConfigs {
		result[sc.Key] = sc.Value
	}

	s.config = &result
	return &result, nil
}

func (s *gormStorage) WriteConfig() error {
	if s.config == nil {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sqlConfigs []*sqlConfig
	for k, v := range *s.config {
		sc := toSqlConfig(k, v)
		if prepareChange(&s.sqlConfigs, sc) {
			sqlConfigs = append(sqlConfigs, sc)
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlConfigs).Error
	if err != nil {
		return err
	}

	addList(&s.sqlConfigs, sqlConfigs)

	// TODO delete

	return nil
}

func (s *gormStorage) LoadIgnoreRules() (*model.IgnoreRules, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.ignoreRules != nil {
		return s.ignoreRules, nil
	}

	s.console.Printf("Loading ignore rules...\n")

	result := model.NewIgnoreRules()

	var sqlIgnoreRules []*sqlIgnoreRule
	err := s.db.Find(&sqlIgnoreRules).Error
	if err != nil {
		return nil, err
	}

	s.sqlIgnoreRules = createCache(sqlIgnoreRules)

	for _, sr := range sqlIgnoreRules {
		result.AddRuleEx(sr.ToModel())
	}

	s.ignoreRules = result
	return result, nil
}

func (s *gormStorage) WriteIgnoreRules() error {
	if s.ignoreRules == nil {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	sqlIgnoreRules := prepareChanges(s.ignoreRules.ListRules(), newSqlIgnoreRule, &s.sqlIgnoreRules)

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlIgnoreRules).Error
	if err != nil {
		return err
	}

	addList(&s.sqlIgnoreRules, sqlIgnoreRules)

	return nil
}

func toSqlMonthLines(l *model.MonthlyStatsLine) *sqlMonthLines {
	return &sqlMonthLines{
		ID:           l.ID,
		Month:        l.Month,
		RepositoryID: l.RepositoryID,
		AuthorID:     l.AuthorID,
		CommitterID:  l.CommitterID,
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
		ProjectName:  p.Name,
		Groups:       p.Groups,
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
		Branch:       r.Branch,
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
		DateAuthored:  c.DateAuthored,
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

func toSqlRepositoryCommitPerson(commit *model.RepositoryCommit, personID model.ID, role CommitRole, order int) *sqlRepositoryCommitPerson {
	return &sqlRepositoryCommitPerson{
		CommitID: commit.ID,
		PersonID: personID,
		Role:     role,
		Order:    order,
	}
}

func toSqlRepositoryCommitFile(c model.UUID, f *model.RepositoryCommitFile) *sqlRepositoryCommitFile {
	return &sqlRepositoryCommitFile{
		CommitID:      c,
		FileID:        f.FileID,
		Hash:          f.Hash,
		Change:        f.Change,
		OldIDs:        encodeOldFileIDs(f.OldIDs),
		OldHashes:     encodeOldFileHashes(f.OldHashes),
		LinesModified: encodeMetric(f.LinesModified),
		LinesAdded:    encodeMetric(f.LinesAdded),
		LinesDeleted:  encodeMetric(f.LinesDeleted),
	}
}

func encodeOldFileIDs(v map[model.UUID]model.ID) string {
	var sb strings.Builder

	for k, v := range v {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(string(k))
		sb.WriteString(":")
		sb.WriteString(v.String())
	}

	return sb.String()
}
func decodeOldFileIDs(v string) map[model.UUID]model.ID {
	result := make(map[model.UUID]model.ID)
	if v == "" {
		return result
	}

	for _, line := range strings.Split(v, "\n") {
		cols := strings.Split(line, ":")
		result[model.UUID(cols[0])] = model.MustStringToID(cols[1])
	}

	return result
}

func encodeOldFileHashes(v map[model.UUID]string) string {
	var sb strings.Builder

	for k, v := range v {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(string(k))
		sb.WriteString(":")
		sb.WriteString(v)
	}

	return sb.String()
}
func decodeOldFileHashes(v string) map[model.UUID]string {
	result := make(map[model.UUID]string)
	if v == "" {
		return result
	}

	for _, line := range strings.Split(v, "\n") {
		cols := strings.Split(line, ":")
		result[model.UUID(cols[0])] = cols[1]
	}

	return result
}

func addMap[K comparable, V any](target *map[K]V, toAdd map[K]V) {
	for k, v := range toAdd {
		(*target)[k] = v
	}
}
func addList[T sqlTable](target *map[string]T, toAdd []T) {
	for _, v := range toAdd {
		(*target)[v.CacheKey()] = v
	}
}

func prepareChanges[S sqlTable, M any](models []M, toSql func(M) S, cache *map[string]S) []S {
	var result []S
	for _, m := range models {
		s := toSql(m)
		if prepareChange(cache, s) {
			result = append(result, s)
		}
	}
	return result
}

func prepareChange[T sqlTable](byID *map[string]T, n T) bool {
	o, ok := (*byID)[n.CacheKey()]
	if ok {
		ro := reflect.Indirect(reflect.ValueOf(o))
		rn := reflect.Indirect(reflect.ValueOf(n))

		rn.FieldByName("CreatedAt").Set(ro.FieldByName("CreatedAt"))
		rn.FieldByName("UpdatedAt").Set(ro.FieldByName("UpdatedAt"))
	}

	if reflect.DeepEqual(n, o) {
		return false
	} else {
		(*byID)[n.CacheKey()] = n
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
