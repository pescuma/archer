package model

type Repositories struct {
	all map[string]*Repository
}

func NewRepositories() *Repositories {
	return &Repositories{
		all: map[string]*Repository{},
	}
}

func (rs *Repositories) Get(vcs string, rootDir string) *Repository {
	if len(vcs) == 0 {
		panic("empty vcs not supported")
	}
	if len(rootDir) == 0 {
		panic("empty rootDir not supported")
	}

	key := vcs + "\n" + rootDir
	result, ok := rs.all[key]

	if !ok {
		result = NewRepository(vcs, rootDir)
		rs.all[key] = result
	}

	return result
}
