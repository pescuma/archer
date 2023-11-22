package git

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-set/v2"
	"github.com/hhatto/gocloc"
	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type gitBlameImporter struct {
	rootDirs []string
	options  BlameOptions

	abort error
}

type BlameOptions struct {
	Incremental      bool
	MaxImportedFiles *int
	SaveEvery        *time.Duration
}

func NewBlameImporter(rootDirs []string, options BlameOptions) archer.Importer {
	return &gitBlameImporter{
		rootDirs: rootDirs,
		options:  options,
		abort:    errors.New("ABORT"),
	}
}

type blameWork struct {
	repo         *model.Repository
	gitRepo      *git.Repository
	gitCommit    *object.Commit
	file         *model.File
	relativePath string
	lastMod      string
}

func (g *gitBlameImporter) Import(storage archer.Storage) error {
	fmt.Printf("Loading existing data...\n")

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

	statsDB, err := storage.LoadMonthlyStats()
	if err != nil {
		return err
	}

	fmt.Printf("Finding out which files to process...\n")

	imported := 0

	for _, rootDir := range g.rootDirs {
		rootDir, err := filepath.Abs(rootDir)
		if err != nil {
			return err
		}

		gitRepo, err := git.PlainOpen(rootDir)
		if err != nil {
			fmt.Printf("Skipping %s: %s\n", rootDir, err)
			continue
		}

		gitHead, err := gitRepo.Head()
		if err != nil {
			return err
		}

		gitCommit, err := gitRepo.CommitObject(gitHead.Hash())
		if err != nil {
			return err
		}

		gitTree, err := gitCommit.Tree()
		if err != nil {
			return err
		}

		repo := reposDB.Get(rootDir)

		if repo == nil {
			fmt.Printf("%v: repository history not fully imported. run 'import git history'\n", rootDir)
			continue
		}

		importedHistory, err := g.checkImportedHistory(repo, gitRepo)
		if err != nil {
			return err
		}

		if !importedHistory {
			fmt.Printf("%v: repository history not fully imported. run 'import git history'\n", repo.Name)
			continue
		}

		repo.SeenAt(time.Now())

		imported, err = g.computeBlame(storage, filesDB, peopleDB, reposDB, statsDB, repo, gitRepo, gitTree, gitCommit, imported)
		if err != nil {
			return err
		}

		err = g.deleteBlame(storage, filesDB, repo, gitTree)
		if err != nil {
			return err
		}

	}

	err = g.writeResults(storage, filesDB, peopleDB, reposDB, statsDB)
	if err != nil {
		return err
	}

	return nil
}

func (g *gitBlameImporter) writeResults(storage archer.Storage,
	filesDB *model.Files, peopleDB *model.People, reposDB *model.Repositories, statsDB *model.MonthlyStats,
) error {

	fmt.Printf("Propagating changes to parents...\n")

	err := g.propagateChangesToParents(storage, filesDB, peopleDB, reposDB, statsDB)
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

	err = storage.WritePeople(peopleDB)
	if err != nil {
		return err
	}

	err = storage.WriteRepositories(reposDB)
	if err != nil {
		return err
	}

	err = storage.WriteMonthlyStats(statsDB)
	if err != nil {
		return err
	}

	return nil
}

func (g *gitBlameImporter) deleteBlame(storage archer.Storage, filesDB *model.Files, repo *model.Repository,
	gitTree *object.Tree,
) error {
	toDelete, err := g.listToDelete(filesDB, repo, gitTree)
	if err != nil {
		return err
	}

	if len(toDelete) == 0 {
		return nil
	}

	fmt.Printf("%v: Cleaning %v deleted files...\n", repo.Name, len(toDelete))

	bar := utils.NewProgressBar(len(toDelete))
	for _, file := range toDelete {
		err = g.deleteFileBlame(storage, file)
		if err != nil {
			return err
		}

		_ = bar.Add(1)
	}

	return nil
}

