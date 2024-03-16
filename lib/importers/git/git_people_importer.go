package git

import (
	"fmt"
	"path/filepath"

	"github.com/go-enry/go-enry/v2/regex"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
)

var coAuthorsRE regex.EnryRegexp

func init() {
	coAuthorsRE = regex.MustCompile(`(?m)^\s*Co-authored-by\s*:\s*([^<]*?)\s*<([^>]*)>\s*$`)
}

type PeopleImporter struct {
	console consoles.Console
	storage storages.Storage
}

type PeopleOptions struct {
	Branch string
}

func NewPeopleImporter(console consoles.Console, storage storages.Storage) *PeopleImporter {
	return &PeopleImporter{
		console: console,
		storage: storage,
	}
}

func (i PeopleImporter) Import(dirs []string, opts *PeopleOptions) error {
	configDB, err := i.storage.LoadConfig()
	if err != nil {
		return err
	}

	peopleDB, err := i.storage.LoadPeople()
	if err != nil {
		return err
	}

	reposDB, err := i.storage.LoadRepositories()
	if err != nil {
		return err
	}

	dirs, err = findRootDirs(dirs)
	if err != nil {
		return err
	}

	i.console.Printf("Importing people...\n")

	_, err = importPeople(configDB, peopleDB, reposDB, dirs, opts.Branch)
	if err != nil {
		return err
	}

	return nil
}

func importPeople(configDB *map[string]string, peopleDB *model.People, reposDB *model.Repositories,
	dirs []string, branch string,
) (*nameEmailGrouper, error) {
	grouper := newNameEmailGrouperFrom(configDB, peopleDB)

	for _, dir := range dirs {
		gitRepo, err := git.PlainOpen(dir)
		if err != nil {
			fmt.Printf("Skipping '%s': %s\n", dir, err)
			continue
		}

		repo := reposDB.GetOrCreate(dir)
		repo.Name = filepath.Base(dir)
		repo.VCS = "git"

		_, gitRevision, err := findBranchHash(repo, gitRepo, branch)
		if err != nil {
			return nil, err
		}

		commitsIter, err := log(gitRepo, gitRevision)
		if err != nil {
			return nil, err
		}

		total := 0
		err = commitsIter.ForEach(func(commit *object.Commit) error { total++; return nil })

		commitsIter, err = log(gitRepo, gitRevision)
		if err != nil {
			return nil, err
		}

		bar := utils.NewProgressBar(total)
		err = commitsIter.ForEach(func(commit *object.Commit) error {
			bar.Describe(filepath.Base(dir) + ": " + commit.Committer.When.Format("2006-01-02 15"))
			_ = bar.Add(1)

			grouper.add(commit.Author.Name, commit.Author.Email)
			grouper.add(commit.Committer.Name, commit.Committer.Email)

			coAuthors := coAuthorsRE.FindAllStringSubmatch(commit.Message, -1)
			for _, ca := range coAuthors {
				grouper.add(ca[1], ca[2])
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

		_ = bar.Add(1)
	}

	grouper.copyToPeopleDB()

	return grouper, nil
}
