package git

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
)

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

	peopleDB, err := g.storage.LoadPeople()
	if err != nil {
		return err
	}

	dirs, err = findRootDirs(dirs)
	if err != nil {
		return err
	}

	fmt.Printf("Importing people...\n")

	_, err = importPeople(peopleDB, dirs)
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

func importPeople(peopleDB *model.People, dirs []string) (*nameEmailGrouper, error) {
	grouper := newNameEmailGrouperFrom(peopleDB)

	bar := utils.NewProgressBar(len(dirs))
	for _, dir := range dirs {
		bar.Describe(utils.TruncateFilename(dir))

		gr, err := git.PlainOpen(dir)
		if err != nil {
			fmt.Printf("Skipping '%s': %s\n", dir, err)
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

		_ = bar.Add(1)
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
