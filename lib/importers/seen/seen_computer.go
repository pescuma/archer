package seen

import (
	"time"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
)

type Computer struct {
	console consoles.Console
	storage storages.Storage
}

func NewComputer(console consoles.Console, storage storages.Storage) *Computer {
	return &Computer{
		console: console,
		storage: storage,
	}
}

func (c *Computer) Compute() error {
	projectsDB, err := c.storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := c.storage.LoadFiles()
	if err != nil {
		return err
	}

	peopleDB, err := c.storage.LoadPeople()
	if err != nil {
		return err
	}

	peopleRelationsDB, err := c.storage.LoadPeopleRelations()
	if err != nil {
		return err
	}

	reposDB, err := c.storage.LoadRepositories()
	if err != nil {
		return err
	}

	c.console.Printf("Computing seen at information ...\n")

	for _, repo := range reposDB.List() {
		repo.SeenAt.Clear()
	}

	for _, pf := range peopleRelationsDB.ListFiles() {
		pf.SeenAt.Clear()
	}
	for _, pr := range peopleRelationsDB.ListRepositories() {
		pr.SeenAt.Clear()
	}

	for _, p := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		p.SeenAt.Clear()
		for _, d := range p.Dirs {
			d.SeenAt.Clear()
		}
	}

	for _, p := range peopleDB.ListPeople() {
		p.SeenAt.Clear()
	}
	for _, pa := range peopleDB.ListProductAreas() {
		pa.SeenAt.Clear()
	}

	now := time.Now()
	for _, file := range filesDB.List() {
		exists, err := utils.FileExists(file.Path)
		if err != nil {
			return err
		}

		file.Exists = exists
		if exists {
			file.SeenAt.Add(now)
		}
	}

	for _, repo := range reposDB.List() {
		for _, commit := range repo.ListCommits() {
			if commit.Ignore {
				continue
			}

			dates := []time.Time{commit.Date, commit.DateAuthored}

			processPerson := func(personID model.ID) {
				person := peopleDB.GetPersonByID(personID)

				person.SeenAt.Add(dates...)

				peopleRelationsDB.GetOrCreatePersonRepo(personID, repo.ID).
					SeenAt.Add(dates...)
			}

			processFile := func(fileID model.ID) {
				file := filesDB.GetByID(fileID)

				file.SeenAt.Add(dates...)

				peopleRelationsDB.GetOrCreatePersonFile(commit.CommitterID, file.ID).
					SeenAt.Add(dates...)
				for _, a := range commit.AuthorIDs {
					peopleRelationsDB.GetOrCreatePersonFile(a, file.ID).
						SeenAt.Add(dates...)
				}
			}

			repo.SeenAt.Add(dates...)

			processPerson(commit.CommitterID)
			for _, a := range commit.AuthorIDs {
				processPerson(a)
			}

			details, err := c.storage.LoadRepositoryCommitDetails(repo, commit)
			if err != nil {
				return err
			}

			for _, cf := range commit.Files {
				processFile(cf.FileID)

				cfd := details.GetOrCreateFile(cf.FileID)
				for _, oldFileID := range cfd.OldIDs {
					processFile(oldFileID)
				}
			}
		}
	}

	filesByDir := filesDB.GroupByDirectory()

	for _, proj := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		for _, dir := range proj.Dirs {
			for _, file := range filesByDir[dir.ID] {
				if file.Ignore {
					continue
				}

				proj.SeenAt.Merge(file.SeenAt)
				dir.SeenAt.Merge(file.SeenAt)

				if file.ProductAreaID != nil {
					a := peopleDB.GetProductAreaByID(*file.ProductAreaID)
					a.SeenAt.Merge(file.SeenAt)
				}
			}
		}
	}

	return nil
}
