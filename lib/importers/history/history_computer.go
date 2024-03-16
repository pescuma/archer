package history

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

	reposDB, err := c.storage.LoadRepositories()
	if err != nil {
		return err
	}

	statsDB, err := c.storage.LoadMonthlyStats()
	if err != nil {
		return err
	}

	c.console.Printf("Computing history for projects, dirs, files, people, areas and monthly stats...\n")

	dirsByIDs := map[model.UUID]*model.ProjectDirectory{}
	for _, p := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		for _, d := range p.Dirs {
			dirsByIDs[d.ID] = d
		}
	}

	for _, p := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		if p.RepositoryID == nil {
			p.Changes.Reset()
		} else {
			p.Changes.Clear()
		}

		for _, d := range p.Dirs {
			if p.RepositoryID == nil {
				d.Changes.Reset()
			} else {
				d.Changes.Clear()
			}
		}
	}
	for _, f := range filesDB.ListFiles() {
		if f.RepositoryID == nil {
			f.Changes.Reset()
		} else {
			f.Changes.Clear()
		}
	}

	for _, p := range peopleDB.ListPeople() {
		p.Changes.Clear()
	}
	for _, a := range peopleDB.ListProductAreas() {
		a.Changes.Clear()
	}

	for _, s := range statsDB.ListLines() {
		s.Changes.Clear()
	}

	now := time.Now()

	for _, repo := range reposDB.List() {
		files := make(map[*model.File]bool)

		for _, commit := range repo.ListCommits() {
			if commit.Ignore {
				continue
			}

			inLast6Months := now.Sub(commit.Date) < 6*30*24*time.Hour
			addChanges := func(c *model.Changes) {
				c.In6Months += utils.IIf(inLast6Months, 1, 0)
				c.Total++
			}

			for _, a := range commit.AuthorIDs {
				author := peopleDB.GetPersonByID(a)
				addChanges(author.Changes)
			}

			commitFiles, err := c.storage.LoadRepositoryCommitFiles(repo, commit)
			if err != nil {
				return err
			}

			projs := make(map[*model.Project]bool)
			dirs := make(map[*model.ProjectDirectory]bool)
			areas := make(map[*model.ProductArea]bool)
			msls := make(map[*model.MonthlyStatsLine]bool)
			for _, cf := range commitFiles.List() {
				addLinesFactor := func(c *model.Changes, factor int) {
					if cf.LinesModified != -1 {
						c.LinesModified += cf.LinesModified / factor
						c.LinesAdded += cf.LinesAdded / factor
						c.LinesDeleted += cf.LinesDeleted / factor
					}
				}
				addLines := func(c *model.Changes) {
					addLinesFactor(c, 1)
				}

				file := filesDB.GetFileByID(cf.FileID)
				files[file] = true

				addChanges(file.Changes)
				addLines(file.Changes)

				if file.ProjectID != nil {
					p := projectsDB.GetByID(*file.ProjectID)
					addLines(p.Changes)
					projs[p] = true
				}
				if file.ProjectDirectoryID != nil {
					d := dirsByIDs[*file.ProjectDirectoryID]
					addLines(d.Changes)
					dirs[d] = true
				}

				if file.ProductAreaID != nil {
					a := peopleDB.GetProductAreaByID(*file.ProductAreaID)
					addLines(a.Changes)
					areas[a] = true
				}

				for _, a := range commit.AuthorIDs {
					author := peopleDB.GetPersonByID(a)
					addLinesFactor(author.Changes, len(commit.AuthorIDs))

					s := statsDB.GetOrCreateLines(commit.Date.Format("2006-01"), repo.ID, author.ID, commit.CommitterID, file.ProjectID)
					if s.Changes.IsEmpty() {
						s.Changes.Clear()
					}
					addLinesFactor(s.Changes, len(commit.AuthorIDs))
					msls[s] = true
				}
			}

			for p := range projs {
				addChanges(p.Changes)
			}
			for d := range dirs {
				addChanges(d.Changes)
			}

			for a := range areas {
				addChanges(a.Changes)
			}

			for s := range msls {
				addChanges(s.Changes)
			}
		}

		repo.FilesTotal = len(files)
	}

	return nil
}
