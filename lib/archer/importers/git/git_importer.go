package git

import (
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/Faire/archer/lib/archer"
)

type gitImporter struct {
	path string
}

func NewImporter(path string) archer.Importer {
	return &gitImporter{
		path: path,
	}
}

func (g gitImporter) Import(storage archer.Storage) error {
	repositories, err := storage.LoadRepositories()
	if err != nil {
		return err
	}

	files, err := storage.LoadFiles()
	if err != nil {
		return err
	}

	people, err := storage.LoadPeople()
	if err != nil {
		return err
	}

	path, err := filepath.Abs(g.path)
	if err != nil {
		return err
	}

	gr, err := git.PlainOpen(path)
	if err != nil {
		return err
	}

	repo := repositories.Get("git", path)

	commits, err := gr.Log(&git.LogOptions{})
	if err != nil {
		return err
	}

	err = commits.ForEach(func(gc *object.Commit) error {
		author := people.Get(gc.Author.Name)
		author.AddEmail(gc.Author.Email)

		committer := people.Get(gc.Committer.Name)
		committer.AddEmail(gc.Committer.Email)

		commit := repo.GetCommit(gc.Hash.String())
		commit.Date = gc.Committer.When
		commit.DateAuthored = gc.Author.When
		commit.AuthorID = author.ID
		commit.CommitterID = committer.ID

		gs, err := gc.Stats()
		if err != nil {
			return err
		}

		for _, gf := range gs {
			file := files.Get(gf.Name)
			file.RepositoryID = &repo.ID

			commit.AddFile(file.ID, gf.Addition, gf.Deletion)
			commit.AddedLines += gf.Addition
			commit.DeletedLines += gf.Deletion
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = storage.WritePeople(people, archer.ChangedBasicInfo)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(files, archer.ChangedBasicInfo)
	if err != nil {
		return err
	}

	err = storage.WriteRepository(repo, archer.ChangedBasicInfo|archer.ChangedHistory)
	if err != nil {
		return err
	}

	return nil
}
