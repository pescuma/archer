package model

import (
	"strings"
)

type File struct {
	Path string
	ID   UUID

	ProjectID          *UUID
	ProjectDirectoryID *UUID

	RepositoryID *UUID

	ProductAreaID  *UUID
	OrganizationID *UUID
	GroupID        *UUID
	TeamID         *UUID

	Exists  bool
	Size    *Size
	Changes *Changes
	Metrics *Metrics
	Data    map[string]string

	Classes   map[string]*Class
	Functions map[string]*Function
}

func NewFile(path string, id *UUID) *File {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("f")
	} else {
		uuid = *id
	}

	return &File{
		Path:    path,
		ID:      uuid,
		Exists:  true,
		Size:    NewSize(),
		Changes: NewChanges(),
		Metrics: NewMetrics(),
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
