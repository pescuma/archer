package gomod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/importers/common"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/rogpeppe/go-internal/modfile"
	"golang.org/x/mod/module"
)

type gomodImporter struct {
	rootDir  string
	rootName string
	options  Options
}

type Options struct {
	RespectGitignore bool
}

func NewImporter(rootDir string, rootName string, options Options) archer.Importer {
	return &gomodImporter{
		rootDir:  rootDir,
		rootName: rootName,
		options:  options,
	}
}

func (i *gomodImporter) Import(storage archer.Storage) error {
	fmt.Printf("Loading existing data...\n")

	projsDB, err := storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := storage.LoadFiles()
	if err != nil {
		return err
	}

	err = common.FindAndImportFiles("projects", i.rootDir,
		func(name string) bool {
			return name == "go.mod"
		},
		func(path string) error {
			return i.process(projsDB, filesDB, path)
		})
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

	err = storage.WriteProjects(projsDB, archer.ChangedBasicInfo|archer.ChangedDependencies)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB, archer.ChangedBasicInfo)
	if err != nil {
		return err
	}

	return nil
}

func (i *gomodImporter) process(projsDB *model.Projects, filesDB *model.Files, path string) error {
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
		fmt.Printf("Ignoring %v because of empty module name", path)
		return nil
	}

	proj := projsDB.GetOrCreate(i.rootName, name)
	proj.Type = model.CodeType
	proj.RootDir = filepath.Dir(path)
	proj.ProjectFile = path
	proj.Dependencies = make(map[string]*model.ProjectDependency)

	dir := proj.GetDirectory(".")
	dir.Type = model.SourceDir

	projFile := filesDB.GetOrCreateFile(path)
	projFile.ProjectID = &proj.ID
	projFile.ProjectDirectoryID = &dir.ID

	proj.RepositoryID = projFile.RepositoryID

	for _, req := range ast.Require {
		i.addDep(projsDB, proj, req.Mod)
	}

	for _, rep := range ast.Replace {
		i.addDep(projsDB, proj, rep.New)
	}

	filter, _ := common.CreateFileFilter(proj.RootDir, i.options.RespectGitignore,
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

func (i *gomodImporter) addDep(projsDB *model.Projects, proj *model.Project, mod module.Version) {
	if mod.Path == "" {
		return
	}

	dp := projsDB.GetOrCreate(i.rootName, mod.Path)

	dep := proj.GetOrCreateDependency(dp)
	if mod.Version != "" {
		dep.Versions.Insert(strings.TrimPrefix(mod.Version, "v"))
	}
}
