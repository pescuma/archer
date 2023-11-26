package git

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-set/v2"
	"github.com/hhatto/gocloc"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/caches"
	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/importers"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
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

func NewBlameImporter(rootDirs []string, options BlameOptions) importers.Importer {
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

func (g *gitBlameImporter) Import(console consoles.Console, storage storages.Storage) error {
	console.Printf("Loading existing data...\n")

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

	rootDirs, err := findRootDirs(g.rootDirs)
	if err != nil {
		return err
	}

	console.Printf("Finding out which files to process...\n")

	justWroteResults := false

	for _, rootDir := range rootDirs {
		rootDir, err := filepath.Abs(rootDir)
		if err != nil {
			return err
		}

		gitRepo, err := git.PlainOpen(rootDir)
		if err != nil {
			console.Printf("Skipping %s: %s\n", rootDir, err)
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
			console.Printf("%v: repository history not fully imported. run 'import git history'\n", rootDir)
			continue
		}

		importedHistory, err := g.checkImportedHistory(repo, gitRepo)
		if err != nil {
			return err
		}

		if !importedHistory {
			console.Printf("%v: repository history not fully imported. run 'import git history'\n", repo.Name)
			continue
		}

		repo.SeenAt(time.Now())

		imported, err := g.importBlame(console, storage, filesDB, peopleDB, reposDB, statsDB, repo, gitRepo, gitTree, gitCommit)
		if err != nil {
			return err
		}

		err = g.deleteBlame(console, storage, filesDB, repo, gitTree)
		if err != nil {
			return err
		}

		if g.options.SaveEvery != nil && imported > 0 {
			err = g.writeResults(console, storage, filesDB, peopleDB, reposDB, statsDB)
			if err != nil {
				return err
			}

			justWroteResults = true
		} else {
			justWroteResults = false
		}
	}

	if !justWroteResults {
		err = g.writeResults(console, storage, filesDB, peopleDB, reposDB, statsDB)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *gitBlameImporter) writeResults(console consoles.Console, storage storages.Storage, filesDB *model.Files, peopleDB *model.People, reposDB *model.Repositories, statsDB *model.MonthlyStats) error {
	console.Printf("Propagating changes to parents...\n")

	err := g.propagateChangesToParents(storage, filesDB, peopleDB, reposDB, statsDB)
	if err != nil {
		return err
	}

	console.Printf("Writing results...\n")

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

func (g *gitBlameImporter) deleteBlame(console consoles.Console, storage storages.Storage, filesDB *model.Files, repo *model.Repository, gitTree *object.Tree) error {
	toDelete, err := g.listToDelete(filesDB, repo, gitTree)
	if err != nil {
		return err
	}

	if len(toDelete) == 0 {
		return nil
	}

	console.Printf("%v: Cleaning %v deleted files...\n", repo.Name, len(toDelete))

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

func (g *gitBlameImporter) deleteFileBlame(storage storages.Storage, file *model.File) error {
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

func (g *gitBlameImporter) importBlame(console consoles.Console, storage storages.Storage, filesDB *model.Files, peopleDB *model.People, reposDB *model.Repositories, statsDB *model.MonthlyStats, repo *model.Repository, gitRepo *git.Repository, gitTree *object.Tree, gitCommit *object.Commit) (int, error) {
	toProcess, err := g.listToCompute(console, filesDB, repo, gitRepo, gitTree, gitCommit)
	if err != nil {
		return 0, err
	}

	if len(toProcess) == 0 {
		return 0, nil
	}

	console.Printf("%v: Computing blame of %v files...\n", repo.Name, len(toProcess))

	cache, err := g.newBlameCache(storage, filesDB, repo, gitRepo)
	if err != nil {
		return 0, err
	}

	bar := utils.NewProgressBar(len(toProcess))
	start := time.Now()
	for _, w := range toProcess {
		bar.Describe(utils.TruncateFilename(w.relativePath))

		err = g.computeFileBlame(storage, w, cache)
		if err != nil {
			return 0, err
		}

		if g.options.SaveEvery != nil && time.Since(start) >= *g.options.SaveEvery {
			_ = bar.Clear()

			err = g.writeResults(console, storage, filesDB, peopleDB, reposDB, statsDB)
			if err != nil {
				return 0, err
			}

			start = time.Now()
		}

		_ = bar.Add(1)
	}

	return len(toProcess), nil
}

type blameCacheImpl struct {
	storage storages.Storage
	filesDB *model.Files
	repo    *model.Repository
	gitRepo *git.Repository
	commits caches.Cache[plumbing.Hash, *BlameCommitCache]
}

func (g *gitBlameImporter) newBlameCache(storage storages.Storage, filesDB *model.Files, repo *model.Repository, gitRepo *git.Repository) (BlameCache, error) {
	cache := &blameCacheImpl{
		storage: storage,
		filesDB: filesDB,
		repo:    repo,
		gitRepo: gitRepo,
		commits: caches.NewUnlimited[plumbing.Hash, *BlameCommitCache](),
	}

	return cache, nil
}

func (c *blameCacheImpl) GetFileHash(commit *object.Commit, path string) (plumbing.Hash, error) {
	tree, err := object.GetTree(c.gitRepo.Storer, commit.TreeHash)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	e, err := tree.FindEntry(path)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	return e.Hash, nil
}

func (c *blameCacheImpl) GetCommit(hash plumbing.Hash) (*BlameCommitCache, error) {
	return c.commits.Get(hash, c.loadCommit)
}

func (c *blameCacheImpl) loadCommit(hash plumbing.Hash) (*BlameCommitCache, error) {
	result := &BlameCommitCache{
		Parents: make(map[plumbing.Hash]*BlameParentCache),
		Changes: make(map[string]*BlameFileCache),
	}

	repoCommit := c.repo.GetCommit(hash.String())

	for _, repoParentID := range repoCommit.Parents {
		repoParent := c.repo.GetCommitByID(repoParentID)

		gitParent, err := c.gitRepo.CommitObject(plumbing.NewHash(repoParent.Hash))
		if err != nil {
			return nil, err
		}

		result.Parents[gitParent.Hash] = &BlameParentCache{
			Commit:  gitParent,
			Renames: make(map[string]string),
		}
	}

	getFilename := func(fileID model.UUID) (string, error) {
		f := c.filesDB.GetFileByID(fileID)

		rel, err := filepath.Rel(c.repo.RootDir, f.Path)
		if err != nil {
			return "", err
		}

		rel = strings.ReplaceAll(rel, string(filepath.Separator), "/")
		return rel, nil
	}

	files, err := c.storage.LoadRepositoryCommitFiles(c.repo, repoCommit)
	if err != nil {
		return nil, err
	}

	for _, commitFile := range files.List() {
		filename, err := getFilename(commitFile.FileID)
		if err != nil {
			return nil, err
		}

		result.Changes[filename] = &BlameFileCache{
			Hash:    plumbing.NewHash(commitFile.Hash),
			Created: commitFile.Change == model.FileCreated,
		}

		for repoParentID, oldFileID := range commitFile.OldFileIDs {
			oldFilename, err := getFilename(oldFileID)
			if err != nil {
				return nil, err
			}

			repoParent := c.repo.GetCommitByID(repoParentID)

			parentCache := result.Parents[plumbing.NewHash(repoParent.Hash)]
			parentCache.Renames[filename] = oldFilename
		}
	}

	return result, nil
}

func (c *blameCacheImpl) GetFile(name string, hash plumbing.Hash) (*object.File, error) {
	blob, err := object.GetBlob(c.gitRepo.Storer, hash)
	if err != nil {
		return nil, err
	}

	return object.NewFile(name, filemode.Regular, blob), nil
}

func (g *gitBlameImporter) listToCompute(console consoles.Console, filesDB *model.Files, repo *model.Repository, gitRepo *git.Repository, gitTree *object.Tree, gitCommit *object.Commit) ([]*blameWork, error) {
	var result []*blameWork

	limit := math.MaxInt
	if repo.CountCommits() > 50000 {
		limit = 5000
	} else if repo.CountCommits() > 10000 {
		limit = 10000
	} else if repo.CountCommits() > 1000 {
		limit = 50000
	}

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

		lines, err := gitFile.Lines()
		if err != nil {
			return err
		}

		// Otherwise we go out of memory
		if len(lines) >= limit {
			console.Printf("%v: Skipping %v: too many lines (%v)\n", repo.Name, path, len(lines))
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
	//for i := range result {
	//	j := rand.Intn(i + 1)
	//	result[i], result[j] = result[j], result[i]
	//}

	return result, nil
}

func (g *gitBlameImporter) computeFileBlame(storage storages.Storage, w *blameWork, cache BlameCache) error {
	//now := time.Now()

	blame, err := Blame(w.gitCommit, w.relativePath, cache)
	if err != nil {
		return err
	}

	//console.Printf(" %v (in %v)\n", w.relativePath, time.Since(now))

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

func (g *gitBlameImporter) propagateChangesToParents(storage storages.Storage,
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

		s := statsDB.GetOrCreateLines(c.Date.Format("2006-01"), blame.RepositoryID, blame.AuthorID, blame.CommitterID, file.ID, file.ProjectID)

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