func (g *gitBlameImporter) listToDelete(filesDB *model.Files, repo *model.Repository, gitTree *object.Tree) (map[string]*model.File, error) {
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

	for _, file := range filesDB.ListFiles() {
		if file.RepositoryID == nil || *file.RepositoryID != repo.ID {
			continue
		}

		if !existing.Contains(file.Path) {
			result[file.Path] = file
		}
	}

	return result, nil
}

func (g *gitBlameImporter) deleteFileBlame(storage archer.Storage, file *model.File) error {
	contents, err := storage.LoadFileContents(file.ID)
	if err != nil {
		return err
	}

	if len(contents.Lines) == 0 {
		return nil
	}

	contents.Lines = nil

	err = storage.WriteFileContents(contents)
	if err != nil {
		return err
	}

	return nil
}

func (g *gitBlameImporter) computeBlame(storage archer.Storage,
	filesDB *model.Files, peopleDB *model.People, reposDB *model.Repositories, statsDB *model.MonthlyStats,
	repo *model.Repository, gitRepo *git.Repository, gitTree *object.Tree, gitCommit *object.Commit,
	imported int,
) (int, error) {
	toProcess, err := g.listToCompute(filesDB, repo, gitRepo, gitTree, gitCommit)
	if err != nil {
		return 0, err
	}

	if len(toProcess) == 0 {
		return imported, nil
	}

	fmt.Printf("%v: Computing blame of %v files...\n", repo.Name, len(toProcess))

	cache, err := g.createCache(storage, filesDB, repo, gitRepo)
	if err != nil {
		return 0, err
	}

	writeMutex := sync.RWMutex{}

	bar := utils.NewProgressBar(len(toProcess))
	start := time.Now()

	group := utils.ParallelFor(toProcess,
		func(w *blameWork) (*blameWork, error) {
			writeMutex.RLock()
			defer writeMutex.RUnlock()

			err = g.computeFileBlame(storage, w, cache)
			return w, err
		})

	for w := range group.Output {
		imported++

		if g.options.SaveEvery != nil && time.Since(start) >= *g.options.SaveEvery {
			_ = bar.Clear()

			writeMutex.Lock()
			err = g.writeResults(storage, filesDB, peopleDB, reposDB, statsDB)
			writeMutex.Unlock()

			if err != nil {
				return 0, err
			}

			start = time.Now()
		}

		bar.Describe(utils.TruncateFilename(w.relativePath))
		_ = bar.Add(1)
	}

	if err = group.Error(); err != nil {
		return imported, err
	}

	return imported, nil
}

type blameCacheImpl struct {
	repo    *git.Repository
	trees   *utils.Cache[plumbing.Hash, *object.Tree]
	commits *utils.Cache[plumbing.Hash, *BlameCommitCache]
	load    func(plumbing.Hash) (*BlameCommitCache, error)
}

func (c *blameCacheImpl) GetCommitTreeEntry(commit *object.Commit, path string) (*object.Tree, *object.TreeEntry, error) {
	loader := func(hash plumbing.Hash) (*object.Tree, error) {
		tree, err := object.GetTree(c.repo.Storer, hash)
		if err != nil {
			return nil, err
		}

		// Load internal caches
		_, _ = tree.FindEntry("")

		return tree, nil
	}

	tc, err := c.trees.Get(commit.TreeHash, loader)
	if err != nil {
		return nil, nil, err
	}

	paths := strings.Split(path, "/")
	t := tc
	for i := 0; i < len(paths)-1; i++ {
		ce, err := t.FindEntry(paths[i])
		if err != nil {
			return nil, nil, err
		}

		ct, err := c.trees.Get(ce.Hash, loader)
		if err != nil {
			return nil, nil, err
		}

		t = ct
	}

	e, err := t.FindEntry(paths[len(paths)-1])
	if err != nil {
		return nil, nil, err
	}

	return tc, e, err
}

func (c *blameCacheImpl) GetCommit(hash plumbing.Hash) (*BlameCommitCache, error) {
	return c.commits.Get(hash, func(hash plumbing.Hash) (*BlameCommitCache, error) {
		return c.load(hash)
	})
}

