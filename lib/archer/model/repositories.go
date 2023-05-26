package model

import (
	"sort"

	"github.com/samber/lo"
)

type Repositories struct {
	byRootDir map[string]*Repository
	byID      map[UUID]*Repository
}

func NewRepositories() *Repositories {
	return &Repositories{
		byRootDir: map[string]*Repository{},
		byID:      map[UUID]*Repository{},
	}
}

func (s *Repositories) Get(rootDir string) *Repository {
	return s.byRootDir[rootDir]
}

func (s *Repositories) GetOrCreate(rootDir string) *Repository {
	return s.GetOrCreateEx(rootDir, nil)
}

func (s *Repositories) GetOrCreateEx(rootDir string, id *UUID) *Repository {
	if len(rootDir) == 0 {
		panic("empty rootDir not supported")
	}

	result, ok := s.byRootDir[rootDir]

	if !ok {
		result = NewRepository(rootDir, id)
		s.byRootDir[rootDir] = result
		s.byID[result.ID] = result
	}

	return result
}

func (s *Repositories) GetByID(id UUID) *Repository {
	return s.byID[id]
}

func (s *Repositories) List() []*Repository {
	result := lo.Values(s.byRootDir)

	sortRepositories(result)

	return result
}

func sortRepositories(result []*Repository) {
	sort.Slice(result, func(i, j int) bool {
		return result[i].RootDir < result[j].RootDir
	})
}
