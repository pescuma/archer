package git

import (
	"fmt"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

type gitPeopleImporter struct {
	rootDirs []string
}

func NewPeopleImporter(rootDirs []string) archer.Importer {
	return &gitPeopleImporter{
		rootDirs: rootDirs,
	}
}

func (g gitPeopleImporter) Import(storage archer.Storage) error {
	fmt.Printf("Loading existing data...\n")

	peopleDB, err := storage.LoadPeople()
	if err != nil {
		return err
	}

	fmt.Printf("Importing and grouping authors...\n")

	grouper := newNameEmailGrouperFrom(peopleDB)

	for _, rootDir := range g.rootDirs {
		rootDir, err = filepath.Abs(rootDir)
		if err != nil {
			return err
		}

		gr, err := git.PlainOpen(rootDir)
		if err != nil {
			fmt.Printf("Skipping '%s': %s\n", rootDir, err)
			continue
		}

		commitsIter, err := gr.Log(&git.LogOptions{})
		if err != nil {
			return err
		}

		err = commitsIter.ForEach(func(gc *object.Commit) error {
			grouper.add(gc.Author.Name, gc.Author.Email, nil)
			grouper.add(gc.Committer.Name, gc.Committer.Email, nil)
			return nil
		})
		if err != nil {
			return err
		}
	}

	grouper.prepare()

	for _, ne := range grouper.list() {
		var person *model.Person
		if len(ne.people) == 0 {
			person = peopleDB.GetOrCreatePerson(ne.Name)

		} else {
			people := lo.Filter(ne.people, func(p *model.Person, _ int) bool { return p.Name == ne.Name })
			if len(people) > 0 {
				person = people[0]
			} else {
				person = peopleDB.GetPerson(ne.Name)
				if person == nil {
					person = ne.people[0]
					peopleDB.ChangePersonName(person, ne.Name)
				}
			}
		}

		for n := range ne.Names {
			person.AddName(n)
		}
		for e := range ne.Emails {
			person.AddEmail(e)
		}
	}

	fmt.Printf("Writing results...\n")

	err = storage.WritePeople(peopleDB, archer.ChangedBasicInfo)
	if err != nil {
		return err
	}

	return nil
}
