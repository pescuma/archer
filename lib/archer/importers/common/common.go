package common

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
)

func FindAndImportFiles(name string, rootDir string, matcher func(string) bool, process func(string) error) error {
	fmt.Printf("Finding %v...\n", name)

	rootDir, err := utils.PathAbs(rootDir)
	if err != nil {
		return err
	}

	queue, err := utils.ListFilesRecursive(rootDir, matcher)
	if err != nil {
		return err
	}

	fmt.Printf("Importing %v...\n", name)

	return ImportFiles(rootDir, queue, func(path string) error {
		return process(path)
	})
}

func ImportFiles(rootDir string, files []string, process func(string) error) error {
	bar := utils.NewProgressBar(len(files))
	for _, file := range files {
		relativePath, err := filepath.Rel(rootDir, file)
		if err != nil {
			return err
		}

		bar.Describe(relativePath)

		err = process(file)
		if err != nil {
			return err
		}

		_ = bar.Add(1)
	}
	_ = bar.Clear()
	return nil
}

func CreateFileFilter(dir string, gitignore bool,
	defaultMatcher func(path string) bool,
	excludes func(path string) bool,
) (func(path string, isDir bool) bool, error) {
	if excludes == nil {
		excludes = func(path string) bool {
			return false
		}
	}

	result := func(path string, isDir bool) bool {
		if excludes(path) {
			return false
		}

		name := filepath.Base(path)

		if strings.HasPrefix(name, ".") {
			return false
		}

		return isDir || defaultMatcher(path)
	}

	if !gitignore {
		return result, nil
	}

	matcher, err := utils.FindGitIgnore(dir)
	if err != nil {
		return nil, err
	}

	if matcher != nil {
		result = func(path string, isDir bool) bool {
			if excludes(path) {
				return false
			}

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

func MarkDeletedFilesAndUnmarkExistingOnes(filesDB *model.Files, proj *model.Project, dir *model.ProjectDirectory, filter func(path string, isDir bool) bool) error {
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

	return nil
}

func AddFiles(filesDB *model.Files, proj *model.Project, dir *model.ProjectDirectory, filter func(path string, isDir bool) bool) error {
	err := filepath.WalkDir(proj.RootDir, func(path string, entry fs.DirEntry, err error) error {
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
