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
	"github.com/samber/lo"
	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
)

type gitHistoryImporter struct {
	rootDirs []string
	options  HistoryOptions
}

type HistoryOptions struct {
	Incremental        bool
	MaxImportedCommits *int
	MaxCommits         *int
	After              *time.Time
	Before             *time.Time
	SaveEvery          *int
}

func NewHistoryImporter(rootDirs []string, options HistoryOptions) archer.Importer {
	return &gitHistoryImporter{
		rootDirs: rootDirs,
		options:  options,
	}
}

type work struct {
	rootDir string
	repo    *model.Repository
}

func (g gitHistoryImporter) Import(storage archer.Storage) error {
	fmt.Printf("Loading existing data...\n")

	projectsDB, err := storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := storage.LoadFiles()
	if err != nil {
		return err
	}

	peopleDB, err := storage.LoadPeople()
	if err != nil {
		return err
	}

	reposDB, err := storage.LoadRepositories()
	if err != nil {
		return err
	}

	abort := errors.New("ABORT")

	fmt.Printf("Preparing...\n")

	grouper := newNameEmailGrouperFrom(peopleDB)

	commitNumber := 0
	imported := 0
	ws := make([]*work, 0, len(g.rootDirs))
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

		repo := reposDB.GetOrCreate(rootDir)
		repo.Name = filepath.Base(rootDir)
		repo.VCS = "git"

		importedFromRepo := 0

		err = commitsIter.ForEach(func(gc *object.Commit) error {
			if !g.options.ShouldContinue(commitNumber, imported, gc.Committer.When) {
				return abort
			}

			commitNumber++

			if g.options.Incremental && repo.ContainsCommit(gc.Hash.String()) {
				return nil
			}

			imported++
			importedFromRepo++
			return nil
		})
		if err != nil && err != abort {
			return err
		}

		if importedFromRepo > 0 {
			ws = append(ws, &work{
				rootDir: rootDir,
				repo:    repo,
			})
		}
	}

	fmt.Printf("Loading history...\n")

	bar := utils.NewProgressBar(imported)

	write := func(repo *model.Repository) error {
		_ = bar.Clear()
		fmt.Printf("Writing results...")

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

	commitNumber = 0
	imported = 0
	for i, w := range ws {
		if w == nil {
			continue
		}

		gr, err := git.PlainOpen(w.rootDir)
		if err != nil {
			return err
		}

		commitsIter, err := gr.Log(&git.LogOptions{})
		if err != nil {
			return err
		}

		touchedFiles := map[model.UUID]*model.File{}
		err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
			if !g.options.ShouldContinue(commitNumber, imported, gitCommit.Committer.When) {
				return abort
			}

			commitNumber++

			if g.options.Incremental && w.repo.ContainsCommit(gitCommit.Hash.String()) {
				return nil
			}

			imported++

			bar.Describe(w.repo.Name + " " + gitCommit.Committer.When.Format("2006-01-02 15"))
			_ = bar.Add(1)

			author := peopleDB.GetOrCreatePerson(grouper.getName(gitCommit.Author.Email))
			author.AddName(gitCommit.Author.Name)
			author.AddEmail(gitCommit.Author.Email)

			committer := peopleDB.GetOrCreatePerson(grouper.getName(gitCommit.Committer.Email))
			committer.AddName(gitCommit.Committer.Name)
			committer.AddEmail(gitCommit.Committer.Email)

			commit := w.repo.GetCommit(gitCommit.Hash.String())
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
				filePath := filepath.Join(w.rootDir, gitFile.Name)
				oldFilePath := filepath.Join(w.rootDir, gitFile.OldName)

				file := filesDB.GetOrCreateFile(filePath)
				file.RepositoryID = &w.repo.ID

				oldFile := filesDB.GetOrCreateFile(oldFilePath)
				oldFile.RepositoryID = &w.repo.ID

				commit.AddFile(file.ID, utils.IIf(file != oldFile, &oldFile.ID, nil), gitFile.Modified, gitFile.Added, gitFile.Deleted)

				commit.ModifiedLines += gitFile.Modified
				commit.AddedLines += gitFile.Added
				commit.DeletedLines += gitFile.Deleted

				touchedFiles[file.ID] = file
			}

			if g.options.SaveEvery != nil && imported%*g.options.SaveEvery == 0 {
				err = write(w.repo)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil && err != abort {
			return err
		}

		err = write(w.repo)
		if err != nil {
			return err
		}

		// Free memory
		ws[i] = nil
	}
	_ = bar.Clear()

	fmt.Printf("Propagating changes to parents...\n")

	propagateChangesToParents(reposDB, projectsDB, filesDB, peopleDB)

	fmt.Printf("Writing results...\n")

	err = storage.WriteProjects(projectsDB, archer.ChangedChanges)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB, archer.ChangedChanges)
	if err != nil {
		return err
	}

	err = storage.WritePeople(peopleDB, archer.ChangedBasicInfo|archer.ChangedChanges)
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
				diffs := diff.DoWithTimeout(parentContent, commitContent, 30*time.Second)
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

