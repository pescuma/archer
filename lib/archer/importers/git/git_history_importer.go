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

	grouper         *nameEmailGrouper
	commitsTotal    int
	commitsImported int
	abort           error
}

type HistoryOptions struct {
	Incremental        bool
	MaxImportedCommits *int
	MaxCommits         *int
	After              *time.Time
	Before             *time.Time
	SaveEvery          *time.Duration
}

func NewHistoryImporter(rootDirs []string, options HistoryOptions) archer.Importer {
	return &gitHistoryImporter{
		rootDirs: rootDirs,
		options:  options,
		abort:    errors.New("ABORT"),
	}
}

func (g *gitHistoryImporter) Import(storage archer.Storage) error {
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

	peopleRelationsDB, err := storage.LoadPeopleRelations()
	if err != nil {
		return err
	}

	statsDB, err := storage.LoadMonthlyStats()
	if err != nil {
		return err
	}

	fmt.Printf("Importing and grouping authors...\n")

	g.grouper, err = importPeople(peopleDB, g.rootDirs)
	if err != nil {
		return err
	}

	if g.options.SaveEvery != nil {
		err = storage.WritePeople(peopleDB)
		if err != nil {
			return err
		}
	}

	for _, rootDir := range g.rootDirs {
		rootDir, err := filepath.Abs(rootDir)
		if err != nil {
			return err
		}

		gitRepo, err := git.PlainOpen(rootDir)
		if err != nil {
			fmt.Printf("Skipping '%s': %s\n", rootDir, err)
			continue
		}

		repo := reposDB.GetOrCreate(rootDir)
		repo.Name = filepath.Base(rootDir)
		repo.VCS = "git"

		commitsImported, err := g.importCommits(peopleDB, peopleRelationsDB, repo, gitRepo)
		if err != nil {
			return err
		}

		repo.FilesHead, err = g.countFilesAtHEAD(gitRepo)
		if err != nil {
			return err
		}

		if g.options.SaveEvery != nil && commitsImported > 0 {
			fmt.Printf("%v: Writing results...\n", repo.Name)

			err = storage.WritePeople(peopleDB)
			if err != nil {
				return err
			}

			err := storage.WriteRepository(repo)
			if err != nil {
				return err
			}

			err = storage.WritePeopleRelations(peopleRelationsDB)
			if err != nil {
				return err
			}
		}

		err = g.importChanges(storage, filesDB, projectsDB, peopleRelationsDB, repo, gitRepo)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Propagating changes to parents...\n")

	err = g.propagateChangesToParents(storage, reposDB, projectsDB, filesDB, peopleDB, statsDB)
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

	err = storage.WriteProjects(projectsDB)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB)
	if err != nil {
		return err
	}

	err = storage.WritePeople(peopleDB)
	if err != nil {
		return err
	}

	err = storage.WriteRepositories(reposDB)
	if err != nil {
		return err
	}

	err = storage.WritePeopleRelations(peopleRelationsDB)
	if err != nil {
		return err
	}

	err = storage.WriteMonthlyStats(statsDB)
	if err != nil {
		return err
	}

	return nil
}

func (g *gitHistoryImporter) countFilesAtHEAD(gitRepo *git.Repository) (int, error) {
	gitHead, err := gitRepo.Head()
	if err != nil {
		return 0, err
	}

	gitCommit, err := gitRepo.CommitObject(gitHead.Hash())
	if err != nil {
		return 0, err
	}

	gitTree, err := gitCommit.Tree()
	if err != nil {
		return 0, err
	}

	result := 0
	err = gitTree.Files().ForEach(func(gitFile *object.File) error {
		result++
		return nil
	})
	if err != nil {
		return 0, err
	}

	return result, nil
}

func log(gitRepo *git.Repository) (object.CommitIter, error) {
	return gitRepo.Log(&git.LogOptions{
		Order: git.LogOrderCommitterTime,
	})
}

func (g *gitHistoryImporter) countCommitsToImport(repo *model.Repository, gitRepo *git.Repository) (int, error) {
	commitsIter, err := log(gitRepo)
	if err != nil {
		return 0, err
	}

	imported := 0
	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		if g.options.Incremental && repo.ContainsCommit(gitCommit.Hash.String()) {
			return nil
		}

		imported++

		return nil
	})
	if err != nil {
		return 0, err
	}

	return imported, nil
}

