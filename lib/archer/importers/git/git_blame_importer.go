package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
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
	SaveEvery        *int
}

func NewBlameImporter(rootDirs []string, options BlameOptions) archer.Importer {
	return &gitBlameImporter{
		rootDirs: rootDirs,
		options:  options,
	}
}

func (g *gitBlameImporter) Import(storage archer.Storage) error {
	type work struct {
		repo         *model.Repository
		commit       *object.Commit
		file         *model.File
		relativePath string
		lastMod      string
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

	reposDB, err := storage.LoadRepositories()
	if err != nil {
		return err
	}

	fmt.Printf("Preparing...\n")

	abort := errors.New("ABORT")

	var ws []*work
	var repos []*model.Repository
	toDelete := make(map[string]*model.File)
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
			fmt.Printf("Skipping '%s': %s\n", rootDir, err)
			continue
		}

		commit, err := gr.CommitObject(head.Hash())
		if err != nil {
			fmt.Printf("Skipping '%s': %s\n", rootDir, err)
			continue
		}

		tree, err := commit.Tree()
		if err != nil {
			fmt.Printf("Skipping '%s': %s\n", rootDir, err)
			continue
		}

		repo := reposDB.GetOrCreate(rootDir)
		repo.Name = filepath.Base(rootDir)
		repo.VCS = "git"

		repos = append(repos, repo)

		// Mark to process
		err = tree.Files().ForEach(func(gitFile *object.File) error {
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

			ws = append(ws, &work{
				repo:         repo,
				commit:       commit,
				file:         file,
				lastMod:      lastMod,
				relativePath: gitFile.Name,
			})

			return nil
		})
		if err != nil && err != abort {
			return err
		}

		// Mark to delete
		for _, file := range filesDB.ListFiles() {
			if file.RepositoryID == nil || *file.RepositoryID != repo.ID {
				continue
			}

			toDelete[file.Path] = file
		}

		err = tree.Files().ForEach(func(file *object.File) error {
			path, err := utils.PathAbs(filepath.Join(rootDir, file.Name))
			if err != nil {
				return err
			}

			delete(toDelete, path)

			return nil
		})
		if err != nil {
			return err
		}
	}

	fmt.Printf("Computing blame of %v files...\n", len(ws))

	bar := utils.NewProgressBar(len(ws))

	write := func() error {
		_ = bar.Clear()
		fmt.Printf("Writing results...\n")

		err = storage.WriteFiles(filesDB, archer.ChangedData|archer.ChangedChanges)
		if err != nil {
			return err
		}

		for _, repo := range repos {
			err = storage.WriteRepository(repo, archer.ChangedBasicInfo|archer.ChangedHistory)
			if err != nil {
				return err
			}
		}

		return nil
	}

	imported := 0
	for i, w := range ws {
		bar.Describe(w.repo.Name + " " + w.relativePath)
		_ = bar.Add(1)

		blame, err := git.Blame(w.commit, w.relativePath)
		if err != nil {
			return err
		}

		contents := strings.Join(lo.Map(blame.Lines, func(item *git.Line, _ int) string { return item.Text }), "\n")
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

		imported++
		if g.options.SaveEvery != nil && imported%*g.options.SaveEvery == 0 {
			err = write()
			if err != nil {
				return err
			}
		}

		// Free memory
		ws[i] = nil
	}
	_ = bar.Clear()

	fmt.Printf("Cleaning %v deleted files...\n", len(toDelete))

	bar = utils.NewProgressBar(len(toDelete))
	for _, file := range toDelete {
		_ = bar.Add(1)

		contents, err := storage.LoadFileContents(file.ID)
		if err != nil {
			return err
		}

		if len(contents.Lines) == 0 {
			continue
		}

		contents.Lines = nil

		err = storage.WriteFileContents(contents, archer.ChangedBasicInfo|archer.ChangedChanges)
		if err != nil {
			return err
		}
	}
	_ = bar.Clear()

	err = write()
	if err != nil {
		return err
	}

	fmt.Printf("Propagating changes to parents...\n")

	err = g.propagateChangesToParents(storage, peopleDB, repos)
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

	for _, repo := range repos {
		err = storage.WriteRepository(repo, archer.ChangedHistory)
		if err != nil {
			return err
		}
	}

	err = storage.WritePeople(peopleDB, archer.ChangedChanges)
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
