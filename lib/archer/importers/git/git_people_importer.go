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

	fmt.Printf("Importing people...\n")

	_, err = importPeople(peopleDB, g.rootDirs)
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

	err = storage.WritePeople(peopleDB)
	if err != nil {
		return err
	}

	return nil
}

func importPeople(peopleDB *model.People, rootDirs []string) (*nameEmailGrouper, error) {
	grouper := newNameEmailGrouperFrom(peopleDB)

	for _, rootDir := range rootDirs {
		rootDir, err := filepath.Abs(rootDir)
		if err != nil {
			return nil, err
		}

		gr, err := git.PlainOpen(rootDir)
		if err != nil {
			fmt.Printf("Skipping '%s': %s\n", rootDir, err)
			continue
		}

		commitsIter, err := gr.Log(&git.LogOptions{})
		if err != nil {
			return nil, err
		}

		err = commitsIter.ForEach(func(gc *object.Commit) error {
			grouper.add(gc.Author.Name, gc.Author.Email, nil)
			grouper.add(gc.Committer.Name, gc.Committer.Email, nil)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	grouper.prepare()

	for _, ne := range grouper.list() {
		var person *model.Person
		if ne.people.Empty() {
			person = peopleDB.GetOrCreatePerson(ne.Name)

		} else {
			people := lo.Filter(ne.people.Slice(), func(p *model.Person, _ int) bool { return p.Name == ne.Name })
			if len(people) > 0 {
				person = people[0]
			} else {
				person = peopleDB.GetPerson(ne.Name)
				if person == nil {
					person = ne.people.Slice()[0]
					peopleDB.ChangePersonName(person, ne.Name)
				}
			}
		}

		for _, n := range ne.Names.Slice() {
			person.AddName(n)
		}
		for _, e := range ne.Emails.Slice() {
			person.AddEmail(e)
		}
	}

	return grouper, nil
}