func (g *gitHistoryImporter) importCommits(peopleDB *model.People, peopleRelationsDB *model.PeopleRelations,
	repo *model.Repository, gitRepo *git.Repository,
) (int, error) {
	imported, err := g.countCommitsToImport(repo, gitRepo)
	if err != nil {
		return 0, err
	}

	if imported == 0 {
		return 0, nil
	}

	fmt.Printf("%v: Importing commits...\n", repo.Name)

	commitsIter, err := log(gitRepo)
	if err != nil {
		return 0, err
	}

	bar := utils.NewProgressBar(imported)
	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		if g.options.Incremental && repo.ContainsCommit(gitCommit.Hash.String()) {
			return nil
		}

		bar.Describe(gitCommit.Committer.When.Format("2006-01-02 15"))
		_ = bar.Add(1)

		author := peopleDB.GetOrCreatePerson(g.grouper.getName(gitCommit.Author.Email, gitCommit.Author.Name))
		committer := peopleDB.GetOrCreatePerson(g.grouper.getName(gitCommit.Committer.Email, gitCommit.Committer.Name))

		commit := repo.GetOrCreateCommit(gitCommit.Hash.String())
		commit.Message = strings.TrimSpace(gitCommit.Message)
		commit.Date = gitCommit.Committer.When
		commit.CommitterID = committer.ID
		commit.DateAuthored = gitCommit.Author.When
		commit.AuthorID = author.ID

		repo.SeenAt(commit.Date, commit.DateAuthored)
		author.SeenAt(commit.Date, commit.DateAuthored)
		committer.SeenAt(commit.Date, commit.DateAuthored)

		peopleRelationsDB.GetOrCreatePersonRepo(author.ID, repo.ID).SeenAt(commit.Date, commit.DateAuthored)
		peopleRelationsDB.GetOrCreatePersonRepo(committer.ID, repo.ID).SeenAt(commit.Date, commit.DateAuthored)

		return nil
	})
	if err != nil {
		return 0, err
	}

	commitsIter, err = log(gitRepo)
	if err != nil {
		return 0, err
	}

	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		repoCommit := repo.GetCommit(gitCommit.Hash.String())

		if g.options.Incremental && len(repoCommit.Parents) > 0 {
			return nil
		}

		err = gitCommit.Parents().ForEach(func(gitParent *object.Commit) error {
			repoParent := repo.GetCommit(gitParent.Hash.String())
			repoCommit.Parents = append(repoCommit.Parents, repoParent.ID)
			repoParent.Children = append(repoParent.Children, repoCommit.ID)
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return imported, nil
}

func (g *gitHistoryImporter) countChangesToImport(repo *model.Repository, gitRepo *git.Repository) (int, error) {
	commitsIter, err := log(gitRepo)
	if err != nil {
		return 0, err
	}

	imported := 0
	total := 0
	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		if !g.options.ShouldContinue(g.commitsTotal+total, g.commitsImported+imported, gitCommit.Committer.When) {
			return g.abort
		}
		total++

		commit := repo.GetCommit(gitCommit.Hash.String())

		if g.options.Incremental && commit.FilesModified != -1 {
			return nil
		}
		imported++

		return nil
	})
	if err != nil && err != g.abort {
		return 0, err
	}

	return imported, nil
}

