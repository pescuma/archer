package git

import (
	"fmt"
	"github.com/go-enry/go-enry/v2/regex"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
	"path/filepath"
)

var coAuthorsRE regex.EnryRegexp

func init() {
	coAuthorsRE = regex.MustCompile(`(?m)^\s*Co-authored-by\s*:\s*([^<]*?)\s*<([^>]*)>\s*$`)
}

type PeopleImporter struct {
	console consoles.Console
	storage storages.Storage
}

func NewPeopleImporter(console consoles.Console, storage storages.Storage) *PeopleImporter {
	return &PeopleImporter{
		console: console,
		storage: storage,
	}
}

func (g PeopleImporter) Import(dirs []string) error {
	fmt.Printf("Loading existing data...\n")

	configDB, err := g.storage.LoadConfig()
	if err != nil {
		return err
	}

	peopleDB, err := g.storage.LoadPeople()
	if err != nil {
		return err
	}

	dirs, err = findRootDirs(dirs)
	if err != nil {
		return err
	}

	fmt.Printf("Importing people...\n")

	_, err = importPeople(configDB, peopleDB, dirs)
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

	err = g.storage.WritePeople()
	if err != nil {
		return err
	}

	return nil
}

func importPeople(configDB *map[string]string, peopleDB *model.People, dirs []string) (*nameEmailGrouper, error) {
	grouper := newNameEmailGrouperFrom(configDB, peopleDB)

	for _, dir := range dirs {
		gr, err := git.PlainOpen(dir)
		if err != nil {
			fmt.Printf("Skipping '%s': %s\n", dir, err)
			continue
		}

		commitsIter, err := gr.Log(&git.LogOptions{})
		if err != nil {
			return nil, err
		}

		total := 0
		err = commitsIter.ForEach(func(commit *object.Commit) error { total++; return nil })

		commitsIter, err = gr.Log(&git.LogOptions{})
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