func (g *gitBlameImporter) createCache(storage archer.Storage, filesDB *model.Files, repo *model.Repository, gitRepo *git.Repository) (BlameCache, error) {
	cache := &blameCacheImpl{
		repo:    gitRepo,
		commits: utils.NewCache[plumbing.Hash, *BlameCommitCache](),
		trees:   utils.NewCache[plumbing.Hash, *object.Tree](),
	}

	cache.load = func(hash plumbing.Hash) (*BlameCommitCache, error) {
		result := &BlameCommitCache{
			Parents: make(map[plumbing.Hash]*BlameParentCache),
			Files:   make(map[string]*BlameFileCache),
		}

		repoCommit := repo.GetCommit(hash.String())

		for _, repoParentID := range repoCommit.Parents {
			repoParent := repo.GetCommitByID(repoParentID)

			gitParent, err := gitRepo.CommitObject(plumbing.NewHash(repoParent.Hash))
			if err != nil {
				return nil, err
			}

			result.Parents[gitParent.Hash] = &BlameParentCache{
				Commit:  gitParent,
				Renames: make(map[string]string),
			}
		}

		getFilename := func(fileID model.UUID) (string, error) {
			f := filesDB.GetFileByID(fileID)

			rel, err := filepath.Rel(repo.RootDir, f.Path)
			if err != nil {
				return "", err
			}

			rel = strings.ReplaceAll(rel, string(filepath.Separator), "/")
			return rel, nil
		}

		files, err := storage.LoadRepositoryCommitFiles(repo, repoCommit)
		if err != nil {
			return nil, err
		}

		for _, commitFile := range files.List() {
			filename, err := getFilename(commitFile.FileID)
			if err != nil {
				return nil, err
			}

			result.Files[filename] = &BlameFileCache{
				Hash:    plumbing.NewHash(commitFile.Hash),
				Created: commitFile.Change == model.FileCreated,
			}

			for repoParentID, oldFileID := range commitFile.OldFileIDs {
				oldFilename, err := getFilename(oldFileID)
				if err != nil {
					return nil, err
				}

				repoParent := repo.GetCommitByID(repoParentID)

				parentCache := result.Parents[plumbing.NewHash(repoParent.Hash)]
				parentCache.Renames[filename] = oldFilename
			}
		}

		return result, nil
	}

	return cache, nil
}

func (g *gitBlameImporter) listToCompute(filesDB *model.Files, repo *model.Repository,
	gitRepo *git.Repository, gitTree *object.Tree, gitCommit *object.Commit,
) ([]*blameWork, error) {
	var result []*blameWork

	err := gitTree.Files().ForEach(func(gitFile *object.File) error {
		if !g.options.ShouldContinue(len(result)) {
			return g.abort
		}

		path, err := utils.PathAbs(repo.RootDir, gitFile.Name)
		if err != nil {
			return err
		}

		file := filesDB.GetFile(path)
		if file == nil {
			return fmt.Errorf("file not found in repo %v: %v", repo.Name, path)
		}

		lastMod := ""
		stat, err := os.Stat(path)
		if err == nil {
			file.SeenAt(time.Now(), stat.ModTime())

			lastMod = stat.ModTime().String()
			if g.options.Incremental && lastMod == file.Data["blame:last_modified"] {
				return nil
			}
		}

		isText, err := utils.IsTextReader(gitFile.Reader())
		if err != nil {
			return err
		}

		if !isText {
			return nil
		}

		result = append(result, &blameWork{
			repo:         repo,
			gitRepo:      gitRepo,
			gitCommit:    gitCommit,
			file:         file,
			lastMod:      lastMod,
			relativePath: gitFile.Name,
		})

		return nil
	})
	if err != nil && err != g.abort {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].file.Path <= result[j].file.Path
	})

	return result, nil
}

