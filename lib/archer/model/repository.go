package model

type Repository struct {
	VCS     string
	RootDir string
	ID      UUID

	Data map[string]string

	Commits map[string]*RepositoryCommit
}

func NewRepository(vcs, rootDir string) *Repository {
	return &Repository{
		VCS:     vcs,
		RootDir: rootDir,
		ID:      NewUUID("r"),
		Data:    map[string]string{},
	}
}

func (r Repository) GetCommit(hash string) *RepositoryCommit {
	result, ok := r.Commits[hash]

	if !ok {
		result = NewRepositoryCommit(hash)
		r.Commits[hash] = result
	}

	return result
}
