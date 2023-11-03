package common

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
)

type RootsFinder struct {
	rootDirs []string
	globs    []string
}

func NewRootsFinder(rootDirs, globs []string) RootsFinder {
	return RootsFinder{
		rootDirs: rootDirs,
		globs:    globs,
	}
}

func (r *RootsFinder) ComputeRootDirs(projectsDB *model.Projects, filesDB *model.Files) ([]RootDir, error) {
	paths := map[string]RootDir{}

	for _, rootDir := range r.rootDirs {
		switch {
		case strings.HasPrefix(rootDir, "archer:"):
			ps, err := projectsDB.FilterProjects([]string{strings.TrimPrefix(rootDir, "archer:")}, model.FilterExcludeExternal)
			if err != nil {
				return nil, err
			}

			for _, p := range ps {
				paths[p.FullName()] = RootDir{
					filesDB: filesDB,
					Project: p,
					globs:   r.globs,
				}
			}

		default:
			dir, err := utils.PathAbs(rootDir)
			if err != nil {
				return nil, err
			}

			paths[dir] = RootDir{
				filesDB: filesDB,
				Dir:     &dir,
				globs:   r.globs,
			}
		}
	}

	result := make([]RootDir, 0, len(paths))
	for _, d := range paths {
		result = append(result, d)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result, nil
}

type RootDir struct {
	filesDB *model.Files
	Project *model.Project
	Dir     *string
	globs   []string
}

func (r *RootDir) String() string {
	if r.Dir != nil {
		return *r.Dir
	} else {
		return r.Project.FullName()
	}
}

func (r *RootDir) createGlobsMatcher(path string) func(string) (bool, error) {
	if len(r.globs) == 0 {
		return func(_ string) (bool, error) { return true, nil }
	}

	globs := make([]string, len(r.globs))
	for i, g := range r.globs {
		if !filepath.IsAbs(g) {
			g = filepath.Join(path, g)
		}

		globs[i] = g
	}

	return func(path string) (bool, error) {
		for _, g := range globs {
			m, err := doublestar.PathMatch(g, path)
			if err != nil {
				return false, err
			}
			if m {
				return true, nil
			}
		}

		return false, nil
	}
}

func (r *RootDir) WalkDir(cb func(proj *model.Project, file *model.File, path string) error) error {
	if r.Dir != nil {
		globsMatch := r.createGlobsMatcher(*r.Dir)

		return filepath.WalkDir(*r.Dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return filepath.SkipDir
			}

			if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}

			match, err := globsMatch(path)
			if err != nil {
				return err
			}

			if !match {
				return nil
			}

			file := r.filesDB.GetOrCreateFile(path)

			return cb(r.Project, file, path)
		})

	} else {
		globsMatch := r.createGlobsMatcher(r.Project.RootDir)

		for _, file := range r.filesDB.ListFilesByProject(r.Project) {
			match, err := globsMatch(file.Path)
			if err != nil {
				return err
			}

			if match {
				err = cb(r.Project, file, file.Path)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}
}
