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

	files    *model.Files
	projects *model.Projects
	repos    *model.Repositories
	people   *model.People
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

	s.people, err = storage.LoadPeople()
	if err != nil {
		return err
	}

	return nil
}

func (s *server) run() error {
	r := gin.Default()

	s.initFiles(r)
	s.initProjects(r)

	return r.Run(fmt.Sprintf(":%v", s.opts.Port))
}
