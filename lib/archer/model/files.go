package model

import (
	"sort"

	"github.com/samber/lo"
)

type Files struct {
	all map[string]*File
}

func NewFiles() *Files {
	return &Files{
		all: map[string]*File{},
	}
}

func (fs *Files) Get(path string) *File {
	if len(path) == 0 {
		panic("empty path not supported")
	}

	result, ok := fs.all[path]

	if !ok {
		result = NewFile(path)
		fs.all[path] = result
	}

	return result
}

func (fs *Files) List() []*File {
	result := lo.Values(fs.all)

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

	result := lo.Filter(lo.Values(fs.all), func(f *File, _ int) bool {
		return f.ProjectID != nil && consider[*f.ProjectID]
	})

	sortFiles(result)

	return result
}

func (fs *Files) ListByProjectDirectory(dir *ProjectDirectory) []*File {
	result := lo.Filter(lo.Values(fs.all), func(f *File, _ int) bool {
		return f.ProjectDirectoryID != nil && *f.ProjectDirectoryID == dir.ID
	})

	sortFiles(result)

	return result
}

func sortFiles(result []*File) {
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
}
