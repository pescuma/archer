package gomod

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rogpeppe/go-internal/modfile"
	"golang.org/x/mod/module"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/importers/common"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
)

type Importer struct {
	console consoles.Console
	storage storages.Storage
}

type Options struct {
	Groups           []string
	RespectGitignore bool
}

func NewImporter(console consoles.Console, storage storages.Storage) *Importer {
	return &Importer{
		console: console,
		storage: storage,
	}
}

func (i *Importer) Import(dirs []string, opts *Options) error {
	projsDB, err := i.storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := i.storage.LoadFiles()
	if err != nil {
		return err
	}

	return common.FindAndImportFiles(i.console, "projects", dirs,
		func(name string) bool {
			return name == "go.mod"
		},
		func(path string) error {
			return i.process(projsDB, filesDB, path, opts)
		},
	)
}

func (i *Importer) process(projsDB *model.Projects, filesDB *model.Files, path string, opts *Options) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ast, err := modfile.ParseLax(path, data, nil)
	if err != nil {
		return err
	}

	name := ast.Module.Mod.Path
	if name == "" {
		i.console.Printf("Ignoring %v because of empty module name", path)
		return nil
	}

	proj := projsDB.GetOrCreate(name)
	proj.Groups = opts.Groups
	proj.Type = model.CodeType
	proj.RootDir = filepath.Dir(path)
	proj.ProjectFile = path
	proj.Dependencies = make(map[string]*model.ProjectDependency)
	proj.SeenAt(time.Now())

	dir := proj.GetDirectory(".")
	dir.Type = model.SourceDir
	dir.SeenAt(time.Now())

	projFile := filesDB.GetOrCreateFile(path)
	projFile.ProjectID = &proj.ID
	projFile.ProjectDirectoryID = &dir.ID
	projFile.SeenAt(time.Now())

	if projFile.RepositoryID == nil {
		proj.RepositoryID = projFile.RepositoryID
	}

	for _, req := range ast.Require {
		i.addDep(projsDB, proj, req.Mod)
	}

	for _, rep := range ast.Replace {
		i.addDep(projsDB, proj, rep.New)
	}

	filter, _ := common.CreateFileFilter(proj.RootDir, opts.RespectGitignore,
		func(path string) bool {
			name := filepath.Base(path)
			return name == "go.mod" || strings.HasSuffix(name, ".go")
		},
		nil)
	if err != nil {
		return err
	}

	err = common.MarkDeletedFilesAndUnmarkExistingOnes(filesDB, proj, dir, filter)
	if err != nil {
		return err
	}

	err = common.AddFiles(filesDB, proj, dir, filter)
	if err != nil {
		return err
	}

	return nil
}

func (i *Importer) addDep(projsDB *model.Projects, proj *model.Project, mod module.Version) {
	if mod.Path == "" {
		return
	}

	dp := projsDB.GetOrCreate(mod.Path)

	dep := proj.GetOrCreateDependency(dp)
	if mod.Version != "" {
		dep.Versions.Insert(strings.TrimPrefix(mod.Version, "v"))
	}
}
