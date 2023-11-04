package sqlite

import (
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

	configs         map[string]*sqlConfig
	projs           map[model.UUID]*sqlProject
	projDeps        map[model.UUID]*sqlProjectDependency
	projDirs        map[model.UUID]*sqlProjectDirectory
	files           map[model.UUID]*sqlFile
	people          map[model.UUID]*sqlPerson
	area            map[model.UUID]*sqlProductArea
	orgs            map[model.UUID]*sqlOrg
	groups          map[model.UUID]*sqlOrgGroup
	teams           map[model.UUID]*sqlOrgTeam
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
		root := filepath.Dir(file)
		err = os.MkdirAll(root, 0o700)
		if err != nil {
			return nil, err
		}
	}

	return newFrom(file + "?_pragma=journal_mode(WAL)")
}

func NewSqliteMemoryStorage(file string) (archer.Storage, error) {
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
		&sqlPerson{}, &sqlOrg{}, &sqlOrgGroup{}, &sqlOrgTeam{}, &sqlOrgTeamMember{}, &sqlOrgTeamArea{}, &sqlProductArea{},
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
		p.Changes = toModelChanges(sp.Changes)
		p.Metrics = toModelMetricsAggregate(sp.Metrics)
		p.Data = cloneMap(sp.Data)
	}

	for _, sd := range deps {
		source := result.GetByID(sd.SourceID)
		target := result.GetByID(sd.TargetID)

		d := source.GetOrCreateDependency(target)
		d.ID = sd.ID
		d.Data = cloneMap(sd.Data)
	}

	for _, sd := range dirs {
		p := result.GetByID(sd.ProjectID)

		d := p.GetDirectory(sd.Name)
		d.ID = sd.ID
		d.Type = sd.Type
		d.Size = toModelSize(sd.Size)
		d.Changes = toModelChanges(sd.Changes)
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
	changedProjs := changed(changes, archer.ChangedBasicInfo, archer.ChangedData, archer.ChangedSize, archer.ChangedChanges, archer.ChangedMetrics)
	changedDeps := changed(changes, archer.ChangedDependencies, archer.ChangedData)
	changedDirs := changed(changes, archer.ChangedBasicInfo, archer.ChangedSize, archer.ChangedChanges, archer.ChangedMetrics)

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
		CreateBatchSize: 1000,
	})

	if changedProjs {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlProjs).Error
		if err != nil {
			return err
		}

		addList(&s.projs, sqlProjs, func(s *sqlProject) model.UUID { return s.ID })
	}

	if changedDeps {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDeps).Error
		if err != nil {
			return err
		}

		addList(&s.projDeps, sqlDeps, func(s *sqlProjectDependency) model.UUID { return s.ID })
	}

	if changedDirs {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlDirs).Error
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
		f := result.GetOrCreateFileEx(sf.Name, &sf.ID)
		f.ProjectID = sf.ProjectID
		f.ProjectDirectoryID = sf.ProjectDirectoryID
		f.RepositoryID = sf.RepositoryID
		f.ProductAreaID = sf.ProductAreaID
		f.OrganizationID = sf.OrgID
		f.GroupID = sf.OrgGroupID
		f.TeamID = sf.OrgTeamID
		f.Exists = sf.Exists
		f.Size = toModelSize(sf.Size)
		f.Changes = toModelChanges(sf.Changes)
		f.Metrics = toModelMetrics(sf.Metrics)
		f.Data = cloneMap(sf.Data)
	}

	return result, nil
}

func (s *sqliteStorage) WriteFiles(files *model.Files, changes archer.StorageChanges) error {
	changedFiles := changed(changes, archer.ChangedBasicInfo, archer.ChangedData,
		archer.ChangedSize, archer.ChangedChanges, archer.ChangedMetrics, archer.ChangedTeams)
	if !changedFiles {
		return nil
	}

	all := files.ListFiles()

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

	// TODO delete

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

		line.CommitID = sf.CommitID
		line.Type = sf.Type
		line.Text = sf.Text
	}

	return result, nil
}

