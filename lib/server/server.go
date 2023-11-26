package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
)

type Options struct {
	Port uint
}

func Run(console consoles.Console, storage storages.Storage, opts *Options) error {
	s := newServer(opts)

	console.Printf("Loading existing data...\n")

	err := s.load(storage)
	if err != nil {
		return err
	}

	console.Printf("Starting server on port %v...\n", s.opts.Port)

	return s.run()
}

type server struct {
	opts *Options

	storage         storages.Storage
	people          *model.People
	peopleRelations *model.PeopleRelations
	files           *model.Files
	projects        *model.Projects
	repos           *model.Repositories
	commits         map[model.UUID]*model.RepositoryCommit
	stats           *model.MonthlyStats
}

func newServer(opts *Options) *server {
	if opts == nil {
		opts = &Options{}
	}
	if opts.Port == 0 {
		opts.Port = 2427
	}

	return &server{
		opts: opts,
	}
}

func (s *server) load(storage storages.Storage) error {
	var err error

	s.storage = storage

	s.people, err = storage.LoadPeople()
	if err != nil {
		return err
	}

	s.peopleRelations, err = storage.LoadPeopleRelations()
	if err != nil {
		return err
	}

	s.files, err = storage.LoadFiles()
	if err != nil {
		return err
	}

	s.projects, err = storage.LoadProjects()
	if err != nil {
		return err
	}

	s.repos, err = storage.LoadRepositories()
	if err != nil {
		return err
	}

	s.commits = make(map[model.UUID]*model.RepositoryCommit)
	for _, repo := range s.repos.List() {
		for _, commit := range repo.ListCommits() {
			s.commits[commit.ID] = commit
		}
	}

	s.stats, err = storage.LoadMonthlyStats()
	if err != nil {
		return err
	}

	return nil
}

func (s *server) run() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	s.initFiles(r)
	s.initProjects(r)
	s.initRepos(r)
	s.initPeople(r)
	s.initArch(r)

	return r.Run(fmt.Sprintf(":%v", s.opts.Port))
}
