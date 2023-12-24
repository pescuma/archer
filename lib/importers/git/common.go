package git

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-set/v2"

	"github.com/pescuma/archer/lib/model"
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

func log(gitRepo *git.Repository, gitRevision plumbing.Hash) (object.CommitIter, error) {
	return gitRepo.Log(&git.LogOptions{
		From:  gitRevision,
		Order: git.LogOrderCommitterTime,
	})
}

func findBranchHash(repo *model.Repository, gitRepo *git.Repository, branch string) (string, plumbing.Hash, error) {
	if branch == "" && repo != nil {
		branch = repo.Branch
	}

	if branch == "" {
		gitHead, err := gitRepo.Head()
		if err != nil {
			return "", plumbing.ZeroHash, err
		}

		return "HEAD", gitHead.Hash(), nil
	}

	for _, candidate := range strings.Split(branch, ",") {
		revision, err := gitRepo.ResolveRevision(plumbing.Revision(candidate))
		if err == nil {
			return candidate, *revision, err
		}
	}

	return "", plumbing.ZeroHash, fmt.Errorf("%v: no branch found with name: %v", repo.Name, branch)
}