func (g *gitHistoryImporter) importChanges(storage archer.Storage,
	filesDB *model.Files, projsDB *model.Projects, peopleRelationsDB *model.PeopleRelations,
	repo *model.Repository, gitRepo *git.Repository,
) error {
	imported, err := g.countChangesToImport(repo, gitRepo)
	if err != nil {
		return err
	}

	if imported == 0 {
		return nil
	}

	fmt.Printf("%v: Importing changes...\n", repo.Name)

	commitsIter, err := log(gitRepo)
	if err != nil {
		return err
	}

	writeResults := func(commitFiles []*model.RepositoryCommitFiles) error {
		fmt.Printf("%v: Writing results...\n", repo.Name)

		err := storage.WriteFiles(filesDB)
		if err != nil {
			return nil
		}

		err = storage.WriteProjects(projsDB)
		if err != nil {
			return nil
		}

		err = storage.WriteRepository(repo)
		if err != nil {
			return err
		}

		err = storage.WriteRepositoryCommitFiles(commitFiles)
		if err != nil {
			return err
		}

		err = storage.WritePeopleRelations(peopleRelationsDB)
		if err != nil {
			return err
		}

		return nil
	}

	var commitFilesToWrite []*model.RepositoryCommitFiles

	bar := utils.NewProgressBar(imported)
	imported = 0
	start := time.Now()
	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		if !g.options.ShouldContinue(g.commitsTotal, g.commitsImported, gitCommit.Committer.When) {
			return g.abort
		}
		g.commitsTotal++

		commit := repo.GetCommit(gitCommit.Hash.String())

		if g.options.Incremental && commit.FilesModified != -1 {
			return nil
		}
		g.commitsImported++
		imported++

		bar.Describe(gitCommit.Committer.When.Format("2006-01-02 15"))

		commitFiles, err := storage.LoadRepositoryCommitFiles(repo, commit)
		if err != nil {
			return err
		}

		if len(commit.Parents) == 0 {
			err = g.computeChangesRootCommit(filesDB, repo, commitFiles, gitCommit)
			if err != nil {
				return err
			}

		} else if len(commit.Parents) == 1 {
			err = g.computeChangesSimpleCommit(filesDB, repo, commitFiles, gitCommit)
			if err != nil {
				return err
			}

		} else if len(commit.Parents) > 1 {
			err = g.computeChangesMergeCommit(filesDB, repo, commitFiles, gitCommit)
			if err != nil {
				return err
			}
		}

		commit.FilesModified = 0
		commit.FilesCreated = 0
		commit.FilesDeleted = 0

		if lo.SomeBy(commitFiles.List(), func(i *model.RepositoryCommitFile) bool {
			return i.LinesModified != -1
		}) {
			commit.LinesModified = 0
			commit.LinesAdded = 0
			commit.LinesDeleted = 0
		}

		for _, cf := range commitFiles.List() {
			switch cf.Change {
			case model.FileModified:
				commit.FilesModified++
			case model.FileRenamed:
				commit.FilesModified++
			case model.FileCreated:
				commit.FilesCreated++
			case model.FileDeleted:
				commit.FilesDeleted++
			}

			if cf.LinesModified != -1 {
				commit.LinesModified += cf.LinesModified
				commit.LinesAdded += cf.LinesAdded
				commit.LinesDeleted += cf.LinesDeleted
			}

			file := filesDB.GetFileByID(cf.FileID)
			file.RepositoryID = &repo.ID
			file.SeenAt(commit.Date, commit.DateAuthored)

			if file.ProjectID != nil {
				proj := projsDB.GetByID(*file.ProjectID)
				proj.SeenAt(commit.Date, commit.DateAuthored)
				proj.RepositoryID = &repo.ID
			}

			peopleRelationsDB.GetOrCreatePersonFile(commit.AuthorID, file.ID).SeenAt(commit.Date, commit.DateAuthored)
			peopleRelationsDB.GetOrCreatePersonFile(commit.CommitterID, file.ID).SeenAt(commit.Date, commit.DateAuthored)

			for _, of := range cf.OldFileIDs {
				oldFile := filesDB.GetFileByID(of)
				oldFile.RepositoryID = &repo.ID
				oldFile.SeenAt(commit.Date, commit.DateAuthored)

				if oldFile.ProjectID != nil {
					proj := projsDB.GetByID(*oldFile.ProjectID)
					proj.SeenAt(commit.Date, commit.DateAuthored)
					proj.RepositoryID = &repo.ID
				}

				peopleRelationsDB.GetOrCreatePersonFile(commit.AuthorID, oldFile.ID).SeenAt(commit.Date, commit.DateAuthored)
				peopleRelationsDB.GetOrCreatePersonFile(commit.CommitterID, oldFile.ID).SeenAt(commit.Date, commit.DateAuthored)
			}
		}

		commitFilesToWrite = append(commitFilesToWrite, commitFiles)

		if g.options.SaveEvery != nil && time.Since(start) >= *g.options.SaveEvery {
			_ = bar.Clear()

			err = writeResults(commitFilesToWrite)
			if err != nil {
				return err
			}

			commitFilesToWrite = nil
			start = time.Now()
		}

		_ = bar.Add(1)

		return nil
	})
	_ = bar.Clear()
	if err != nil && err != g.abort {
		return err
	}

	err = writeResults(commitFilesToWrite)
	if err != nil {
		return err
	}

	return nil
}

