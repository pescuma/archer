package model

import (
	"sort"

	"github.com/samber/lo"
)

type Files struct {
	filesByName        map[string]*File
	filesByID          map[UUID]*File
}

func NewFiles() *Files {
	return &Files{
		filesByName: map[string]*File{},
		filesByID:   map[UUID]*File{},
	}
}

func (fs *Files) GetOrCreateFile(path string) *File {
	return fs.GetOrCreateFileEx(path, nil)
}

func (fs *Files) GetOrCreateFileEx(path string, id *UUID) *File {
	if len(path) == 0 {
		panic("empty path not supported")
	}

	result, ok := fs.filesByName[path]

	if !ok {
		result = NewFile(path, id)
		fs.filesByName[path] = result
		fs.filesByID[result.ID] = result
	}

	return result
}

func (fs *Files) GetFileByID(id UUID) *File {
	return fs.filesByID[id]
}

func (fs *Files) ListFiles() []*File {
	result := lo.Values(fs.filesByName)

	sortFiles(result)

	return result
}

func (fs *Files) ListFilesByProject(proj *Project) []*File {
	return fs.ListFilesByProjects([]*Project{proj})
}

func (fs *Files) ListFilesByProjects(ps []*Project) []*File {
	consider := map[UUID]bool{}
	for _, p := range ps {
		consider[p.ID] = true
	}

	result := lo.Filter(lo.Values(fs.filesByName), func(f *File, _ int) bool {
		return f.ProjectID != nil && consider[*f.ProjectID]
	})

	sortFiles(result)

	return result
}

func (fs *Files) GroupFilesByDirectory() map[UUID][]*File {
	return lo.GroupBy(
		lo.Filter(fs.ListFiles(), func(f *File, _ int) bool { return f.ProjectDirectoryID != nil }),
		func(f *File) UUID { return *f.ProjectDirectoryID },
	)
}

func sortFiles(result []*File) {
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
}

