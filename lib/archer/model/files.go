package model

import (
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
	return lo.Values(fs.all)
}

func (fs *Files) ListByProject(proj *Project) []*File {
	return lo.Filter(lo.Values(fs.all), func(f *File, _ int) bool {
		return f.ProjectID != nil && *f.ProjectID == proj.ID
	})
}

func (fs *Files) ListByProjectDirectory(dir *ProjectDirectory) []*File {
	return lo.Filter(lo.Values(fs.all), func(f *File, _ int) bool {
		return f.ProjectDirectoryID != nil && *f.ProjectDirectoryID == dir.ID
	})
}
