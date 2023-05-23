package git

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type gitImporter struct {
	rootDir string
	limits  Limits
}

type Limits struct {
	Incremental        bool
	MaxImportedCommits *int
	MaxCommits         *int
	After              *time.Time
	Before             *time.Time
}

func NewImporter(rootDir string, limits Limits) archer.Importer {
	return &gitImporter{
		rootDir: rootDir,
		limits:  limits,
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

	fmt.Printf("Loading existing data...\n")

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

	commitNumber := 0
	imported := 0
	abort := errors.New("ABORT")

	grouper := newNameEmailGrouper()

	for _, p := range people.List() {
		emails := p.ListEmails()
		if len(emails) == 0 {
			continue
		}

		for _, email := range emails {
			grouper.add(p.Name, email)
		}
		for _, name := range p.ListNames() {
			grouper.add(name, emails[0])
		}
	}

	total := 0
	err = commitsIter.ForEach(func(gc *object.Commit) error {
		if !g.limits.ShouldContinue(total, imported, gc.Committer.When) {
			return abort
		}

		total++

		if g.limits.Incremental && repo.ContainsCommit(gc.Hash.String()) {
			return nil
		}

		imported++

		grouper.add(gc.Author.Name, gc.Author.Email)
		grouper.add(gc.Committer.Name, gc.Committer.Email)
		return nil
	})
	if err != nil && err != abort {
		return err
	}

	grouper.prepare()

	fmt.Printf("Loading history...\n")

	if imported == 0 {
		fmt.Printf("No new commits to import.\n")
		return nil
	}

	commitNumber = 0
	imported = 0

	commitsIter, err = gr.Log(&git.LogOptions{})
	if err != nil {
		return err
	}

	bar := utils.NewProgressBar(total)
	err = commitsIter.ForEach(func(gc *object.Commit) error {
		if !g.limits.ShouldContinue(commitNumber, imported, gc.Committer.When) {
			return abort
		}

		commitNumber++

		bar.Describe(gc.Committer.When.Format("2006-01-02 15:04"))
		_ = bar.Add(1)

		if g.limits.Incremental && repo.ContainsCommit(gc.Hash.String()) {
			return nil
		}

		imported++

		author := people.GetOrCreate(grouper.getName(gc.Author.Email))
		author.AddName(gc.Author.Name)
		author.AddEmail(gc.Author.Email)

		committer := people.GetOrCreate(grouper.getName(gc.Committer.Email))
		committer.AddName(gc.Committer.Name)
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

				file := files.GetOrCreate(path)
				file.RepositoryID = &repo.ID

				commit.AddFile(file.ID, gf.Addition, gf.Deletion)
			}

			commit.AddedLines += gf.Addition
			commit.DeletedLines += gf.Deletion
		}

		return nil
	})
	if err != nil && err != abort {
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

func (l *Limits) ShouldContinue(commit int, imported int, date time.Time) bool {
	if l.After != nil && date.Before(*l.After) {
		return false
	}
	if l.Before != nil && !date.Before(*l.Before) {
		return false
	}

	if l.MaxCommits != nil && commit >= *l.MaxCommits {
		return false
	}

	if l.MaxImportedCommits != nil && imported >= *l.MaxImportedCommits {
		return false
	}

	return true
}
