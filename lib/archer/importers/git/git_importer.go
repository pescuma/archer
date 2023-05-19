package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type gitImporter struct {
	rootDir string
}

func NewImporter(rootDir string) archer.Importer {
	return &gitImporter{
		rootDir: rootDir,
	}
}

func (g gitImporter) Import(storage archer.Storage) error {
	rootDir, err := filepath.Abs(g.rootDir)
	if err != nil {
		return err
	}

	fmt.Printf("Opening git repo...\n")

	gr, err := git.PlainOpen(rootDir)
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

	repo, err := storage.LoadRepository(rootDir)
	if err != nil {
		return err
	}

	if repo == nil {
		repo = model.NewRepository(rootDir)
	}

	repo.VCS = "git"

	fmt.Printf("Grouping authors...\n")

	commitsIter, err := gr.Log(&git.LogOptions{})
	if err != nil {
		return err
	}

	grouper := newNameEmailGrouper()

	total := 0
	err = commitsIter.ForEach(func(gc *object.Commit) error {
		grouper.add(gc.Author.Name, gc.Author.Email)
		grouper.add(gc.Committer.Name, gc.Committer.Email)
		total++
		return nil
	})

	grouper.prepare()

	fmt.Printf("Loading history...\n")

	commitsIter, err = gr.Log(&git.LogOptions{})
	if err != nil {
		return err
	}

	bar := utils.NewProgressBar(total)
	err = commitsIter.ForEach(func(gc *object.Commit) error {
		_ = bar.Add(1)
		bar.Describe(gc.Committer.When.Format("2006-01-02 15:04"))

		if repo.ContainsCommit(gc.Hash.String()) {
			return nil
		}

		author := people.Get(grouper.getName(gc.Author.Email))
		author.AddEmail(gc.Author.Email)

		committer := people.Get(grouper.getName(gc.Committer.Email))
		committer.AddEmail(gc.Committer.Email)

		commit := repo.GetCommit(gc.Hash.String())
		commit.Message = strings.TrimSpace(gc.Message)
		commit.Date = gc.Committer.When
		commit.CommitterID = committer.ID
		commit.DateAuthored = gc.Author.When
		commit.AuthorID = author.ID

		err := gc.Parents().ForEach(func(p *object.Commit) error {
			commit.Parents = append(commit.Parents, p.Hash.String())
			return nil
		})
		if err != nil {
			return err
		}

		gs, err := gc.Stats()
		if err != nil {
			return err
		}

		for _, gf := range gs {
			if gf.Name != "" {
				path := filepath.Join(rootDir, gf.Name)

				file := files.Get(path)
				file.RepositoryID = &repo.ID

				commit.AddFile(file.ID, gf.Addition, gf.Deletion)
			}

			commit.AddedLines += gf.Addition
			commit.DeletedLines += gf.Deletion
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

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
