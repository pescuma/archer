package sqlite

import (
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type sqliteStorage struct {
	db *gorm.DB

	configs         map[string]*sqlConfig
	projs           map[model.UUID]*sqlProject
	projDeps        map[model.UUID]*sqlProjectDependency
	projDirs        map[model.UUID]*sqlProjectDirectory
	files           map[model.UUID]*sqlFile
	people          map[model.UUID]*sqlPerson
	teams           map[model.UUID]*sqlTeam
	repos           map[model.UUID]*sqlRepository
	repoCommits     map[model.UUID]*sqlRepositoryCommit
	repoCommitFiles map[model.UUID]*sqlRepositoryCommitFile
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

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(sqlite.Open(file), &gorm.Config{
		NamingStrategy: &NamingStrategy{},
		Logger:         newLogger,
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&sqlConfig{},
		&sqlProject{}, &sqlProjectDependency{}, &sqlProjectDirectory{},
		&sqlFile{},
		&sqlPerson{}, &sqlTeam{},
		&sqlRepository{}, &sqlRepositoryCommit{}, &sqlRepositoryCommitFile{},
	)
	if err != nil {
		return nil, err
	}

	return &sqliteStorage{
		db:              db,
		configs:         map[string]*sqlConfig{},
		projs:           map[model.UUID]*sqlProject{},
		projDeps:        map[model.UUID]*sqlProjectDependency{},
		projDirs:        map[model.UUID]*sqlProjectDirectory{},
		files:           map[model.UUID]*sqlFile{},
		people:          map[model.UUID]*sqlPerson{},
		repos:           map[model.UUID]*sqlRepository{},
		repoCommits:     map[model.UUID]*sqlRepositoryCommit{},
		repoCommitFiles: map[model.UUID]*sqlRepositoryCommitFile{},
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

		for k, v := range sp.Sizes {
			p.Sizes[k] = toModelSize(v)
		}
		p.Metrics = toModelMetricsAggregate(sp.Metrics)
		p.Data = cloneMap(sp.Data)
	}

	for _, sd := range deps {
		source := result.GetByID(sd.SourceID)
		target := result.GetByID(sd.TargetID)

		d := source.GetDependency(target)
		d.ID = sd.ID
		d.Data = cloneMap(sd.Data)
	}

	for _, sd := range dirs {
		p := result.GetByID(sd.ProjectID)

		d := p.GetDirectory(sd.Name)
		d.ID = sd.ID
		d.Type = sd.Type
		d.Size = toModelSize(sd.Size)
		d.Metrics = toModelMetricsAggregate(sd.Metrics)
		d.Data = cloneMap(sd.Data)
	}

	return result, nil
}

func (s *sqliteStorage) WriteProjects(projs *model.Projects, changes archer.StorageChanges) error {
	all := projs.ListProjects(model.FilterAll)

	return s.writeProjects(all, changes)
}

func (s *sqliteStorage) WriteProject(proj *model.Project, changes archer.StorageChanges) error {
	projs := []*model.Project{proj}

	return s.writeProjects(projs, changes)
}

func (s *sqliteStorage) writeProjects(projs []*model.Project, changes archer.StorageChanges) error {
	changedProjs := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedData != 0 || changes&archer.ChangedSize != 0 || changes&archer.ChangedMetrics != 0
	changedDeps := changes&archer.ChangedDependencies != 0 || changes&archer.ChangedData != 0
	changedDirs := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedSize != 0 || changes&archer.ChangedMetrics != 0

	var sqlProjs []*sqlProject
	if changedProjs {
		for _, p := range projs {
			sp := toSqlProject(p)
			if prepareChange(&s.projs, sp.ID, sp) {
				sqlProjs = append(sqlProjs, sp)
			}
		}
	}

	var sqlDeps []*sqlProjectDependency
	if changedDeps {
		for _, p := range projs {
			for _, d := range p.Dependencies {
				sd := toSqlProjectDependency(d)
				if prepareChange(&s.projDeps, sd.ID, sd) {
					sqlDeps = append(sqlDeps, sd)
				}
			}
		}
	}

	var sqlDirs []*sqlProjectDirectory
	if changedDirs {
		for _, p := range projs {
			for _, d := range p.Dirs {
				sd := toSqlProjectDirectory(d, p)
				if prepareChange(&s.projDirs, sd.ID, sd) {
					sqlDirs = append(sqlDirs, sd)
				}
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 100,
	})

	if changedProjs {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlProjs, 1000).Error
		if err != nil {
			return err
		}

		addList(&s.projs, sqlProjs, func(s *sqlProject) model.UUID { return s.ID })
	}

	if changedDeps {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlDeps, 1000).Error
		if err != nil {
			return err
		}

		addList(&s.projDeps, sqlDeps, func(s *sqlProjectDependency) model.UUID { return s.ID })
	}

	if changedDirs {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlDirs, 1000).Error
		if err != nil {
			return err
		}

		addList(&s.projDirs, sqlDirs, func(s *sqlProjectDirectory) model.UUID { return s.ID })
	}

	// TODO delete

	return nil
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
		f := result.GetOrCreateEx(sf.Name, &sf.ID)
		f.ProjectID = sf.ProjectID
		f.ProjectDirectoryID = sf.ProjectDirectoryID
		f.RepositoryID = sf.RepositoryID
		f.TeamID = sf.TeamID
		f.Exists = sf.Exists
		f.Size = toModelSize(sf.Size)
		f.Metrics = toModelMetrics(sf.Metrics)
		f.Data = cloneMap(sf.Data)
	}

	return result, nil
}

