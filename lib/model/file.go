package model

import (
	"strings"
)

type File struct {
	Path string
	ID   ID

	ProjectID          *ID
	ProjectDirectoryID *ID

	RepositoryID *ID

	ProductAreaID *ID

	Exists  bool
	Size    *Size
	Changes *Changes
	Metrics *Metrics
	SeenAt  *SeenAt
	Data    map[string]string

	Classes   map[string]*Class
	Functions map[string]*Function

	Ignore bool
}

func NewFile(path string, id ID) *File {
	if strings.Contains(path, "node_modules") {
		println("a")
	}
	return &File{
		Path:    path,
		ID:      id,
		Exists:  true,
		Size:    NewSize(),
		Changes: NewChanges(),
		Metrics: NewMetrics(),
		SeenAt:  NewSeenAt(),
		Data:    map[string]string{},
	}
}

func (f *File) GetOrCreateClass(pkg string, name string) *Class {
	var fullName string
	if pkg != "" {
		fullName = pkg + "." + name
	} else {
		fullName = name
	}

	result, ok := f.Classes[fullName]

	if !ok {
		result = NewClass(pkg, name, nil)
		f.Classes[fullName] = result
	}

	return result
}

func (f *File) GetOrCreateFunction(name string, args []string) *Function {
	fullName := name + "(" + strings.Join(args, ",") + ")"
	result, ok := f.Functions[fullName]

	if !ok {
		result = NewFunction(name, args, nil)
		f.Functions[fullName] = result
	}

	return result
}