func (s *sqliteStorage) WriteFileContents(contents *model.FileContents, changes archer.StorageChanges) error {
	changed := changed(changes, archer.ChangedBasicInfo, archer.ChangedChanges)
	if !changed {
		return nil
	}

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

	return db.Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlLines).Error
		if err != nil {
			return err
		}

		err = tx.Where("file_id = ? and line > ?", contents.FileID, len(contents.Lines)).Delete(&sqlFileLine{}).Error
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *sqliteStorage) ComputeBlamePerAuthor() ([]*archer.BlamePerAuthor, error) {
	var result []*archer.BlamePerAuthor

	err := s.db.Raw(`
		select c.author_id, l.commit_id, l.file_id, l.type line_type, count(*) lines
		from file_lines l
				 join repository_commits c
					  on l.commit_id = c.id
		group by c.author_id, l.commit_id, l.file_id, l.type
		`).Scan(&result).Error
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

	var orgs []*sqlOrg
	err = s.db.Find(&orgs).Error
	if err != nil {
		return nil, err
	}

	s.orgs = lo.Associate(orgs, func(i *sqlOrg) (model.UUID, *sqlOrg) {
		return i.ID, i
	})

	var groups []*sqlOrgGroup
	err = s.db.Find(&groups).Error
	if err != nil {
		return nil, err
	}

	s.groups = lo.Associate(groups, func(i *sqlOrgGroup) (model.UUID, *sqlOrgGroup) {
		return i.ID, i
	})

	var teams []*sqlOrgTeam
	err = s.db.Find(&teams).Error
	if err != nil {
		return nil, err
	}

	s.teams = lo.Associate(teams, func(i *sqlOrgTeam) (model.UUID, *sqlOrgTeam) {
		return i.ID, i
	})

	groupsByOrg := lo.GroupBy(groups, func(t *sqlOrgGroup) model.UUID { return t.OrgID })
	teamsByGroup := lo.GroupBy(teams, func(t *sqlOrgTeam) model.UUID { return t.OrgGroupID })

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
		p.Data = cloneMap(sp.Data)
	}

	for _, sa := range areas {
		a := result.GetOrCreateProductAreaEx(sa.Name, &sa.ID)
		a.Size = toModelSize(sa.Size)
		a.Changes = toModelChanges(sa.Changes)
		a.Metrics = toModelMetricsAggregate(sa.Metrics)
		a.Data = cloneMap(sa.Data)
	}

	for _, so := range orgs {
		o := result.GetOrCreateOrganizationEx(so.Name, &so.ID)
		o.Size = toModelSize(so.Size)
		o.Blame = toModelSize(so.Blame)
		o.Changes = toModelChanges(so.Changes)
		o.Metrics = toModelMetricsAggregate(so.Metrics)
		o.Data = cloneMap(so.Data)

		for _, sg := range groupsByOrg[so.ID] {
			g := o.GetOrCreateGroupEx(sg.Name, &sg.ID)
			g.Size = toModelSize(sg.Size)
			g.Blame = toModelSize(sg.Blame)
			g.Changes = toModelChanges(sg.Changes)
			g.Metrics = toModelMetricsAggregate(sg.Metrics)
			g.Data = cloneMap(sg.Data)

			for _, st := range teamsByGroup[sg.ID] {
				t := g.GetOrCreateTeamEx(st.Name, &st.ID)
				t.Size = toModelSize(st.Size)
				t.Blame = toModelSize(st.Blame)
				t.Changes = toModelChanges(st.Changes)
				t.Metrics = toModelMetricsAggregate(st.Metrics)
				t.Data = cloneMap(st.Data)
			}
		}
	}

	return result, nil
}

