package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/diff"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type gitImporter struct {
	rootDir string
	options Options
}

type Options struct {
	Incremental        bool
	MaxImportedCommits *int
	MaxCommits         *int
	After              *time.Time
	Before             *time.Time
	SaveEvery          *int
}

func NewImporter(rootDir string, options Options) archer.Importer {
	return &gitImporter{
		rootDir: rootDir,
		options: options,
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

	filesDB, err := storage.LoadFiles()
	if err != nil {
		return err
	}

	peopleDB, err := storage.LoadPeople()
	if err != nil {
		return err
	}

	repo, err := storage.LoadRepository(rootDir)
	if err != nil {
		return err
	}

	if repo == nil {
		repo = model.NewRepository(rootDir, nil)
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

	for _, p := range peopleDB.List() {
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
		if !g.options.ShouldContinue(total, imported, gc.Committer.When) {
			return abort
		}

		total++

		if g.options.Incremental && repo.ContainsCommit(gc.Hash.String()) {
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

	for _, ne := range grouper.list() {
		person := peopleDB.GetOrCreate(ne.Name)

		for n := range ne.Names {
			person.AddName(n)
		}
		for e := range ne.Emails {
			person.AddEmail(e)
		}
	}

	fmt.Printf("Loading history...\n")

	commitsIter, err = gr.Log(&git.LogOptions{})
	if err != nil {
		return err
	}

	bar := utils.NewProgressBar(imported)
	commitNumber = 0
	imported = 0
	touchedFiles := map[model.UUID]*model.File{}
	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		if !g.options.ShouldContinue(commitNumber, imported, gitCommit.Committer.When) {
			return abort
		}

		commitNumber++

		if g.options.Incremental && repo.ContainsCommit(gitCommit.Hash.String()) {
			return nil
		}

		imported++

		bar.Describe(gitCommit.Committer.When.Format("2006-01-02 15:04"))
		_ = bar.Add(1)

		author := peopleDB.GetOrCreate(grouper.getName(gitCommit.Author.Email))
		author.AddName(gitCommit.Author.Name)
		author.AddEmail(gitCommit.Author.Email)

		committer := peopleDB.GetOrCreate(grouper.getName(gitCommit.Committer.Email))
		committer.AddName(gitCommit.Committer.Name)
		committer.AddEmail(gitCommit.Committer.Email)

		commit := repo.GetCommit(gitCommit.Hash.String())
		commit.Message = strings.TrimSpace(gitCommit.Message)
		commit.Date = gitCommit.Committer.When
		commit.CommitterID = committer.ID
		commit.DateAuthored = gitCommit.Author.When
		commit.AuthorID = author.ID

		var firstParent *object.Commit
		err := gitCommit.Parents().ForEach(func(p *object.Commit) error {
			if firstParent == nil {
				firstParent = p
			}
			commit.Parents = append(commit.Parents, p.Hash.String())
			return nil
		})
		if err != nil {
			return err
		}

		gitChanges, err := computeChanges(gitCommit, firstParent)
		if err != nil {
			return err
		}

		commit.ModifiedLines = 0
		commit.AddedLines = 0
		commit.DeletedLines = 0
		commit.Files = nil

		for _, gitFile := range gitChanges {
			filePath := filepath.Join(rootDir, gitFile.Name)
			oldFilePath := filepath.Join(rootDir, gitFile.OldName)

			file := filesDB.GetOrCreate(filePath)
			file.RepositoryID = &repo.ID

			oldFile := filesDB.GetOrCreate(oldFilePath)
			oldFile.RepositoryID = &repo.ID

			commit.AddFile(file.ID, utils.IIf(file != oldFile, &oldFile.ID, nil), gitFile.Modified, gitFile.Added, gitFile.Deleted)

			commit.ModifiedLines += gitFile.Modified
			commit.AddedLines += gitFile.Added
			commit.DeletedLines += gitFile.Deleted

			touchedFiles[file.ID] = file
		}

		if g.options.SaveEvery != nil && imported%*g.options.SaveEvery == 0 {
			_ = bar.Clear()
			fmt.Print("Writing results...")

			err := storage.WritePeople(peopleDB, archer.ChangedBasicInfo)
			if err != nil {
				return err
			}

			err = storage.WriteFiles(filesDB, archer.ChangedBasicInfo)
			if err != nil {
				return err
			}

			err = storage.WriteRepository(repo, archer.ChangedBasicInfo|archer.ChangedHistory)
			if err != nil {
				return err
			}

			fmt.Print("\r")
			_ = bar.RenderBlank()
		}

		return nil
	})
	if err != nil && err != abort {
		return err
	}

	if imported == 0 {
		fmt.Printf("No new commits to import.\n")
	}

	fmt.Printf("Writing results...\n")

	err = storage.WritePeople(peopleDB, archer.ChangedBasicInfo)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB, archer.ChangedBasicInfo)
	if err != nil {
		return err
	}

	err = storage.WriteRepository(repo, archer.ChangedBasicInfo|archer.ChangedHistory)
	if err != nil {
		return err
	}

	return nil
}

func computeChanges(commit *object.Commit, parent *object.Commit) ([]*gitFileChange, error) {
	commitTree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var parentTree *object.Tree
	if parent != nil {
		parentTree, err = parent.Tree()
		if err != nil {
			return nil, err
		}
	}

	changes, err := commitTree.DiffContext(context.Background(), parentTree)
	if err != nil {
		return nil, err
	}

	var result []*gitFileChange
	for _, change := range changes {
		commitFile, parentFile, err := change.Files()
		if err != nil {
			return nil, err
		}

		commitContent, commitIsBinary, err := fileContent(commitFile)
		if err != nil {
			return nil, err
		}

		parentContent, parentIsBinary, err := fileContent(parentFile)
		if err != nil {
			return nil, err
		}

		gitChange := gitFileChange{}

		if commitFile != nil {
			gitChange.Name = change.From.Name
		} else {
			gitChange.Name = change.To.Name
		}
		if parentFile != nil {
			gitChange.OldName = change.To.Name
		} else {
			gitChange.OldName = change.From.Name
		}

		if !commitIsBinary && !parentIsBinary {
			commitLines := countLines(commitContent)
			parentLines := countLines(parentContent)

			if parentLines == 0 {
				gitChange.Added += commitLines

			} else if commitLines == 0 {
				gitChange.Deleted += parentLines

			} else if parentLines > 10_000 || commitLines > 10_000 {
				// gotextdiff goes out of memory
				diffs := diff.Do(parentContent, commitContent)
				for _, d := range diffs {
					switch d.Type {
					case diffmatchpatch.DiffDelete:
						gitChange.Deleted += countLines(d.Text)
					case diffmatchpatch.DiffInsert:
						gitChange.Added += countLines(d.Text)
					}
				}

			} else {
				edits := myers.ComputeEdits("parent", parentContent, commitContent)
				unified := gotextdiff.ToUnified("parent", "commit", parentContent, edits)

				// Modified is defined as changes that happened without a line without change in the middle
				for _, hunk := range unified.Hunks {
					add := 0
					del := 0
					for _, line := range hunk.Lines {
						switch line.Kind {
						case gotextdiff.Insert:
							add++
						case gotextdiff.Delete:
							del++
						default:
							min := utils.Min(add, del)
							gitChange.Modified += min
							gitChange.Added += add - min
							gitChange.Deleted += del - min

							add = 0
							del = 0
						}
					}

					min := utils.Min(add, del)
					gitChange.Modified += min
					gitChange.Added += add - min
					gitChange.Deleted += del - min
				}
			}
		}

		result = append(result, &gitChange)
	}

	return result, nil
}

func fileContent(f *object.File) (string, bool, error) {
	if f == nil {
		return "", false, nil
	}

	isBinary, err := f.IsBinary()
	if err != nil {
		return "", false, err
	}

	if isBinary {
		return "", true, nil
	}

	content, err := f.Contents()
	if err != nil {
		return "", false, err
	}

	return content, isBinary, err
}

func countLines(text string) int {
	if text == "" {
		return 0
	}

	result := strings.Count(text, "\n")
	if text[len(text)-1] != '\n' {
		result++
	}
	return result
}

type gitFileChange struct {
	Name     string
	OldName  string
	Modified int
	Added    int
	Deleted  int
}

func (l *Options) ShouldContinue(commit int, imported int, date time.Time) bool {
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
