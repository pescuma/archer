package git

import (
	"path/filepath"

	"github.com/go-git/go-git/v5"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/storages"
)

type ReposImporter struct {
	console consoles.Console
	storage storages.Storage
}

type ReposOptions struct {
	Branch string
}

func NewReposImporter(console consoles.Console, storage storages.Storage) *ReposImporter {
	return &ReposImporter{
		console: console,
		storage: storage,
	}
}

func (i *ReposImporter) Import(dirs []string, opts *ReposOptions) error {
	i.console.Printf("Loading existing data...\n")

	reposDB, err := i.storage.LoadRepositories()
	if err != nil {
		return err
	}

	i.console.Printf("Importing git repositories...\n")

	dirs, err = findRootDirs(dirs)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		gitRepo, err := git.PlainOpen(dir)
		if err != nil {
			i.console.Printf("Skipping '%s': %s\n", dir, err)
			continue
		}

		repo := reposDB.GetOrCreate(dir)
		repo.Name = filepath.Base(dir)
		repo.VCS = "git"

		branch, _, err := findBranchHash(repo, gitRepo, opts.Branch)
		if err != nil {
			return err
		}

		repo.Branch = branch
	}

	i.console.Printf("Writing results...\n")

	err = i.storage.WriteRepositories()
	if err != nil {
		return err
	}

	return nil
}