func (s *sqliteStorage) WritePeople(peopleDB *model.People, changes archer.StorageChanges) error {
	changedPeople := changed(changes, archer.ChangedBasicInfo, archer.ChangedData, archer.ChangedSize, archer.ChangedChanges, archer.ChangedMetrics)
	changedAreas := changed(changes, archer.ChangedBasicInfo, archer.ChangedData, archer.ChangedSize, archer.ChangedChanges, archer.ChangedMetrics)
	changedTeams := changed(changes, archer.ChangedTeams, archer.ChangedData, archer.ChangedSize, archer.ChangedChanges, archer.ChangedMetrics)

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

	var sqlAreas []*sqlProductArea
	if changedAreas {
		area := peopleDB.ListProductAreas()
		for _, p := range area {
			sp := toSqlProductArea(p)
			if prepareChange(&s.area, sp.ID, sp) {
				sqlAreas = append(sqlAreas, sp)
			}
		}
	}

	var sqlOrgs []*sqlOrg
	var sqlGroups []*sqlOrgGroup
	var sqlTeams []*sqlOrgTeam
	if changedTeams {
		orgs := peopleDB.ListOrganizations()
		for _, o := range orgs {
			so := toSqlOrg(o)
			if prepareChange(&s.orgs, so.ID, so) {
				sqlOrgs = append(sqlOrgs, so)
			}

			for _, g := range o.ListGroups() {
				sg := toSqlOrgGroup(g)
				if prepareChange(&s.groups, sg.ID, sg) {
					sqlGroups = append(sqlGroups, sg)
				}

				for _, t := range g.ListTeams() {
					st := toSqlOrgTeam(t)
					if prepareChange(&s.teams, st.ID, st) {
						sqlTeams = append(sqlTeams, st)
					}
				}
			}
		}
	}

	now := time.Now().Local()
	db := s.db.Session(&gorm.Session{
		NowFunc:         func() time.Time { return now },
		CreateBatchSize: 1000,
	})

	if changedPeople {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlPeople).Error
		if err != nil {
			return err
		}

		addList(&s.people, sqlPeople, func(s *sqlPerson) model.UUID { return s.ID })
	}

	if changedAreas {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlAreas).Error
		if err != nil {
			return err
		}

		addList(&s.area, sqlAreas, func(s *sqlProductArea) model.UUID { return s.ID })
	}

	if changedTeams {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlOrgs).Error
		if err != nil {
			return err
		}

		err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlGroups).Error
		if err != nil {
			return err
		}

		err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlTeams).Error
		if err != nil {
			return err
		}

		addList(&s.orgs, sqlOrgs, func(s *sqlOrg) model.UUID { return s.ID })
		addList(&s.groups, sqlGroups, func(s *sqlOrgGroup) model.UUID { return s.ID })
		addList(&s.teams, sqlTeams, func(s *sqlOrgTeam) model.UUID { return s.ID })
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
		c.SurvivedLines = decodeMetric(sc.SurvivedLines)

		commitsById[c.ID] = c
	}

	for _, sf := range commitFiles {
		commit := commitsById[sf.CommitID]

		file := commit.AddFile(sf.FileID, sf.OldFileID, sf.ModifiedLines, sf.AddedLines, sf.DeletedLines)
		file.SurvivedLines = decodeMetric(sf.SurvivedLines)
	}

	return result, nil
}

func (s *sqliteStorage) WriteRepository(repo *model.Repository, changes archer.StorageChanges) error {
	changedRepos := changed(changes, archer.ChangedBasicInfo)
	changedCommits := changed(changes, archer.ChangedHistory)
	changedFiles := changed(changes, archer.ChangedHistory)

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
		CreateBatchSize: 1000,
	})

	if changedRepos {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlRepos).Error
		if err != nil {
			return err
		}

		addList(&s.repos, sqlRepos, func(s *sqlRepository) model.UUID { return s.ID })
	}

	if changedCommits {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommits).Error
		if err != nil {
			return err
		}

		addList(&s.repoCommits, sqlCommits, func(s *sqlRepositoryCommit) model.UUID { return s.ID })
	}

	if changedFiles {
		err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&sqlCommitFiles).Error
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

