package gomod

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
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
	projsDB, err := storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := storage.LoadFiles()
	if err != nil {
		return err
	}

	fmt.Printf("Finding projects...\n")

	rootDir, err := utils.PathAbs(i.rootDir)
	if err != nil {
		return err
	}

	queue := make([]string, 0, 100)
	err = filepath.WalkDir(rootDir, func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil

		case entry.IsDir() && strings.HasPrefix(entry.Name(), "."):
			return filepath.SkipDir

		case !entry.IsDir() && entry.Name() == "go.mod":
			queue = append(queue, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("Processing projects...\n")

	bar := utils.NewProgressBar(len(queue))
	for _, path := range queue {
		path, err = utils.PathAbs(path)
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		bar.Describe(relativePath)
		_ = bar.Add(1)

		err = i.process(projsDB, filesDB, path)
		if err != nil {
			return err
		}
	}
	_ = bar.Clear()

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

	for _, req := range ast.Require {
		i.addDep(projsDB, proj, req.Mod)
	}

	for _, rep := range ast.Replace {
		i.addDep(projsDB, proj, rep.New)
	}

	dir := proj.GetDirectory(".")
	dir.Type = model.SourceDir

	filter, _ := i.createFileFilter(proj.RootDir)

	// Check deleted files and clean existing ones
	rootDir := proj.RootDir + string(filepath.Separator)
	for _, file := range filesDB.ListFiles() {
		if !strings.HasPrefix(file.Path, rootDir) {
			continue
		}
		if file.ProjectID != nil && *file.ProjectID != proj.ID {
			continue
		}

		file.ProjectID = nil
		file.ProjectDirectoryID = nil

		exists, err := utils.FileExists(file.Path)
		if err != nil {
			return err
		}
		if !exists && filter(file.Path, false) {
			file.ProjectID = &proj.ID
			file.ProjectDirectoryID = &dir.ID
			file.Exists = false
		}
	}

	err = filepath.WalkDir(proj.RootDir, func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil

		case entry.IsDir():
			return utils.IIf(filter(path, entry.IsDir()), nil, filepath.SkipDir)

		default:
			path, err := utils.PathAbs(path)
			if err != nil {
				return err
			}

			if filter(path, entry.IsDir()) {
				file := filesDB.GetOrCreateFile(path)
				file.ProjectID = &proj.ID
				file.ProjectDirectoryID = &dir.ID
			}
			return nil
		}
	})
	if err != nil {
		return err
	}

	return nil
}

func (i *gomodImporter) createFileFilter(dir string) (func(path string, isDir bool) bool, error) {
	result := func(path string, isDir bool) bool {
		name := filepath.Base(path)

		if strings.HasPrefix(name, ".") {
			return false
		}

		return isDir || name == "go.mod" || strings.HasSuffix(name, ".go")
	}

	if !i.options.RespectGitignore {
		return result, nil
	}

	matcher, err := utils.FindGitIgnore(dir)
	if err != nil {
		return nil, err
	}

	if matcher != nil {
		result = func(path string, isDir bool) bool {
			name := filepath.Base(path)

			if isDir && name == ".git" {
				return false
			}

			if isDir {
				path += string(filepath.Separator)
			}

			return !matcher(path)
		}
	}

	return result, nil
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