func propagateChangesToParents(reposDB *model.Repositories, projectsDB *model.Projects, filesDB *model.Files, peopleDB *model.People) {
	dirsByIDs := map[model.UUID]*model.ProjectDirectory{}
	for _, p := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		for _, d := range p.Dirs {
			dirsByIDs[d.ID] = d
		}
	}

	for _, p := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		p.Changes.Clear()
		for _, d := range p.Dirs {
			d.Changes.Clear()
		}
	}
	for _, f := range filesDB.ListFiles() {
		f.Changes.Clear()
	}

	for _, p := range peopleDB.ListPeople() {
		p.Changes.Clear()
	}
	for _, o := range peopleDB.ListOrganizations() {
		o.Changes.Clear()
	}
	gs := peopleDB.ListGroupsByID()
	for _, g := range gs {
		g.Changes.Clear()
	}
	ts := peopleDB.ListTeamsByID()
	for _, t := range ts {
		t.Changes.Clear()
	}
	for _, a := range peopleDB.ListProductAreas() {
		a.Changes.Clear()
	}

	now := time.Now()

	for _, repo := range reposDB.List() {
		for _, c := range repo.ListCommits() {
			inLast6Months := now.Sub(c.Date) < 6*30*24*time.Hour
			addChanges := func(c *model.Changes) {
				c.In6Months += utils.IIf(inLast6Months, 1, 0)
				c.Total++
			}

			author := peopleDB.GetPersonByID(c.AuthorID)
			addChanges(author.Changes)

			projs := map[*model.Project]bool{}
			dirs := map[*model.ProjectDirectory]bool{}
			areas := map[*model.ProductArea]bool{}
			orgs := map[*model.Organization]bool{}
			groups := map[*model.Group]bool{}
			teams := map[*model.Team]bool{}
			for _, cf := range c.Files {
				addLines := func(c *model.Changes) {
					c.ModifiedLines += cf.ModifiedLines
					c.AddedLines += cf.AddedLines
					c.DeletedLines += cf.DeletedLines
				}

				file := filesDB.GetFileByID(cf.FileID)
				addChanges(file.Changes)
				addLines(file.Changes)

				addLines(author.Changes)

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

				// TODO Make this dependent on time
				if file.OrganizationID != nil {
					o := peopleDB.GetOrganizationByID(*file.OrganizationID)
					addLines(o.Changes)
					orgs[o] = true
				}
				if file.GroupID != nil {
					g := gs[*file.GroupID]
					addLines(g.Changes)
					groups[g] = true
				}
				if file.TeamID != nil {
					t := ts[*file.TeamID]
					addLines(t.Changes)
					teams[t] = true
				}
			}

			for _, p := range lo.Keys(projs) {
				addChanges(p.Changes)
			}
			for _, d := range lo.Keys(dirs) {
				addChanges(d.Changes)
			}

			for _, a := range lo.Keys(areas) {
				addChanges(a.Changes)
			}

			for _, o := range lo.Keys(orgs) {
				addChanges(o.Changes)
			}
			for _, g := range lo.Keys(groups) {
				addChanges(g.Changes)
			}
			for _, t := range lo.Keys(teams) {
				addChanges(t.Changes)
			}
		}
	}
}

type gitFileChange struct {
	Name     string
	OldName  string
	Modified int
	Added    int
	Deleted  int
}

func (l *HistoryOptions) ShouldContinue(commit int, imported int, date time.Time) bool {
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