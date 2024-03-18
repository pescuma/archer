package blame

import (
	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
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

func (i *Computer) Compute() error {
	filesDB, err := i.storage.LoadFiles()
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

	statsDB, err := i.storage.LoadMonthlyStats()
	if err != nil {
		return err
	}

	i.console.Printf("Computing blame per author and monthly stats...\n")

	blames, err := i.storage.QueryBlamePerAuthor()
	if err != nil {
		return err
	}

	commits := make(map[model.UUID]*model.RepositoryCommit)
	for _, r := range reposDB.List() {
		for _, c := range r.ListCommits() {
			commits[c.ID] = c
			c.Blame.Clear()
		}
	}

	for _, p := range peopleDB.ListPeople() {
		p.Blame.Clear()
	}

	for _, s := range statsDB.ListLines() {
		s.Blame.Clear()
	}

	add := func(blame *model.Blame, query *storages.BlamePerAuthor) {
		switch query.LineType {
		case model.CodeFileLine:
			blame.Code += query.Lines
		case model.CommentFileLine:
			blame.Comment += query.Lines
		case model.BlankFileLine:
			blame.Blank += query.Lines
		default:
			panic(query.LineType)
		}
	}

	for _, blame := range blames {
		c := commits[blame.CommitID]
		add(c.Blame, blame)

		if c.Ignore {
			continue
		}

		pa := peopleDB.GetPersonByID(blame.AuthorID)
		add(pa.Blame, blame)

		file := filesDB.GetFileByID(blame.FileID)

		s := statsDB.GetOrCreateLines(c.Date.Format("2006-01"), blame.RepositoryID,
			blame.AuthorID, blame.CommitterID, file.ProjectID)
		add(s.Blame, blame)
	}

	return nil
}