func (s *sqliteStorage) WriteFiles(files *model.Files, changes archer.StorageChanges) error {
	changedFiles := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedData != 0 ||
		changes&archer.ChangedSize != 0 || changes&archer.ChangedMetrics != 0 ||
		changes&archer.ChangedTeams != 0
	if !changedFiles {
		return nil
	}

	all := files.List()

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
		CreateBatchSize: 100,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlFiles, 1000).Error
	if err != nil {
		return err
	}

	addList(&s.files, sqlFiles, func(s *sqlFile) model.UUID { return s.ID })

	// TODO delete

	return nil
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

	var teams []*sqlTeam
	err = s.db.Find(&teams).Error
	if err != nil {
		return nil, err
	}

	s.teams = lo.Associate(teams, func(i *sqlTeam) (model.UUID, *sqlTeam) {
		return i.ID, i
	})

	for _, st := range teams {
		t := result.GetOrCreateTeamEx(st.Name, &st.ID)
		t.Size = toModelSize(st.Size)
		t.Metrics = toModelMetricsAggregate(st.Metrics)
		t.Data = cloneMap(st.Data)
	}

	for _, sp := range people {
		p := result.GetOrCreatePersonEx(sp.Name, &sp.ID)

		for _, name := range sp.Names {
			p.AddName(name)
		}
		for _, email := range sp.Emails {
			p.AddEmail(email)
		}
		p.Data = cloneMap(sp.Data)

		if sp.TeamID != nil {
			p.Team = result.GetTeamByID(*sp.TeamID)
		}
	}

	return result, nil
}

func (s *sqliteStorage) WritePeople(peopleDB *model.People, changes archer.StorageChanges) error {
	changedPeople := changes&archer.ChangedBasicInfo != 0 || changes&archer.ChangedData != 0
	changedTeams := changes&archer.ChangedTeams != 0 || changes&archer.ChangedData != 0 || changes&archer.ChangedSize != 0 || changes&archer.ChangedMetrics != 0

	var sqlPeople []*sqlPerson
	if changedPeople {
		people := peopleDB.ListPeople()
		for _, p := range people {
			sp := toSqlPerson(p)
			if prepareChange(&s.people, sp.ID, sp) {
				sqlPeople = append(sqlPeople, sp)
			}
		}
	}

	var sqlTeams []*sqlTeam
	if changedTeams {
		teams := peopleDB.ListTeams()
		for _, t := range teams {
			st := toSqlTeam(t)
			if prepareChange(&s.teams, st.ID, st) {
				sqlTeams = append(sqlTeams, st)
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 100,
	})

	if changedPeople {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlPeople, 1000).Error
		if err != nil {
			return err
		}

		addList(&s.people, sqlPeople, func(s *sqlPerson) model.UUID { return s.ID })
	}

	if changedTeams {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlTeams, 1000).Error
		if err != nil {
			return err
		}

		addList(&s.teams, sqlTeams, func(s *sqlTeam) model.UUID { return s.ID })
	}

	// TODO delete

	return nil
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

	var commitFiles []*sqlRepositoryCommitFile
	err = s.db.Scopes(scope(repos)).Find(&commitFiles).Error
	if err != nil {
		return nil, err
	}

	addMap(&s.repoCommitFiles, lo.Associate(commitFiles, func(i *sqlRepositoryCommitFile) (model.UUID, *sqlRepositoryCommitFile) {
		return i.CommitID + "\n" + i.FileID, i
	}))

	for _, sr := range repos {
		r := result.GetOrCreateEx(sr.RootDir, &sr.ID)
		r.Name = sr.Name
		r.VCS = sr.VCS
		r.Data = cloneMap(sr.Data)
	}

	commitsById := map[model.UUID]*model.RepositoryCommit{}
	for _, sc := range commits {
		repo := result.GetByID(sc.RepositoryID)

		c := repo.GetCommit(sc.Name)
		c.ID = sc.ID
		c.Message = sc.Message
		c.Parents = sc.Parents
		c.Date = sc.Date
		c.CommitterID = sc.CommitterID
		c.DateAuthored = sc.DateAuthored
		c.AuthorID = sc.AuthorID
		c.ModifiedLines = sc.ModifiedLines
		c.AddedLines = sc.AddedLines
		c.DeletedLines = sc.DeletedLines

		commitsById[c.ID] = c
	}

	for _, sf := range commitFiles {
		commit := commitsById[sf.CommitID]

		commit.AddFile(sf.FileID, sf.OldFileID, sf.ModifiedLines, sf.AddedLines, sf.DeletedLines)
	}

	return result, nil
}

