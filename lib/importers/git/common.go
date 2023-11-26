package git

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-set/v2"

	"github.com/pescuma/archer/lib/utils"
)

func findRootDirs(baseDirs []string) ([]string, error) {
	found := set.New[string](100)

	for _, baseDir := range baseDirs {
		baseDir, err := utils.PathAbs(baseDir)
		if err != nil {
			return nil, err
		}

		err = filepath.WalkDir(baseDir, func(path string, entry fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return nil

			case entry.IsDir() && entry.Name() == ".git":
				rootDir, err := utils.PathAbs(filepath.Dir(path))
				if err != nil {
					return err
				}

				found.Insert(rootDir)
				return filepath.SkipDir

			case entry.IsDir() && strings.HasPrefix(entry.Name(), "."):
				return filepath.SkipDir
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	result := found.Slice()
	sort.Strings(result)
	return result, nil
}
