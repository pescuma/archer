package orm

import (
	"fmt"
	"log"
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

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
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
	sqlPersonRepos      map[string]*sqlPersonRepository
	sqlPersonFiles      map[string]*sqlPersonFile
	sqlAreas            map[string]*sqlProductArea
	sqlRepos            map[string]*sqlRepository
	sqlRepoCommits      map[string]*sqlRepositoryCommit
	sqlRepoCommitFiles  map[string]*sqlRepositoryCommitFile
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
		&sqlRepository{},
		&sqlRepositoryCommit{},
		&sqlRepositoryCommitFile{}, &sqlRepositoryCommitFileDetails{},
		&sqlRepositoryCommitPerson{},
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
			p.Sizes[k] = v.ToModel()
		}
		p.Size = sp.Size.ToModel()
		p.Changes = sp.Changes.ToModel()
		p.Metrics = sp.Metrics.ToModel()
		p.Data = decodeMap(sp.Data)
		p.FirstSeen = sp.FirstSeen
		p.LastSeen = sp.LastSeen
	}

	for _, sd := range deps {
		source := result.GetByID(sd.SourceID)
		target := result.GetByID(sd.TargetID)

		d := source.GetOrCreateDependencyEx(&sd.ID, target)
		d.Versions.InsertSlice(sd.Versions)
		d.Data = decodeMap(sd.Data)
	}

	for _, sd := range dirs {
		p := result.GetByID(sd.ProjectID)

		d := p.GetDirectoryEx(&sd.ID, sd.Name)
		d.Type = sd.Type
		d.Size = sd.Size.ToModel()
		d.Changes = sd.Changes.ToModel()
		d.Metrics = sd.Metrics.ToModel()
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

	sqlProjs := prepareChanges(projs, newSqlProject, &s.sqlProjs)

	var sqlDeps []*sqlProjectDependency
	for _, p := range projs {
		for _, d := range p.Dependencies {
			sd := newSqlProjectDependency(d)
			if prepareChange(&s.sqlProjDeps, sd) {
				sqlDeps = append(sqlDeps, sd)
			}
		}
	}

	var sqlDirs []*sqlProjectDirectory
	for _, p := range projs {
		for _, d := range p.Dirs {
			sd := newSqlProjectDirectory(d, p)
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
		result.AddFromStorage(sf.ToModel())
	}

	s.files = result
	return result, nil
}

func (s *gormStorage) WriteFiles() error {
	if s.files == nil {
		return nil
	}

	return s.writeFiles(s.files.List())
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
		p.Blame = sp.Blame.ToModel()
		p.Changes = sp.Changes.ToModel()
		p.Data = decodeMap(sp.Data)
		p.FirstSeen = sp.FirstSeen
		p.LastSeen = sp.LastSeen
	}

	for _, sa := range areas {
		a := result.GetOrCreateProductAreaEx(sa.Name, &sa.ID)
		a.Size = sa.Size.ToModel()
		a.Changes = sa.Changes.ToModel()
		a.Metrics = sa.Metrics.ToModel()
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

	sqlPeople := prepareChanges(s.people.ListPeople(), newSqlPerson, &s.sqlPeople)
	sqlAreas := prepareChanges(s.people.ListProductAreas(), newSqlProductArea, &s.sqlAreas)

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

	s.sqlPersonRepos = createCache(rs)

	var fs []*sqlPersonFile
	err = s.db.Find(&fs).Error
	if err != nil {
		return nil, err
	}

	s.sqlPersonFiles = createCache(fs)

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

	rs := prepareChanges(s.peopleRelations.ListRepositories(), newSqlPersonRepository, &s.sqlPersonRepos)
	fs := prepareChanges(s.peopleRelations.ListFiles(), newSqlPersonFile, &s.sqlPersonFiles)

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

	addList(&s.sqlPersonRepos, rs)
	addList(&s.sqlPersonFiles, fs)

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

	var scfs []*sqlRepositoryCommitFile
	err = s.db.Find(&scfs).Error
	if err != nil {
		return nil, err
	}

	s.sqlRepoCommitFiles = createCache(scfs)

	var scps []*sqlRepositoryCommitPerson
	err = s.db.Find(&scps).Error
	if err != nil {
		return nil, err
	}

	s.sqlRepoCommitPeople = createCache(scps)

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

	commitsById := map[model.ID]*model.RepositoryCommit{}
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
		c.Blame = sc.Blame.ToModel()

		commitsById[c.ID] = c
	}

	for _, scf := range scfs {
		commit := commitsById[scf.CommitID]

		cf := model.NewRepositoryCommitFile(scf.FileID)
		cf.Change = scf.Change
		cf.LinesModified = decodeMetric(scf.LinesModified)
		cf.LinesAdded = decodeMetric(scf.LinesAdded)
		cf.LinesDeleted = decodeMetric(scf.LinesDeleted)

		commit.Files[scf.FileID] = cf
	}

	sort.Slice(scps, func(i, j int) bool {
		return scps[i].Order < scps[j].Order
	})
	for _, scp := range scps {
		commit := commitsById[scp.CommitID]

		switch scp.Role {
		case CommitRoleAuthor:
			commit.AuthorIDs = append(commit.AuthorIDs, scp.PersonID)
		case CommitRoleCommitter:
			commit.CommitterID = scp.PersonID
		default:
			panic(fmt.Sprintf("invalid role: %v", scp.Role))
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

	sqlRepos := prepareChanges(repos, newSqlRepository, &s.sqlRepos)

	var sqlCommits []*sqlRepositoryCommit
	var sqlCommitFiles []*sqlRepositoryCommitFile
	var sqlCommitPeople []*sqlRepositoryCommitPerson
	for _, repo := range repos {
		for _, c := range repo.ListCommits() {
			sc := newSqlRepositoryCommit(repo, c)
			if prepareChange(&s.sqlRepoCommits, sc) {
				sqlCommits = append(sqlCommits, sc)
			}

			for _, f := range c.Files {
				cf := newSqlRepositoryCommitFile(c, f)
				if prepareChange(&s.sqlRepoCommitFiles, cf) {
					sqlCommitFiles = append(sqlCommitFiles, cf)
				}
			}

			cp := newSqlRepositoryCommitPerson(c, c.CommitterID, CommitRoleCommitter, 1)
			if prepareChange(&s.sqlRepoCommitPeople, cp) {
				sqlCommitPeople = append(sqlCommitPeople, cp)
			}

			for i, a := range c.AuthorIDs {
				cp = newSqlRepositoryCommitPerson(c, a, CommitRoleAuthor, i+1)
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

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommitFiles).Error
	if err != nil {
		return err
	}

	addList(&s.sqlRepoCommitFiles, sqlCommitFiles)

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
	var sqlCommitFiles []*sqlRepositoryCommitFile
	var sqlCommitPeople []*sqlRepositoryCommitPerson

	sc := newSqlRepositoryCommit(repo, commit)
	if prepareChange(&s.sqlRepoCommits, sc) {
		sqlCommits = append(sqlCommits, sc)
	}

	for _, f := range commit.Files {
		cf := newSqlRepositoryCommitFile(commit, f)
		if prepareChange(&s.sqlRepoCommitFiles, cf) {
			sqlCommitFiles = append(sqlCommitFiles, cf)
		}
	}

	cp := newSqlRepositoryCommitPerson(commit, commit.CommitterID, CommitRoleCommitter, 1)
	if prepareChange(&s.sqlRepoCommitPeople, cp) {
		sqlCommitPeople = append(sqlCommitPeople, cp)
	}

	for i, a := range commit.AuthorIDs {
		cp = newSqlRepositoryCommitPerson(commit, a, CommitRoleAuthor, i+1)
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

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommitFiles).Error
	if err != nil {
		return err
	}

	addList(&s.sqlRepoCommitFiles, sqlCommitFiles)

	err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommitPeople).Error
	if err != nil {
		return err
	}

	addList(&s.sqlRepoCommitPeople, sqlCommitPeople)

	// TODO delete

	return nil
}

func (s *gormStorage) LoadRepositoryCommitDetails(repo *model.Repository, commit *model.RepositoryCommit) (*model.RepositoryCommitDetails, error) {
	if s.repos == nil {
		return nil, errors.New("repos not loaded")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var commitFiles []*sqlRepositoryCommitFileDetails
	err := s.db.Where("commit_id = ?", commit.ID).Find(&commitFiles).Error
	if err != nil {
		return nil, err
	}

	result := model.NewRepositoryCommitDetails(repo.ID, commit.ID)
	for _, sf := range commitFiles {
		file := result.GetOrCreateFile(sf.FileID)
		file.Hash = sf.Hash
		file.OldIDs = decodeOldFileIDs(sf.OldIDs)
		file.OldHashes = decodeOldFileHashes(sf.OldHashes)
	}
	return result, nil
}

func (s *gormStorage) WriteRepositoryCommitDetails(details []*model.RepositoryCommitDetails) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sqlFiles []*sqlRepositoryCommitFileDetails
	for _, d := range details {
		for _, f := range d.ListFiles() {
			sf := newSqlRepositoryCommitFileDetails(d.CommitID, f)
			sqlFiles = append(sqlFiles, sf)
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 300,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlFiles).Error
	if err != nil {
		return err
	}

	// TODO delete

	return nil
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
		l.Changes = sl.Changes.ToModel()
		l.Blame = sl.Blame.ToModel()
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

	sqlLines := prepareChanges(s.stats.ListLines(), newSqlMonthLines, &s.monthLines)

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
		sc := newSqlConfig(k, v)
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
		result.AddFromStorage(sr.ToModel())
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

func compositeKey(ids ...string) string {
	return strings.Join(ids, "\n")
}