func (s *sqliteStorage) WriteRepository(repo *model.Repository, changes archer.StorageChanges) error {
	changedRepos := changes&archer.ChangedBasicInfo != 0
	changedCommits := changes&archer.ChangedHistory != 0
	changedFiles := changes&archer.ChangedHistory != 0

	var sqlRepos []*sqlRepository
	if changedRepos {
		sr := toSqlRepository(repo)
		if prepareChange(&s.repos, sr.ID, sr) {
			sqlRepos = append(sqlRepos, sr)
		}
	}

	var sqlCommits []*sqlRepositoryCommit
	if changedCommits {
		for _, c := range repo.ListCommits() {
			sc := toSqlRepositoryCommit(repo, c)
			if prepareChange(&s.repoCommits, sc.ID, sc) {
				sqlCommits = append(sqlCommits, sc)
			}
		}
	}

	var sqlCommitFiles []*sqlRepositoryCommitFile
	if changedFiles {
		for _, c := range repo.ListCommits() {
			for _, f := range c.Files {
				sf := toSqlRepositoryCommitFile(repo, c, f)
				if prepareChange(&s.repoCommitFiles, sf.CommitID+"\n"+sf.FileID, sf) {
					sqlCommitFiles = append(sqlCommitFiles, sf)
				}
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 100,
	})

	if changedRepos {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlRepos, 1000).Error
		if err != nil {
			return err
		}

		addList(&s.repos, sqlRepos, func(s *sqlRepository) model.UUID { return s.ID })
	}

	if changedCommits {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlCommits, 1000).Error
		if err != nil {
			return err
		}

		addList(&s.repoCommits, sqlCommits, func(s *sqlRepositoryCommit) model.UUID { return s.ID })
	}

	if changedFiles {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlCommitFiles, 1000).Error
		if err != nil {
			return err
		}

		addList(&s.repoCommitFiles, sqlCommitFiles, func(s *sqlRepositoryCommitFile) model.UUID { return s.CommitID + "\n" + s.FileID })
	}

	// TODO delete

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
		CreateBatchSize: 100,
	})

	err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&sqlConfigs, 1000).Error
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
		DependenciesGuice:    encodeMetric(metrics.GuiceDependencies),
		ComplexityCyclomatic: encodeMetric(metrics.CyclomaticComplexity),
		ComplexityCognitive:  encodeMetric(metrics.CognitiveComplexity),
		ChangesSemester:      encodeMetric(metrics.ChangesIn6Months),
		ChangesTotal:         encodeMetric(metrics.ChangesTotal),
	}
}

func toModelMetrics(metrics *sqlMetrics) *model.Metrics {
	return &model.Metrics{
		GuiceDependencies:    decodeMetric(metrics.DependenciesGuice),
		CyclomaticComplexity: decodeMetric(metrics.ComplexityCyclomatic),
		CognitiveComplexity:  decodeMetric(metrics.ComplexityCognitive),
		ChangesIn6Months:     decodeMetric(metrics.ChangesSemester),
		ChangesTotal:         decodeMetric(metrics.ChangesTotal),
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
		ChangesSemester:           encodeMetric(metrics.ChangesIn6Months),
		ChangesTotal:              encodeMetric(metrics.ChangesTotal),
	}
}

func toModelMetricsAggregate(metrics *sqlMetricsAggregate) *model.Metrics {
	return &model.Metrics{
		GuiceDependencies:    decodeMetric(metrics.DependenciesGuiceTotal),
		CyclomaticComplexity: decodeMetric(metrics.ComplexityCyclomaticTotal),
		CognitiveComplexity:  decodeMetric(metrics.ComplexityCognitiveTotal),
		ChangesIn6Months:     decodeMetric(metrics.ChangesSemester),
		ChangesTotal:         decodeMetric(metrics.ChangesTotal),
	}
}

