package git

import (
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"v.io/x/lib/toposort"

	"github.com/pescuma/archer/lib/caches"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
)

type BlameCache interface {
	GetFile(name string, hash plumbing.Hash) (*object.File, error)
	GetCommit(hash plumbing.Hash) (*BlameCommitCache, error)
	GetFileHash(commit *object.Commit, path string) (plumbing.Hash, error)
	CommitCount() int
}

type BlameCommitCache struct {
	Hash    plumbing.Hash
	Order   int
	Parents map[plumbing.Hash]*BlameParentCache
	Changes map[string]*BlameFileCache
}

func (c *BlameCommitCache) Touched(file string) bool {
	_, ok := c.Changes[file]
	return ok
}

type BlameFileCache struct {
	Hash    plumbing.Hash
	Created bool
}

type BlameParentCache struct {
	Commit     *object.Commit
	Renames    map[string]string
	FileHashes map[string]plumbing.Hash
}

func (c *BlameParentCache) FileName(childFileName string) string {
	result, ok := c.Renames[childFileName]
	if !ok {
		result = childFileName
	}
	return result
}

func (c *BlameParentCache) FileHash(childFileName string, childHash plumbing.Hash) plumbing.Hash {
	result, ok := c.FileHashes[childFileName]
	if !ok {
		result = childHash
	}
	return result
}

type blameCacheImpl struct {
	storage storages.Storage
	filesDB *model.Files
	repo    *model.Repository
	gitRepo *git.Repository
	commits caches.Cache[plumbing.Hash, *BlameCommitCache]
	indexes map[model.UUID]int
}

func (c *blameCacheImpl) CommitCount() int {
	return c.repo.CountCommits()
}

func newBlameCache(storage storages.Storage, filesDB *model.Files, repo *model.Repository, gitRepo *git.Repository) BlameCache {
	graph := toposort.Sorter{}
	for _, c := range repo.ListCommits() {
		graph.AddNode(c.Hash)
		for _, p := range c.Parents {
			graph.AddEdge(c.Hash, repo.GetCommitByID(p).Hash)
		}
	}

	sorted, _ := graph.Sort()
	indexes := make(map[model.UUID]int, len(sorted))
	for i, s := range sorted {
		indexes[s.(model.UUID)] = i
	}

	return &blameCacheImpl{
		storage: storage,
		filesDB: filesDB,
		repo:    repo,
		gitRepo: gitRepo,
		commits: caches.NewUnlimited[plumbing.Hash, *BlameCommitCache](),
		indexes: indexes,
	}
}

func (c *blameCacheImpl) GetFileHash(commit *object.Commit, path string) (plumbing.Hash, error) {
	tree, err := object.GetTree(c.gitRepo.Storer, commit.TreeHash)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	e, err := tree.FindEntry(path)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	return e.Hash, nil
}

func (c *blameCacheImpl) GetCommit(hash plumbing.Hash) (*BlameCommitCache, error) {
	return c.commits.Get(hash, c.loadCommit)
}

func (c *blameCacheImpl) loadCommit(hash plumbing.Hash) (*BlameCommitCache, error) {
	result := &BlameCommitCache{
		Hash:    hash,
		Parents: make(map[plumbing.Hash]*BlameParentCache),
		Changes: make(map[string]*BlameFileCache),
	}

	repoCommit := c.repo.GetCommit(hash.String())

	result.Order = c.indexes[repoCommit.ID]

	getFilename := func(fileID model.UUID) (string, error) {
		f := c.filesDB.GetFileByID(fileID)

		rel, err := filepath.Rel(c.repo.RootDir, f.Path)
		if err != nil {
			return "", err
		}

		rel = strings.ReplaceAll(rel, string(filepath.Separator), "/")
		return rel, nil
	}

	files, err := c.storage.LoadRepositoryCommitFiles(c.repo, repoCommit)
	if err != nil {
		return nil, err
	}

	filesList := files.List()

	for _, repoParentID := range repoCommit.Parents {
		repoParent := c.repo.GetCommitByID(repoParentID)

		gitParent, err := c.gitRepo.CommitObject(plumbing.NewHash(repoParent.Hash))
		if err != nil {
			return nil, err
		}

		result.Parents[gitParent.Hash] = &BlameParentCache{
			Commit:     gitParent,
			Renames:    make(map[string]string, len(filesList)),
			FileHashes: make(map[string]plumbing.Hash, len(filesList)),
		}
	}

	for _, commitFile := range filesList {
		filename, err := getFilename(commitFile.FileID)
		if err != nil {
			return nil, err
		}

		result.Changes[filename] = &BlameFileCache{
			Hash:    plumbing.NewHash(commitFile.Hash),
			Created: commitFile.Change == model.FileCreated,
		}

		for repoParentID, oldFileID := range commitFile.OldIDs {
			oldFilename, err := getFilename(oldFileID)
			if err != nil {
				return nil, err
			}

			repoParent := c.repo.GetCommitByID(repoParentID)

			parentCache := result.Parents[plumbing.NewHash(repoParent.Hash)]
			parentCache.Renames[filename] = oldFilename
		}

		for repoParentID, oldFileHash := range commitFile.OldHashes {
			repoParent := c.repo.GetCommitByID(repoParentID)

			parentCache := result.Parents[plumbing.NewHash(repoParent.Hash)]

			if oldFileHash == "-" {
				parentCache.FileHashes[filename] = plumbing.ZeroHash
			} else {
				parentCache.FileHashes[filename] = plumbing.NewHash(oldFileHash)
			}
		}
	}

	return result, nil
}

func (c *blameCacheImpl) GetFile(name string, hash plumbing.Hash) (*object.File, error) {
	blob, err := object.GetBlob(c.gitRepo.Storer, hash)
	if err != nil {
		return nil, err
	}

	return object.NewFile(name, filemode.Regular, blob), nil
}
