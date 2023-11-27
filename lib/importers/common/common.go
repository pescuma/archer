package common

import (
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func FindAndImportFiles(console consoles.Console, name string, dirs []string, matcher func(string) bool, process func(string) error) error {
	console.Printf("Finding %v...\n", name)

	var queue []string

	for _, dir := range dirs {
		dir, err := utils.PathAbs(dir)
		if err != nil {
			return err
		}

		files, err := utils.ListFilesRecursive(dir, matcher)
		if err != nil {
			return err
		}

		queue = append(queue, files...)
	}

	console.Printf("Importing %v...\n", name)

	return ImportFiles(queue, func(file string) error {
		return process(file)
	})
}

func ImportFiles(queue []string, process func(string) error) error {
	bar := utils.NewProgressBar(len(queue))
	for _, file := range queue {
		bar.Describe(utils.TruncateFilename(file))

		err := process(file)
		if err != nil {
			return err
		}

		_ = bar.Add(1)
	}
	return nil
}

func CreateFileFilter(rootDir string, gitignore bool,
	defaultMatcher func(path string) bool,
	excludes func(path string) bool,
) (func(path string, isDir bool) bool, error) {
	if excludes == nil {
		excludes = func(path string) bool {
			return false
		}
	}

	result := func(path string, isDir bool) bool {
		if path == rootDir {
			return true
		}

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

	matcher, err := utils.FindGitIgnore(rootDir)
	if err != nil {
		return nil, err
	}

	if matcher != nil {
		result = func(path string, isDir bool) bool {
			if path == rootDir {
				return true
			}

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
	return filepath.WalkDir(proj.RootDir, func(path string, entry fs.DirEntry, err error) error {
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
				file.SeenAt(time.Now())
			}
			return nil
		}
	})
}
