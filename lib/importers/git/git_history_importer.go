package git

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/linediff"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
)

type HistoryImporter struct {
	console consoles.Console
	storage storages.Storage

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

func NewHistoryImporter(console consoles.Console, storage storages.Storage) *HistoryImporter {
	return &HistoryImporter{
		console: console,
		storage: storage,
		abort:   errors.New("ABORT"),
	}
}

func (i *HistoryImporter) Import(dirs []string, opts *HistoryOptions) error {
	i.console.Printf("Loading existing data...\n")

	projectsDB, err := i.storage.LoadProjects()
	if err != nil {
		return err
	}

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

	peopleRelationsDB, err := i.storage.LoadPeopleRelations()
	if err != nil {
		return err
	}

	statsDB, err := i.storage.LoadMonthlyStats()
	if err != nil {
		return err
	}

	dirs, err = findRootDirs(dirs)
	if err != nil {
		return err
	}

	i.console.Printf("Importing and grouping authors...\n")

	i.grouper, err = importPeople(peopleDB, dirs)
	if err != nil {
		return err
	}

	if opts.SaveEvery != nil {
		err = i.storage.WritePeople()
		if err != nil {
			return err
		}
	}

	for _, dir := range dirs {
		gitRepo, err := git.PlainOpen(dir)
		if err != nil {
			i.console.Printf("Skipping '%s': %s\n", dir, err)
			continue
		}

		repo := reposDB.GetOrCreate(dir)
		repo.Name = filepath.Base(dir)
		repo.VCS = "git"

		commitsImported, err := i.importCommits(peopleDB, peopleRelationsDB, repo, gitRepo, opts)
		if err != nil {
			return err
		}

		repo.FilesHead, err = i.countFilesAtHEAD(gitRepo)
		if err != nil {
			return err
		}

		if opts.SaveEvery != nil && commitsImported > 0 {
			i.console.Printf("%v: Writing results...\n", repo.Name)

			err = i.storage.WritePeople()
			if err != nil {
				return err
			}

			err := i.storage.WriteRepository(repo)
			if err != nil {
				return err
			}

			err = i.storage.WritePeopleRelations()
			if err != nil {
				return err
			}
		}

		err = i.importChanges(filesDB, projectsDB, peopleRelationsDB, repo, gitRepo, opts)
		if err != nil {
			return err
		}
	}

	i.console.Printf("Propagating changes to parents...\n")

	err = i.propagateChangesToParents(reposDB, projectsDB, filesDB, peopleDB, statsDB)
	if err != nil {
		return err
	}

	i.console.Printf("Writing results...\n")

	err = i.storage.WriteProjects()
	if err != nil {
		return err
	}

	err = i.storage.WriteFiles()
	if err != nil {
		return err
	}

	err = i.storage.WritePeople()
	if err != nil {
		return err
	}

	err = i.storage.WriteRepositories()
	if err != nil {
		return err
	}

	err = i.storage.WritePeopleRelations()
	if err != nil {
		return err
	}

	err = i.storage.WriteMonthlyStats()
	if err != nil {
		return err
	}

	return nil
}

func (i *HistoryImporter) countFilesAtHEAD(gitRepo *git.Repository) (int, error) {
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

func (i *HistoryImporter) countCommitsToImport(repo *model.Repository, gitRepo *git.Repository, opts *HistoryOptions) (int, error) {
	commitsIter, err := log(gitRepo)
	if err != nil {
		return 0, err
	}

	imported := 0
	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		if opts.Incremental && repo.ContainsCommit(gitCommit.Hash.String()) {
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

func (i *HistoryImporter) importCommits(peopleDB *model.People, peopleRelationsDB *model.PeopleRelations,
	repo *model.Repository, gitRepo *git.Repository,
	opts *HistoryOptions,
) (int, error) {
	imported, err := i.countCommitsToImport(repo, gitRepo, opts)
	if err != nil {
		return 0, err
	}

	if imported == 0 {
		return 0, nil
	}

	i.console.Printf("%v: Importing commits...\n", repo.Name)

	commitsIter, err := log(gitRepo)
	if err != nil {
		return 0, err
	}

	bar := utils.NewProgressBar(imported)
	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		if opts.Incremental && repo.ContainsCommit(gitCommit.Hash.String()) {
			return nil
		}

		bar.Describe(gitCommit.Committer.When.Format("2006-01-02 15"))
		_ = bar.Add(1)

		author := peopleDB.GetOrCreatePerson(i.grouper.getName(gitCommit.Author.Email, gitCommit.Author.Name))
		committer := peopleDB.GetOrCreatePerson(i.grouper.getName(gitCommit.Committer.Email, gitCommit.Committer.Name))

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

		if opts.Incremental && len(repoCommit.Parents) > 0 {
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

func (i *HistoryImporter) listChangesToImport(repo *model.Repository, gitRepo *git.Repository, opts *HistoryOptions) ([]*changeWork, error) {
	commitsIter, err := log(gitRepo)
	if err != nil {
		return nil, err
	}

	var result []*changeWork

	imported := 0
	total := 0
	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		if !opts.ShouldContinue(i.commitsTotal+total, i.commitsImported+imported, gitCommit.Committer.When) {
			return i.abort
		}
		total++

		commit := repo.GetCommit(gitCommit.Hash.String())

		if opts.Incremental && commit.FilesModified != -1 {
			return nil
		}
		imported++

		result = append(result, &changeWork{gitCommit: gitCommit})

		return nil
	})
	if err != nil && err != i.abort {
		return nil, err
	}

	//for i := range result {
	//	j := rand.Intn(i + 1)
	//	result[i], result[j] = result[j], result[i]
	//}

	return result, nil
}

type changeWork struct {
	gitCommit   *object.Commit
	commit      *model.RepositoryCommit
	commitFiles *model.RepositoryCommitFiles
}

func (i *HistoryImporter) importChanges(filesDB *model.Files, projsDB *model.Projects, peopleRelationsDB *model.PeopleRelations,
	repo *model.Repository, gitRepo *git.Repository,
	opts *HistoryOptions,
) error {
	toProcess, err := i.listChangesToImport(repo, gitRepo, opts)
	if err != nil {
		return err
	}

	if len(toProcess) == 0 {
		return nil
	}

	i.console.Printf("%v: Importing changes...\n", repo.Name)

	writeResults := func(commitFiles []*model.RepositoryCommitFiles) error {
		i.console.Printf("%v: Writing results...\n", repo.Name)

		err := i.storage.WriteFiles()
		if err != nil {
			return nil
		}

		err = i.storage.WriteProjects()
		if err != nil {
			return nil
		}

		err = i.storage.WriteRepository(repo)
		if err != nil {
			return err
		}

		err = i.storage.WriteRepositoryCommitFiles(commitFiles)
		if err != nil {
			return err
		}

		err = i.storage.WritePeopleRelations()
		if err != nil {
			return err
		}

		return nil
	}

	bar := utils.NewProgressBar(len(toProcess))
	start := time.Now()
	var commitFilesToWrite []*model.RepositoryCommitFiles
	for _, w := range toProcess {
		bar.Describe(w.gitCommit.Committer.When.Format("2006-01-02 15"))

		err = i.importCommitChanges(filesDB, repo, w)
		if err != nil {
			return err
		}

		commit := w.commit
		commitFiles := w.commitFiles

		commit.FilesModified = 0
		commit.FilesCreated = 0
		commit.FilesDeleted = 0

		if lo.SomeBy(commitFiles.List(), func(i *model.RepositoryCommitFile) bool {
			return i.LinesModified != -1
		}) {
			commit.LinesModified = 0
			commit.LinesAdded = 0
			commit.LinesDeleted = 0
		} else {
			commit.LinesModified = -1
			commit.LinesAdded = -1
			commit.LinesDeleted = -1
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

		if opts.SaveEvery != nil && time.Since(start) >= *opts.SaveEvery {
			_ = bar.Clear()

			err = writeResults(commitFilesToWrite)
			if err != nil {
				return err
			}

			commitFilesToWrite = nil
			start = time.Now()
		}

		_ = bar.Add(1)
	}

	err = writeResults(commitFilesToWrite)
	if err != nil {
		return err
	}

	return nil
}

func (i *HistoryImporter) importCommitChanges(filesDB *model.Files, repo *model.Repository, w *changeWork) error {
	commit := repo.GetCommit(w.gitCommit.Hash.String())

	commitFiles, err := i.storage.LoadRepositoryCommitFiles(repo, commit)
	if err != nil {
		return err
	}

	if len(commit.Parents) == 0 {
		err = i.computeChangesRootCommit(filesDB, repo, commitFiles, w.gitCommit)

	} else if len(commit.Parents) == 1 {
		err = i.computeChangesSimpleCommit(filesDB, repo, commitFiles, w.gitCommit)

	} else if len(commit.Parents) > 1 {
		err = i.computeChangesMergeCommit(filesDB, repo, commitFiles, w.gitCommit)
	}
	if err != nil {
		return err
	}

	w.gitCommit = nil
	w.commit = commit
	w.commitFiles = commitFiles
	return nil
}

func (i *HistoryImporter) computeChangesMergeCommit(filesDB *model.Files, repo *model.Repository,
	commitFiles *model.RepositoryCommitFiles, gitCommit *object.Commit,
) error {
	changesPerFile := make(map[string]map[*object.Commit]*gitFileChange)
	parents := 0

	err := gitCommit.Parents().ForEach(func(gitParent *object.Commit) error {
		parents++

		gitChanges, err := i.computeChangesNoLines(gitCommit, gitParent)
		if err != nil {
			return err
		}

		for _, gitFile := range gitChanges {
			filePath, err := utils.PathAbs(repo.RootDir, gitFile.Name)
			if err != nil {
				return err
			}

			cs, ok := changesPerFile[filePath]
			if !ok {
				cs = make(map[*object.Commit]*gitFileChange)
				changesPerFile[filePath] = cs
			}

			cs[gitParent] = gitFile
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Only consider files changes by this commit, meaning that they didn't come from any parent
	// That is the same to say that the file was changed in all parents
	for filePath, parentCommits := range changesPerFile {
		if len(parentCommits) != parents {
			continue
		}

		file := filesDB.GetOrCreateFile(filePath)
		cf := commitFiles.GetOrCreate(file.ID)
		var minChange *gitFileChange

		for gitParent, gitFile := range parentCommits {
			repoParent := repo.GetCommit(gitParent.Hash.String())

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
			} else if cf.Change != gitFile.Type {
				cf.Change = model.FileModified
			}

			err = i.computeLinesChanged(gitFile)
			if err != nil {
				return err
			}

			if minChange == nil || minChange.Total() > gitFile.Total() {
				minChange = gitFile
			}
		}

		if minChange.Modified != -1 {
			cf.LinesModified = minChange.Modified
			cf.LinesAdded = minChange.Added
			cf.LinesDeleted = minChange.Deleted
		}
	}

	return nil
}

func (i *HistoryImporter) computeChangesSimpleCommit(filesDB *model.Files, repo *model.Repository,
	commitFiles *model.RepositoryCommitFiles, gitCommit *object.Commit,
) error {
	return gitCommit.Parents().ForEach(func(gitParent *object.Commit) error {
		repoParent := repo.GetCommit(gitParent.Hash.String())

		gitChanges, err := i.computeChanges(gitCommit, gitParent)
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

func (i *HistoryImporter) computeChangesRootCommit(filesDB *model.Files, repo *model.Repository,
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

func (i *HistoryImporter) computeChangesNoLines(commit *object.Commit, parent *object.Commit) ([]*gitFileChange, error) {
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

func (i *HistoryImporter) computeChanges(commit *object.Commit, parent *object.Commit) ([]*gitFileChange, error) {
	result, err := i.computeChangesNoLines(commit, parent)
	if err != nil {
		return nil, err
	}

	for _, gitChange := range result {
		err = i.computeLinesChanged(gitChange)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (i *HistoryImporter) computeLinesChanged(gitChange *gitFileChange) error {
	commitContent, commitIsBinary, err := i.fileContent(gitChange.File)
	if err != nil {
		return err
	}

	parentContent, parentIsBinary, err := i.fileContent(gitChange.OldFile)
	if err != nil {
		return err
	}

	if commitIsBinary || parentIsBinary {
		gitChange.Modified = -1
		gitChange.Added = -1
		gitChange.Deleted = -1
		return nil
	}

	commitLines := i.countLines(commitContent)
	parentLines := i.countLines(parentContent)

	gitChange.Modified = 0
	gitChange.Added = 0
	gitChange.Deleted = 0

	if parentLines == 0 {
		gitChange.Added += commitLines

	} else if commitLines == 0 {
		gitChange.Deleted += parentLines

	} else {
		diffs := linediff.DoWithTimeout(parentContent, commitContent, time.Minute)

		// Modified is defined as changes that happened without a line without change in the middle
		add := 0
		del := 0
		for _, line := range diffs {
			switch line.Type {
			case linediff.DiffInsert:
				add += line.Lines
			case linediff.DiffDelete:
				del += line.Lines
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

	return nil
}

func (i *HistoryImporter) fileContent(f *object.File) (string, bool, error) {
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

func (i *HistoryImporter) countLines(text string) int {
	if text == "" {
		return 0
	}

	result := strings.Count(text, "\n")
	if text[len(text)-1] != '\n' {
		result++
	}
	return result
}

func (i *HistoryImporter) propagateChangesToParents(reposDB *model.Repositories, projectsDB *model.Projects, filesDB *model.Files, peopleDB *model.People, statsDB *model.MonthlyStats) error {
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

			commitFiles, err := i.storage.LoadRepositoryCommitFiles(repo, c)
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

				s := statsDB.GetOrCreateLines(c.Date.Format("2006-01"), repo.ID, c.AuthorID, c.CommitterID, file.ID, file.ProjectID)
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

func (c *gitFileChange) Total() int {
	return c.Modified + c.Added + c.Deleted
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