func (g *gitHistoryImporter) computeChangesMergeCommit(filesDB *model.Files, repo *model.Repository,
	commitFiles *model.RepositoryCommitFiles, gitCommit *object.Commit,
) error {
	return gitCommit.Parents().ForEach(func(gitParent *object.Commit) error {
		repoParent := repo.GetCommit(gitParent.Hash.String())

		gitChanges, err := g.computeChangesNoLines(gitCommit, gitParent)
		if err != nil {
			return err
		}

		for _, gitFile := range gitChanges {
			filePath, err := utils.PathAbs(repo.RootDir, gitFile.Name)
			if err != nil {
				return err
			}

			file := filesDB.GetOrCreateFile(filePath)

			cf := commitFiles.GetOrCreate(file.ID)
			cf.Hash = utils.IIf(gitFile.File != nil, gitFile.File, gitFile.OldFile).Hash.String()

			if gitFile.Name != gitFile.OldName {
				oldFilePath, err := utils.PathAbs(repo.RootDir, gitFile.OldName)
				if err != nil {
					return err
				}

				oldFile := filesDB.GetOrCreateFile(oldFilePath)

				cf.OldFileIDs[repoParent.ID] = oldFile.ID
			}

			if cf.Change == model.FileChangeUnknown {
				cf.Change = gitFile.Type
			} else {
				cf.Change = model.FileModified
			}
		}

		return nil
	})
}

func (g *gitHistoryImporter) computeChangesSimpleCommit(filesDB *model.Files, repo *model.Repository,
	commitFiles *model.RepositoryCommitFiles, gitCommit *object.Commit,
) error {
	return gitCommit.Parents().ForEach(func(gitParent *object.Commit) error {
		repoParent := repo.GetCommit(gitParent.Hash.String())

		gitChanges, err := g.computeChanges(gitCommit, gitParent)
		if err != nil {
			return err
		}

		for _, gitFile := range gitChanges {
			filePath, err := utils.PathAbs(repo.RootDir, gitFile.Name)
			if err != nil {
				return err
			}

			file := filesDB.GetOrCreateFile(filePath)

			cf := commitFiles.GetOrCreate(file.ID)
			cf.Hash = utils.IIf(gitFile.File != nil, gitFile.File, gitFile.OldFile).Hash.String()
			cf.Change = gitFile.Type
			cf.LinesModified = utils.Max(cf.LinesModified, gitFile.Modified)
			cf.LinesAdded = utils.Max(cf.LinesAdded, gitFile.Added)
			cf.LinesDeleted = utils.Max(cf.LinesDeleted, gitFile.Deleted)

			if gitFile.Name != gitFile.OldName {
				oldFilePath, err := utils.PathAbs(repo.RootDir, gitFile.OldName)
				if err != nil {
					return err
				}

				oldFile := filesDB.GetOrCreateFile(oldFilePath)

				cf.OldFileIDs[repoParent.ID] = oldFile.ID
			}
		}

		return nil
	})
}

func (g *gitHistoryImporter) computeChangesRootCommit(filesDB *model.Files, repo *model.Repository,
	commitFiles *model.RepositoryCommitFiles, gitCommit *object.Commit,
) error {
	gitTree, err := gitCommit.Tree()
	if err != nil {
		return err
	}

	return gitTree.Files().ForEach(func(gitFile *object.File) error {
		filePath, err := utils.PathAbs(repo.RootDir, gitFile.Name)
		if err != nil {
			return err
		}

		file := filesDB.GetOrCreateFile(filePath)

		gitLines, err := gitFile.Lines()
		if err != nil {
			return err
		}

		cf := commitFiles.GetOrCreate(file.ID)
		cf.Hash = gitFile.Hash.String()
		cf.Change = model.FileCreated
		cf.LinesModified = 0
		cf.LinesAdded = len(gitLines)
		cf.LinesDeleted = 0

		return nil
	})
}

