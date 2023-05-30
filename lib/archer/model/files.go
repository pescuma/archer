package model

import (
	"sort"

	"github.com/samber/lo"
)

type Files struct {
	byName map[string]*File
	byID   map[UUID]*File
}

func NewFiles() *Files {
	return &Files{
		byName: map[string]*File{},
		byID:   map[UUID]*File{},
	}
}

func (fs *Files) GetOrCreate(path string) *File {
	return fs.GetOrCreateEx(path, nil)
}

func (fs *Files) GetOrCreateEx(path string, id *UUID) *File {
	if len(path) == 0 {
		panic("empty path not supported")
	}

	result, ok := fs.byName[path]

	if !ok {
		result = NewFile(path, id)
		fs.byName[path] = result
		fs.byID[result.ID] = result
	}

	return result
}

func (fs *Files) GetByID(id UUID) *File {
	return fs.byID[id]
}

func (fs *Files) List() []*File {
	result := lo.Values(fs.byName)

	sortFiles(result)

	return result
}

func (fs *Files) ListByProject(proj *Project) []*File {
	return fs.ListByProjects([]*Project{proj})
}

func (fs *Files) ListByProjects(ps []*Project) []*File {
	consider := map[UUID]bool{}
	for _, p := range ps {
		consider[p.ID] = true
	}

	result := lo.Filter(lo.Values(fs.byName), func(f *File, _ int) bool {
		return f.ProjectID != nil && consider[*f.ProjectID]
	})

	sortFiles(result)

	return result
}

func (fs *Files) GroupByDirectory() map[UUID][]*File {
	return lo.GroupBy(
		lo.Filter(fs.List(), func(f *File, _ int) bool { return f.ProjectDirectoryID != nil }),
		func(f *File) UUID { return *f.ProjectDirectoryID },
	)
}

func sortFiles(result []*File) {
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
}
