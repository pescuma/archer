package common

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/utils"
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

func (r *RootsFinder) ComputeRootDirs(projs *archer.Projects) ([]RootDir, error) {
	paths := map[string]RootDir{}

	for _, rootDir := range r.rootDirs {
		switch {
		case strings.HasPrefix(rootDir, "archer:"):
			ps, err := projs.FilterProjects([]string{strings.TrimPrefix(rootDir, "archer:")}, archer.FilterExcludeExternal)
			if err != nil {
				return nil, err
			}

			for _, p := range ps {
				for _, dir := range p.Dirs {
					paths[dir] = NewRootDir(p, dir, r.globs)
				}
			}

		default:
			dir, err := utils.PathAbs(rootDir)
			if err != nil {
				return nil, err
			}

			paths[dir] = NewRootDir(nil, dir, r.globs)
		}
	}

	result := make([]RootDir, 0, len(paths))
	for _, d := range paths {
		result = append(result, d)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Dir < result[j].Dir
	})

	return result, nil
}

type RootDir struct {
	Project *archer.Project
	Dir     string
	globs   []string
}

func NewRootDir(proj *archer.Project, dir string, globs []string) RootDir {
	gs := make([]string, len(globs))
	for i, g := range globs {
		if !filepath.IsAbs(g) {
			g = filepath.Join(dir, g)
		}

		gs[i] = g
	}

	return RootDir{Project: proj, Dir: dir, globs: gs}
}

func (r *RootDir) String() string {
	return r.Dir
}

func (r *RootDir) WalkDir(cb func(proj *archer.Project, path string) error) error {
	return filepath.WalkDir(r.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		match := len(r.globs) == 0
		for _, g := range r.globs {
			m, err := doublestar.PathMatch(g, path)
			if err != nil {
				return err
			}
			if m {
				match = true
			}
		}

		if !match {
			return nil
		}

		return cb(r.Project, path)
	})

}