func (g *gitHistoryImporter) computeChangesNoLines(commit *object.Commit, parent *object.Commit) ([]*gitFileChange, error) {
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

		gitChange := gitFileChange{}

		if commitFile != nil && parentFile != nil && commitFile.Name != parentFile.Name {
			gitChange.Type = model.FileRenamed
		} else if commitFile != nil && parentFile != nil {
			gitChange.Type = model.FileModified
		} else if commitFile == nil {
			gitChange.Type = model.FileDeleted
		} else {
			gitChange.Type = model.FileCreated
		}

		gitChange.File = commitFile
		if commitFile != nil {
			gitChange.Name = change.From.Name
		} else {
			gitChange.Name = change.To.Name
		}

		gitChange.OldFile = parentFile
		if parentFile != nil {
			gitChange.OldName = change.To.Name
		} else {
			gitChange.OldName = change.From.Name
		}

		result = append(result, &gitChange)
	}

	return result, nil
}

func (g *gitHistoryImporter) computeChanges(commit *object.Commit, parent *object.Commit) ([]*gitFileChange, error) {
	result, err := g.computeChangesNoLines(commit, parent)
	if err != nil {
		return nil, err
	}

	for _, gitChange := range result {
		commitContent, commitIsBinary, err := g.fileContent(gitChange.File)
		if err != nil {
			return nil, err
		}

		parentContent, parentIsBinary, err := g.fileContent(gitChange.OldFile)
		if err != nil {
			return nil, err
		}

		if !commitIsBinary && !parentIsBinary {
			commitLines := g.countLines(commitContent)
			parentLines := g.countLines(parentContent)

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
						gitChange.Deleted += g.countLines(d.Text)
					case diffmatchpatch.DiffInsert:
						gitChange.Added += g.countLines(d.Text)
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
							m := utils.Min(add, del)
							gitChange.Modified += m
							gitChange.Added += add - m
							gitChange.Deleted += del - m

							add = 0
							del = 0
						}
					}

					m := utils.Min(add, del)
					gitChange.Modified += m
					gitChange.Added += add - m
					gitChange.Deleted += del - m
				}
			}
		}
	}

	return result, nil
}

func (g *gitHistoryImporter) fileContent(f *object.File) (string, bool, error) {
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

func (g *gitHistoryImporter) countLines(text string) int {
	if text == "" {
		return 0
	}

	result := strings.Count(text, "\n")
	if text[len(text)-1] != '\n' {
		result++
	}
	return result
}

func (g *gitHistoryImporter) propagateChangesToParents(storage archer.Storage, reposDB *model.Repositories, projectsDB *model.Projects,
	filesDB *model.Files, peopleDB *model.People, statsDB *model.MonthlyStats,
) error {
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

		for _, c := range repo.ListCommits() {
			inLast6Months := now.Sub(c.Date) < 6*30*24*time.Hour
			addChanges := func(c *model.Changes) {
				c.In6Months += utils.IIf(inLast6Months, 1, 0)
				c.Total++
			}

			author := peopleDB.GetPersonByID(c.AuthorID)
			addChanges(author.Changes)

			commitFiles, err := storage.LoadRepositoryCommitFiles(repo, c)
			if err != nil {
				return err
			}

			projs := make(map[*model.Project]bool)
			dirs := make(map[*model.ProjectDirectory]bool)
			areas := make(map[*model.ProductArea]bool)
			msls := make(map[*model.MonthlyStatsLine]bool)
			for _, cf := range commitFiles.List() {
				addLines := func(c *model.Changes) {
					if cf.LinesModified != -1 {
						c.LinesModified += cf.LinesModified
						c.LinesAdded += cf.LinesAdded
						c.LinesDeleted += cf.LinesDeleted
					}
				}

				file := filesDB.GetFileByID(cf.FileID)
				files[file] = true

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

				s := statsDB.GetOrCreateLines(c.Date.Format("2006-01"), repo.ID, c.AuthorID, c.CommitterID, file.ProjectID)
				if s.Changes.IsEmpty() {
					s.Changes.Clear()
				}
				addLines(s.Changes)
				msls[s] = true
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

type gitFileChange struct {
	Type     model.FileChangeType
	Name     string
	OldName  string
	File     *object.File
	OldFile  *object.File
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