func toSqlChanges(c *model.Changes) *sqlChanges {
	return &sqlChanges{
		Semester:      encodeMetric(c.In6Months),
		Total:         encodeMetric(c.Total),
		ModifiedLines: encodeMetric(c.ModifiedLines),
		AddedLines:    encodeMetric(c.AddedLines),
		DeletedLines:  encodeMetric(c.DeletedLines),
	}
}

func toModelChanges(sc *sqlChanges) *model.Changes {
	return &model.Changes{
		In6Months:     decodeMetric(sc.Semester),
		Total:         decodeMetric(sc.Total),
		ModifiedLines: decodeMetric(sc.ModifiedLines),
		AddedLines:    decodeMetric(sc.AddedLines),
		DeletedLines:  decodeMetric(sc.DeletedLines),
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
		Changes:     toSqlChanges(p.Changes),
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
		Changes:   toSqlChanges(d.Changes),
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
		ProductAreaID:      f.ProductAreaID,
		OrgID:              f.OrganizationID,
		OrgGroupID:         f.GroupID,
		OrgTeamID:          f.TeamID,
		Exists:             f.Exists,
		Size:               toSqlSize(f.Size),
		Changes:            toSqlChanges(f.Changes),
		Metrics:            toSqlMetrics(f.Metrics),
		Data:               cloneMap(f.Data),
	}
}

func toSqlFileLine(fileID model.UUID, f *model.FileLine) *sqlFileLine {
	return &sqlFileLine{
		FileID:   fileID,
		Line:     f.Line,
		CommitID: f.CommitID,
		Type:     f.Type,
		Text:     f.Text,
	}
}

func toSqlPerson(p *model.Person) *sqlPerson {
	result := &sqlPerson{
		ID:      p.ID,
		Name:    p.Name,
		Names:   p.ListNames(),
		Emails:  p.ListEmails(),
		Blame:   toSqlSize(p.Blame),
		Changes: toSqlChanges(p.Changes),
		Data:    cloneMap(p.Data),
	}

	return result
}

func toSqlProductArea(a *model.ProductArea) *sqlProductArea {
	return &sqlProductArea{
		ID:      a.ID,
		Name:    a.Name,
		Size:    toSqlSize(a.Size),
		Changes: toSqlChanges(a.Changes),
		Metrics: toSqlMetricsAggregate(a.Metrics, a.Size),
		Data:    cloneMap(a.Data),
	}
}

func toSqlOrg(o *model.Organization) *sqlOrg {
	return &sqlOrg{
		ID:      o.ID,
		Name:    o.Name,
		Size:    toSqlSize(o.Size),
		Blame:   toSqlSize(o.Blame),
		Changes: toSqlChanges(o.Changes),
		Metrics: toSqlMetricsAggregate(o.Metrics, o.Size),
		Data:    cloneMap(o.Data),
	}
}

func toSqlOrgGroup(g *model.Group) *sqlOrgGroup {
	return &sqlOrgGroup{
		ID:      g.ID,
		Name:    g.Name,
		Size:    toSqlSize(g.Size),
		Blame:   toSqlSize(g.Blame),
		Changes: toSqlChanges(g.Changes),
		Metrics: toSqlMetricsAggregate(g.Metrics, g.Size),
		Data:    cloneMap(g.Data),
	}
}

func toSqlOrgTeam(t *model.Team) *sqlOrgTeam {
	return &sqlOrgTeam{
		ID:      t.ID,
		Name:    t.Name,
		Size:    toSqlSize(t.Size),
		Blame:   toSqlSize(t.Blame),
		Changes: toSqlChanges(t.Changes),
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
		SurvivedLines: encodeMetric(c.SurvivedLines),
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
		SurvivedLines: encodeMetric(f.SurvivedLines),
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

func changed(changes archer.StorageChanges, desired ...archer.StorageChanges) bool {
	for _, d := range desired {
		if changes&d != 0 {
			return true
		}
	}

	return false
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
