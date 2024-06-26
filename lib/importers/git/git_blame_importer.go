package git

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-set/v2"
	"github.com/hhatto/gocloc"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
)

type BlameImporter struct {
	console consoles.Console
	storage storages.Storage

	abort error
}

type BlameOptions struct {
	Branch           string
	Incremental      bool
	MaxImportedFiles *int
}

func NewBlameImporter(console consoles.Console, storage storages.Storage) *BlameImporter {
	return &BlameImporter{
		console: console,
		storage: storage,
		abort:   errors.New("ABORT"),
	}
}

type blameWork struct {
	repo         *model.Repository
	gitRepo      *git.Repository
	gitCommit    *object.Commit
	gitFileHash  string
	file         *model.File
	relativePath string
}

func (i *BlameImporter) Import(dirs []string, opts *BlameOptions) error {
	filesDB, err := i.storage.LoadFiles()
	if err != nil {
		return err
	}

	reposDB, err := i.storage.LoadRepositories()
	if err != nil {
		return err
	}

	dirs, err = findRootDirs(dirs)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		gitRepo, err := git.PlainOpen(dir)
		if err != nil {
			i.console.Printf("Skipping %s: %s\n", dir, err)
			continue
		}

		repo := reposDB.Get(dir)
		if repo == nil {
			i.console.Printf("%v: Repository history not fully imported. run 'import git history'\n", dir)
			continue
		}

		i.console.Printf("%v: Finding out which files to process...\n", repo.Name)

		_, gitRevision, err := findBranchHash(repo, gitRepo, opts.Branch)
		if err != nil {
			return err
		}

		gitCommit, err := gitRepo.CommitObject(gitRevision)
		if err != nil {
			return err
		}

		gitTree, err := gitCommit.Tree()
		if err != nil {
			return err
		}

		importedHistory, err := i.checkImportedHistory(repo, gitRepo, gitRevision)
		if err != nil {
			return err
		}

		if !importedHistory {
			i.console.Printf("%v: Repository history not fully imported. run 'import git history'\n", repo.Name)
			continue
		}

		repo.SeenAt(time.Now())

		_, err = i.importBlame(filesDB, repo, gitRepo, gitTree, gitCommit, opts)
		if err != nil {
			return err
		}

		err = i.deleteBlame(filesDB, repo, gitTree)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *BlameImporter) deleteBlame(filesDB *model.Files, repo *model.Repository, gitTree *object.Tree) error {
	toDelete, err := i.listToDelete(filesDB, repo, gitTree)
	if err != nil {
		return err
	}

	if len(toDelete) == 0 {
		return nil
	}

	i.console.Printf("%v: Cleaning %v deleted files...\n", repo.Name, len(toDelete))

	bar := utils.NewProgressBar(len(toDelete))
	for _, file := range toDelete {
		err = i.deleteFileBlame(file)
		if err != nil {
			return err
		}

		_ = bar.Add(1)
	}

	return nil
}

func (i *BlameImporter) listToDelete(filesDB *model.Files, repo *model.Repository, gitTree *object.Tree) (map[string]*model.File, error) {
	existing := set.New[string](1000)

	err := gitTree.Files().ForEach(func(file *object.File) error {
		path, err := utils.PathAbs(repo.RootDir, file.Name)
		if err != nil {
			return err
		}

		existing.Insert(path)

		return nil
	})
	if err != nil {
		return nil, err
	}

	result := make(map[string]*model.File)

	for _, file := range filesDB.List() {
		if file.RepositoryID == nil || *file.RepositoryID != repo.ID {
			continue
		}

		if !existing.Contains(file.Path) {
			result[file.Path] = file
		}
	}

	return result, nil
}

func (i *BlameImporter) deleteFileBlame(file *model.File) error {
	contents, err := i.storage.LoadFileContents(file.ID)
	if err != nil {
		return err
	}

	if len(contents.Lines) == 0 {
		return nil
	}

	contents.Lines = nil

	err = i.storage.WriteFileContents(contents)
	if err != nil {
		return err
	}

	return nil
}

func (i *BlameImporter) importBlame(filesDB *model.Files,
	repo *model.Repository, gitRepo *git.Repository, gitTree *object.Tree, gitCommit *object.Commit,
	opts *BlameOptions,
) (int, error) {
	toProcess, err := i.listToCompute(filesDB, repo, gitRepo, gitTree, gitCommit, opts)
	if err != nil {
		return 0, err
	}

	if len(toProcess) == 0 {
		return 0, nil
	}

	i.console.Printf("%v: Computing blame of %v files...\n", repo.Name, len(toProcess))

	cache := newBlameCache(i.storage, filesDB, repo, gitRepo)

	bar := utils.NewProgressBar(len(toProcess))
	for _, w := range toProcess {
		bar.Describe(utils.TruncateFilename(w.relativePath))

		err = i.computeFileBlame(w, cache)
		if err != nil {
			return 0, err
		}

		_ = bar.Add(1)
	}

	return len(toProcess), nil
}

