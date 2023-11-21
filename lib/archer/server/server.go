package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/model"
)

type Options struct {
	Port uint
}

func Run(storage archer.Storage, opts *Options) error {
	s := newServer(opts)

	err := s.load(storage)
	if err != nil {
		return err
	}

	return s.run()
}

type server struct {
	opts *Options

	storage         archer.Storage
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

func (s *server) load(storage archer.Storage) error {
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
	r := gin.Default()

	s.initFiles(r)
	s.initProjects(r)
	s.initRepos(r)
	s.initPeople(r)

	return r.Run(fmt.Sprintf(":%v", s.opts.Port))
}
