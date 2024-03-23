package model

import (
	"sort"
	"sync"

	"github.com/samber/lo"
)

type Files struct {
	mutex sync.RWMutex
	maxID ID

	filesByPath map[string]*File
	filesByID   map[ID]*File
}

func NewFiles() *Files {
	return &Files{
		filesByPath: map[string]*File{},
		filesByID:   map[ID]*File{},
	}
}

func (fs *Files) AddFromStorage(file *File) *File {
	if file.ID > fs.maxID {
		fs.maxID = file.ID
	}

	fs.filesByPath[file.Path] = file
	fs.filesByID[file.ID] = file

	return file
}

func (fs *Files) GetOrCreate(path string) *File {
	if len(path) == 0 {
		panic("empty path not supported")
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	result, ok := fs.filesByPath[path]

	if !ok {
		fs.maxID++
		result = NewFile(path, fs.maxID)
		fs.filesByPath[path] = result
		fs.filesByID[result.ID] = result
	}

	return result
}

func (fs *Files) Get(path string) *File {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	return fs.filesByPath[path]
}

func (fs *Files) GetByID(id ID) *File {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	return fs.filesByID[id]
}

func (fs *Files) List() []*File {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	result := lo.Values(fs.filesByPath)

	sortFiles(result)

	return result
}

func (fs *Files) ListByProject(proj *Project) []*File {
	return fs.ListByProjects([]*Project{proj})
}

func (fs *Files) ListByProjects(ps []*Project) []*File {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	consider := map[ID]bool{}
	for _, p := range ps {
		consider[p.ID] = true
	}

	result := lo.Filter(lo.Values(fs.filesByPath), func(f *File, _ int) bool {
		return f.ProjectID != nil && consider[*f.ProjectID]
	})

	sortFiles(result)

	return result
}

func (fs *Files) GroupByDirectory() map[ID][]*File {
	return lo.GroupBy(
		lo.Filter(fs.List(), func(f *File, _ int) bool { return f.ProjectDirectoryID != nil }),
		func(f *File) ID { return *f.ProjectDirectoryID },
	)
}

func sortFiles(result []*File) {
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
}