func (i *BlameImporter) listToCompute(filesDB *model.Files, repo *model.Repository, gitRepo *git.Repository, gitTree *object.Tree, gitCommit *object.Commit, opts *BlameOptions) ([]*blameWork, error) {
	var result []*blameWork

	err := gitTree.Files().ForEach(func(gitFile *object.File) error {
		if !opts.ShouldContinue(len(result)) {
			return i.abort
		}

		path, err := utils.PathAbs(repo.RootDir, gitFile.Name)
		if err != nil {
			return err
		}

		file := filesDB.Get(path)
		if file == nil {
			return fmt.Errorf("file not found in repo %v: %v", repo.Name, path)
		}

		hash := gitFile.Hash.String()
		if opts.Incremental && hash == file.Data["blame:last_hash"] {
			return nil
		}

		reader, err := gitFile.Reader()
		if err != nil {
			return err
		}

		isText := utils.IsTextReader(gitFile.Name, reader)
		if !isText {
			return nil
		}

		result = append(result, &blameWork{
			repo:         repo,
			gitRepo:      gitRepo,
			gitCommit:    gitCommit,
			gitFileHash:  hash,
			file:         file,
			relativePath: gitFile.Name,
		})

		return nil
	})
	if err != nil && err != i.abort {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].file.Path <= result[j].file.Path
	})
	//for i := range result {
	//	j := rand.Intn(i + 1)
	//	result[i], result[j] = result[j], result[i]
	//}

	return result, nil
}

func (i *BlameImporter) computeFileBlame(w *blameWork, cache BlameCache) error {
	blameLines, err := Blame(w.relativePath, w.gitCommit, cache)
	if err != nil {
		return err
	}

	contents := strings.Join(lo.Map(blameLines, func(item *blameLine, _ int) string { return item.Text }), "\n")

	lineTypes, err := i.computeLOC(filepath.Base(w.file.Path), contents)
	if err != nil {
		return err
	}

	fileLines, err := i.storage.LoadFileContents(w.file.ID)
	if err != nil {
		return err
	}

	for j, bl := range blameLines {
		var lt model.FileLineType
		if j < len(lineTypes) {
			lt = lineTypes[j]
		} else if strings.TrimSpace(bl.Text) == "" {
			lt = model.BlankFileLine
		} else {
			lt = model.CodeFileLine
		}

		var fileLine *model.FileLine
		if j == len(fileLines.Lines) {
			fileLine = fileLines.AppendLine()
		} else {
			fileLine = fileLines.Lines[j]
		}

		commit := w.repo.GetCommit(bl.CommitHash)
		if commit == nil {
			return fmt.Errorf("missing commit '%v':'%v'. run 'import git history' before importing blame", w.repo.Name, bl.CommitHash)
		}

		fileLine.ProjectID = w.file.ProjectID
		fileLine.RepositoryID = &w.repo.ID
		fileLine.CommitID = &commit.ID
		fileLine.AuthorID = &commit.AuthorIDs[0]
		fileLine.CommitterID = &commit.CommitterID
		fileLine.Date = commit.Date
		fileLine.Type = lt
		fileLine.Text = bl.Text
	}

	fileLines.Lines = fileLines.Lines[:len(blameLines)]

	w.file.Data["blame:last_hash"] = w.gitFileHash

	err = i.storage.WriteFileContents(fileLines)
	if err != nil {
		return err
	}

	err = i.storage.WriteFile(w.file)
	if err != nil {
		return err
	}

	return nil
}

func (i *BlameImporter) computeLOC(name string, contents string) ([]model.FileLineType, error) {
	tmp, err := os.CreateTemp("", "archer-*-"+name)
	if err != nil {
		return nil, errors.Wrapf(err, "error computing lines of code")
	}

	defer func() {
		_ = os.Remove(tmp.Name())
	}()

	{
		defer func() {
			_ = tmp.Close()
		}()

		_, err = tmp.Write([]byte(contents))
		if err != nil {
			return nil, errors.Wrapf(err, "error computing lines of code")
		}
	}

	languages := gocloc.NewDefinedLanguages()
	options := gocloc.NewClocOptions()

	var result []model.FileLineType
	options.OnCode = func(line string) {
		result = append(result, model.CodeFileLine)
	}
	options.OnComment = func(line string) {
		result = append(result, model.CommentFileLine)
	}
	options.OnBlank = func(line string) {
		result = append(result, model.BlankFileLine)
	}

	paths := []string{tmp.Name()}

	processor := gocloc.NewProcessor(languages, options)
	_, err = processor.Analyze(paths)
	if err != nil {
		return nil, errors.Wrapf(err, "error computing lines of code")
	}

	return result, nil
}

func (i *BlameImporter) checkImportedHistory(repo *model.Repository, gitRepo *git.Repository, gitRevision plumbing.Hash) (bool, error) {
	if repo == nil {
		return false, nil
	}

	commitsIter, err := log(gitRepo, gitRevision)
	if err != nil {
		return false, err
	}

	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		repoCommit := repo.GetCommit(gitCommit.Hash.String())
		if repoCommit == nil || repoCommit.FilesModified == -1 {
			return i.abort
		}

		return nil
	})
	if err != nil && err != i.abort {
		return false, err
	}

	return err == nil, nil
}

func (l *BlameOptions) ShouldContinue(imported int) bool {
	if l.MaxImportedFiles != nil && imported >= *l.MaxImportedFiles {
		return false
	}

	return true
}