func encodeMetricAggregate(v int, t int) *float32 {
	if v == -1 {
		return nil
	}
	if t == 0 {
		return nil
	}
	a := float32(math.Round(float64(v)*10/float64(t)) * 10)
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

func toSqlProject(p *model.Project) *sqlProject {
	size := p.GetSize()

	sp := &sqlProject{
		ID:          p.ID,
		Name:        p.String(),
		Root:        p.Root,
		ProjectName: p.Name,
		NameParts:   p.NameParts,
		Type:        p.Type,
		RootDir:     p.RootDir,
		ProjectFile: p.ProjectFile,
		Size:        toSqlSize(size),
		Sizes:       map[string]*sqlSize{},
		Metrics:     toSqlMetricsAggregate(p.Metrics, size),
		Data:        cloneMap(p.Data),
	}

	for k, v := range p.Sizes {
		sp.Sizes[k] = toSqlSize(v)
	}

	return sp
}

func toSqlProjectDependency(d *model.ProjectDependency) *sqlProjectDependency {
	return &sqlProjectDependency{
		ID:       d.ID,
		Name:     d.String(),
		SourceID: d.Source.ID,
		TargetID: d.Target.ID,
		Data:     cloneMap(d.Data),
	}
}

func toSqlProjectDirectory(d *model.ProjectDirectory, p *model.Project) *sqlProjectDirectory {
	return &sqlProjectDirectory{
		ID:        d.ID,
		ProjectID: p.ID,
		Name:      d.RelativePath,
		Type:      d.Type,
		Size:      toSqlSize(d.Size),
		Metrics:   toSqlMetricsAggregate(d.Metrics, d.Size),
		Data:      cloneMap(d.Data),
	}
}

func toSqlFile(f *model.File) *sqlFile {
	return &sqlFile{
		ID:                 f.ID,
		Name:               f.Path,
		ProjectID:          f.ProjectID,
		ProjectDirectoryID: f.ProjectDirectoryID,
		RepositoryID:       f.RepositoryID,
		TeamID:             f.TeamID,
		Exists:             f.Exists,
		Size:               toSqlSize(f.Size),
		Metrics:            toSqlMetrics(f.Metrics),
		Data:               cloneMap(f.Data),
	}
}

func toSqlPerson(p *model.Person) *sqlPerson {
	result := &sqlPerson{
		ID:     p.ID,
		Name:   p.Name,
		Names:  p.ListNames(),
		Emails: p.ListEmails(),
		Data:   cloneMap(p.Data),
	}

	if p.Team != nil {
		result.TeamID = &p.Team.ID
	}

	return result
}

func toSqlTeam(t *model.Team) *sqlTeam {
	return &sqlTeam{
		ID:      t.ID,
		Name:    t.Name,
		Size:    toSqlSize(t.Size),
		Metrics: toSqlMetricsAggregate(t.Metrics, t.Size),
		Data:    cloneMap(t.Data),
	}
}

func toSqlRepository(r *model.Repository) *sqlRepository {
	return &sqlRepository{
		ID:      r.ID,
		Name:    r.Name,
		RootDir: r.RootDir,
		VCS:     r.VCS,
		Data:    cloneMap(r.Data),
	}
}

func toSqlRepositoryCommit(r *model.Repository, c *model.RepositoryCommit) *sqlRepositoryCommit {
	return &sqlRepositoryCommit{
		ID:            c.ID,
		RepositoryID:  r.ID,
		Name:          c.Hash,
		Message:       c.Message,
		Parents:       c.Parents,
		Date:          c.Date,
		CommitterID:   c.CommitterID,
		DateAuthored:  c.DateAuthored,
		AuthorID:      c.AuthorID,
		ModifiedLines: c.ModifiedLines,
		AddedLines:    c.AddedLines,
		DeletedLines:  c.DeletedLines,
	}
}

func toSqlRepositoryCommitFile(r *model.Repository, c *model.RepositoryCommit, f *model.RepositoryCommitFile) *sqlRepositoryCommitFile {
	return &sqlRepositoryCommitFile{
		CommitID:      c.ID,
		FileID:        f.FileID,
		OldFileID:     f.OldFileID,
		RepositoryID:  r.ID,
		ModifiedLines: f.ModifiedLines,
		AddedLines:    f.AddedLines,
		DeletedLines:  f.DeletedLines,
	}
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

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	result := make(map[K]V, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
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
