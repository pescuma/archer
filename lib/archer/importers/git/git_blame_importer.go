package git

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
}

type BlameOptions struct {
	Incremental      bool
	MaxImportedFiles *int
}

func NewBlameImporter(rootDirs []string, options BlameOptions) archer.Importer {
	return &gitBlameImporter{
		rootDirs: rootDirs,
		options:  options,
	}
}

type blameWork struct {
	repo         *model.Repository
	gr           *git.Repository
	commit       *object.Commit
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

	fmt.Printf("Finding out which files to process...\n")

	var ws []*blameWork
	toDelete := make(map[string]*model.File)
	var repos []*model.Repository
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

		head, err := gr.Head()
		if err != nil {
			return err
		}

		commit, err := gr.CommitObject(head.Hash())
		if err != nil {
			return err
		}

		tree, err := commit.Tree()
		if err != nil {
			return err
		}

		repo := reposDB.Get(rootDir)
		if repo == nil {
			return fmt.Errorf("repository not found: '%v'. run 'import git history' before importing blame", rootDir)
		}

		repos = append(repos, repo)

		ws, err = g.markToProcess(ws, filesDB, rootDir, gr, repo, tree, commit)
		if err != nil {
			return err
		}

		err = g.markToDelete(toDelete, filesDB, rootDir, repo, tree)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Computing blame of %v files...\n", len(ws))

	bar := utils.NewProgressBar(len(ws))

	sort.Slice(ws, func(i, j int) bool {
		return ws[i].file.Path <= ws[j].file.Path
	})
	byRepo := lo.GroupBy(ws, func(item *blameWork) *model.Repository {
		return item.repo
	})

	for repo, ws := range byRepo {
		_ = bar.Clear()
		fmt.Printf("Creating cache for %v...\n", repo.Name)

		cache, err := g.createCache(filesDB, repo, ws[0].gr)
		if err != nil {
			return err
		}

		for _, w := range ws {
			bar.Describe(repo.Name + " " + utils.TruncateFilename(w.relativePath))

			err = g.computeBlame(storage, w, cache)
			if err != nil {
				return err
			}

			_ = bar.Add(1)
		}
	}

	fmt.Printf("Cleaning %v deleted files...\n", len(toDelete))

	bar = utils.NewProgressBar(len(toDelete))
	for _, file := range toDelete {
		err = g.deleteBlame(storage, file)
		if err != nil {
			return err
		}

		_ = bar.Add(1)
	}

	fmt.Printf("Propagating changes to parents...\n")

	err = g.propagateChangesToParents(storage, peopleDB, repos)
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

	err = storage.WriteFiles(filesDB, archer.ChangedData|archer.ChangedChanges)
	if err != nil {
		return err
	}

	err = storage.WritePeople(peopleDB, archer.ChangedChanges)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		err = storage.WriteRepository(repo, archer.ChangedHistory)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *gitBlameImporter) createCache(filesDB *model.Files, repo *model.Repository, gr *git.Repository) (*BlameCache, error) {
	cache := &BlameCache{
		Commits: make(map[plumbing.Hash]*BlameCommitCache),
	}

	for _, repoCommit := range repo.ListCommits() {
		gitCommit, err := gr.CommitObject(plumbing.NewHash(repoCommit.Hash))
		if err != nil {
			return nil, err
		}

		commitCache := &BlameCommitCache{
			Parents: make(map[plumbing.Hash]*BlameParentCache),
			Touched: set.New[string](10),
		}
		cache.Commits[gitCommit.Hash] = commitCache

		for _, repoParentID := range repoCommit.Parents {
			repoParent := repo.GetCommitByID(repoParentID)

			gitParent, err := gr.CommitObject(plumbing.NewHash(repoParent.Hash))
			if err != nil {
				return nil, err
			}

			commitCache.Parents[gitParent.Hash] = &BlameParentCache{
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

		for _, commitFile := range repoCommit.Files {
			filename, err := getFilename(commitFile.FileID)
			if err != nil {
				return nil, err
			}
			commitCache.Touched.Insert(filename)

			for repoParentID, oldFileID := range commitFile.OldFileIDs {
				oldFilename, err := getFilename(oldFileID)
				if err != nil {
					return nil, err
				}

				repoParent := repo.GetCommitByID(repoParentID)

				parentCache := commitCache.Parents[plumbing.NewHash(repoParent.Hash)]
				parentCache.Renames[filename] = oldFilename
			}
		}
	}

	return cache, nil
}

func (g *gitBlameImporter) deleteBlame(storage archer.Storage, file *model.File) error {
	contents, err := storage.LoadFileContents(file.ID)
	if err != nil {
		return err
	}

	if len(contents.Lines) == 0 {
		return nil
	}

	contents.Lines = nil

	err = storage.WriteFileContents(contents, archer.ChangedBasicInfo|archer.ChangedChanges)
	if err != nil {
		return err
	}

	return nil
}

func (g *gitBlameImporter) markToProcess(ws []*blameWork, filesDB *model.Files, rootDir string, gr *git.Repository, repo *model.Repository, tree *object.Tree, commit *object.Commit) ([]*blameWork, error) {
	abort := errors.New("ABORT")

	err := tree.Files().ForEach(func(gitFile *object.File) error {
		if !g.options.ShouldContinue(len(ws)) {
			return abort
		}

		path, err := utils.PathAbs(filepath.Join(rootDir, gitFile.Name))
		if err != nil {
			return err
		}

		file := filesDB.GetOrCreateFile(path)
		file.RepositoryID = &repo.ID

		lastMod := ""
		stat, err := os.Stat(path)
		if err == nil {
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

		ws = append(ws, &blameWork{
			repo:         repo,
			gr:           gr,
			commit:       commit,
			file:         file,
			lastMod:      lastMod,
			relativePath: gitFile.Name,
		})

		return nil
	})
	if err == abort {
		err = nil
	}

	return ws, err
}

func (g *gitBlameImporter) markToDelete(toDelete map[string]*model.File, filesDB *model.Files, rootDir string, repo *model.Repository, tree *object.Tree) error {
	for _, file := range filesDB.ListFiles() {
		if file.RepositoryID == nil || *file.RepositoryID != repo.ID {
			continue
		}

		toDelete[file.Path] = file
	}

	err := tree.Files().ForEach(func(file *object.File) error {
		path, err := utils.PathAbs(filepath.Join(rootDir, file.Name))
		if err != nil {
			return err
		}

		delete(toDelete, path)

		return nil
	})

	return err
}

func (g *gitBlameImporter) computeBlame(storage archer.Storage, w *blameWork, cache *BlameCache) error {
	blame, err := Blame(w.commit, w.relativePath, cache)
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

		commitHash := blameLine.Hash.String()

		if !w.repo.ContainsCommit(commitHash) {
			return fmt.Errorf("missing commit '%v':'%v'. run 'import git history' before importing blame", w.repo.Name, commitHash)
		}

		commit := w.repo.GetCommit(commitHash)

		fileLine.CommitID = &commit.ID
		fileLine.AuthorID = &commit.AuthorID
		fileLine.Type = lt
		fileLine.Text = blameLine.Text
	}

	fileLines.Lines = fileLines.Lines[:len(blame.Lines)]

	if w.lastMod != "" {
		w.file.Data["blame:last_modified"] = w.lastMod
	} else {
		delete(w.file.Data, "blame:last_modified")
	}

	err = storage.WriteFileContents(fileLines, archer.ChangedBasicInfo|archer.ChangedChanges)
	if err != nil {
		return err
	}

	err = storage.WriteFile(w.file, archer.ChangedChanges)
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

func (g *gitBlameImporter) propagateChangesToParents(storage archer.Storage, peopleDB *model.People, repos []*model.Repository) error {
	blames, err := storage.ComputeBlamePerAuthor()
	if err != nil {
		return err
	}

	commits := make(map[model.UUID]*model.RepositoryCommit)
	for _, r := range repos {
		for _, c := range r.ListCommits() {
			commits[c.ID] = c
			c.SurvivedLines = 0

			for _, f := range c.Files {
				f.SurvivedLines = 0
			}
		}
	}

	filesPerPerson := make(map[model.UUID]*set.Set[model.UUID])

	for _, p := range peopleDB.ListPeople() {
		p.Blame.Clear()
		filesPerPerson[p.ID] = set.New[model.UUID](10)
	}

	for _, blame := range blames {
		if c, ok := commits[blame.CommitID]; ok {
			c.SurvivedLines += blame.Lines

			// TODO Blame code is creating strange commit references
			if file, ok := c.Files[blame.FileID]; ok {
				file.SurvivedLines += blame.Lines
			}
		}

		p := peopleDB.GetPersonByID(blame.AuthorID)

		filesPerPerson[p.ID].Insert(blame.FileID)

		add := func(t string, l int) {
			p.Blame.Lines += l

			if _, ok := p.Blame.Other[t]; !ok {
				p.Blame.Other[t] = l
			} else {
				p.Blame.Other[t] += l
			}
		}

		switch blame.LineType {
		case model.CodeFileLine:
			add("Code", blame.Lines)
		case model.CommentFileLine:
			add("Comment", blame.Lines)
		case model.BlankFileLine:
			add("Blank", blame.Lines)
		default:
			panic(blame.LineType)
		}
	}

	for _, p := range peopleDB.ListPeople() {
		p.Blame.Files = filesPerPerson[p.ID].Size()
	}

	return nil
}

func (l *BlameOptions) ShouldContinue(imported int) bool {
	if l.MaxImportedFiles != nil && imported >= *l.MaxImportedFiles {
		return false
	}

	return true
}