func (g *gitBlameImporter) computeFileBlame(storage archer.Storage, w *blameWork, cache BlameCache) error {
	blame, err := Blame(w.gitCommit, w.relativePath, cache)
	if err != nil {
		return err
	}

	contents := strings.Join(lo.Map(blame.Lines, func(item *Line, _ int) string { return item.Text }), "\n")
	lineTypes, err := g.computeLOC(filepath.Base(w.file.Path), contents)
	if err != nil {
		return err
	}

	fileLines, err := storage.LoadFileContents(w.file.ID)
	if err != nil {
		return err
	}

	for i, blameLine := range blame.Lines {
		var lt model.FileLineType
		if i < len(lineTypes) {
			lt = lineTypes[i]
		} else if strings.TrimSpace(blameLine.Text) == "" {
			lt = model.BlankFileLine
		} else {
			lt = model.CodeFileLine
		}

		var fileLine *model.FileLine
		if i == len(fileLines.Lines) {
			fileLine = fileLines.AppendLine()
		} else {
			fileLine = fileLines.Lines[i]
		}

		commit := w.repo.GetCommit(blameLine.Hash.String())
		if commit == nil {
			return fmt.Errorf("missing commit '%v':'%v'. run 'import git history' before importing blame", w.repo.Name, blameLine.Hash.String())
		}

		fileLine.ProjectID = w.file.ProjectID
		fileLine.RepositoryID = &w.repo.ID
		fileLine.CommitID = &commit.ID
		fileLine.AuthorID = &commit.AuthorID
		fileLine.CommitterID = &commit.CommitterID
		fileLine.Date = commit.Date
		fileLine.Type = lt
		fileLine.Text = blameLine.Text
	}

	fileLines.Lines = fileLines.Lines[:len(blame.Lines)]

	if w.lastMod != "" {
		w.file.Data["blame:last_modified"] = w.lastMod
	} else {
		delete(w.file.Data, "blame:last_modified")
	}

	err = storage.WriteFileContents(fileLines)
	if err != nil {
		return err
	}

	err = storage.WriteFile(w.file)
	if err != nil {
		return err
	}

	return nil
}

func (g *gitBlameImporter) computeLOC(name string, contents string) ([]model.FileLineType, error) {
	tmp, err := os.CreateTemp("", "archer-*-"+name)
	if err != nil {
		return nil, errors.Wrapf(err, "error computing lines of code")
	}

	defer os.Remove(tmp.Name())

	{
		defer tmp.Close()

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

func (g *gitBlameImporter) propagateChangesToParents(storage archer.Storage,
	filesDB *model.Files, peopleDB *model.People, reposDB *model.Repositories, statsDB *model.MonthlyStats,
) error {
	blames, err := storage.QueryBlamePerAuthor()
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

	for _, blame := range blames {
		c := commits[blame.CommitID]
		pa := peopleDB.GetPersonByID(blame.AuthorID)
		file := filesDB.GetFileByID(blame.FileID)

		s := statsDB.GetOrCreateLines(c.Date.Format("2006-01"), blame.RepositoryID, blame.AuthorID, blame.CommitterID, file.ProjectID)
		if s.Blame.IsEmpty() {
			s.Blame.Clear()
		}

		switch blame.LineType {
		case model.CodeFileLine:
			c.Blame.Code += blame.Lines
			pa.Blame.Code += blame.Lines
			s.Blame.Code += blame.Lines
		case model.CommentFileLine:
			c.Blame.Comment += blame.Lines
			pa.Blame.Comment += blame.Lines
			s.Blame.Comment += blame.Lines
		case model.BlankFileLine:
			c.Blame.Blank += blame.Lines
			pa.Blame.Blank += blame.Lines
			s.Blame.Blank += blame.Lines
		default:
			panic(blame.LineType)
		}
	}

	return nil
}

func (g *gitBlameImporter) checkImportedHistory(repo *model.Repository, gitRepo *git.Repository) (bool, error) {
	if repo == nil {
		return false, nil
	}

	commitsIter, err := log(gitRepo)
	if err != nil {
		return false, err
	}

	err = commitsIter.ForEach(func(gitCommit *object.Commit) error {
		repoCommit := repo.GetCommit(gitCommit.Hash.String())
		if repoCommit == nil || repoCommit.FilesModified == -1 {
			return g.abort
		}

		return nil
	})
	if err != nil && err != g.abort {
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
