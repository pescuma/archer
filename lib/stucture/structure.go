package stucture

import (
	"fmt"
	"strings"
)

type StructureElement interface {
	FullName() string
	GetRoot() *FileStructure
	GetParent() StructureElement
	AddClass(pkg string, name string) *ClassStructure
	AddFunction(name string, params []string, result string) *FunctionStructure
}

type FileStructure struct {
	Path string

	Classes   map[string]*ClassStructure
	Functions map[string]*FunctionStructure

	AllClasses    map[string]*ClassStructure
	AllFunctions  map[string]*FunctionStructure
	AllStructures map[any]StructureElement
}

func NewFileStructure(path string) *FileStructure {
	return &FileStructure{
		Path: path,

		Classes:       map[string]*ClassStructure{},
		Functions:     map[string]*FunctionStructure{},
		AllClasses:    map[string]*ClassStructure{},
		AllFunctions:  map[string]*FunctionStructure{},
		AllStructures: map[any]StructureElement{},
	}
}

func (s *FileStructure) FullName() string {
	return s.Path
}

func (s *FileStructure) GetRoot() *FileStructure {
	return s
}

func (s *FileStructure) GetParent() StructureElement {
	return nil
}

func (s *FileStructure) AddClass(pkg string, name string) *ClassStructure {
	c := NewClassStructure(s, s, pkg, name)
	addIfNotExists(s.Classes, c)
	s.AddInternalClass(c)
	return c
}

func (s *FileStructure) AddInternalClass(c *ClassStructure) {
	addIfNotExists(s.AllClasses, c)
}

func (s *FileStructure) AddFunction(name string, params []string, result string) *FunctionStructure {
	f := NewFunctionStructure(s, s, name, params, result)
	addIfNotExists(s.Functions, f)
	s.AddInternalFunctions(f)
	return f
}

func (s *FileStructure) AddInternalFunctions(f *FunctionStructure) {
	addIfNotExists(s.AllFunctions, f)
}

func (s *FileStructure) ResolveClasses() {

}

type BaseStructure struct {
	root   *FileStructure
	parent StructureElement
}

func (s *BaseStructure) GetRoot() *FileStructure {
	return s.root
}

func (s *BaseStructure) GetParent() StructureElement {
	return s.parent
}

type ClassStructure struct {
	BaseStructure

	Package string
	Name    string

	Methods    map[string]*FunctionStructure
	Properties map[string]*FieldStructure
}

func NewClassStructure(root *FileStructure, parent StructureElement, pkg, name string) *ClassStructure {
	return &ClassStructure{
		BaseStructure: BaseStructure{
			root:   root,
			parent: parent,
		},

		Package: pkg,
		Name:    name,

		Methods:    map[string]*FunctionStructure{},
		Properties: map[string]*FieldStructure{},
	}
}

func (s *ClassStructure) FullName() string {
	if s.Package != "" {
		return s.Package + "." + s.Name
	} else {
		return s.Name
	}
}

func (s *ClassStructure) AddClass(pkg string, name string) *ClassStructure {
	c := NewClassStructure(s.root, s, pkg, name)
	s.root.AddInternalClass(c)
	return c
}

func (s *ClassStructure) AddFunction(name string, params []string, result string) *FunctionStructure {
	f := NewFunctionStructure(s.root, s, name, params, result)
	addIfNotExists(s.Methods, f)
	s.root.AddInternalFunctions(f)
	return f
}

type FunctionStructure struct {
	BaseStructure

	Name   string
	Params []string
	Result string
}

func NewFunctionStructure(root *FileStructure, parent StructureElement, name string, params []string, result string) *FunctionStructure {
	return &FunctionStructure{
		BaseStructure: BaseStructure{
			root:   root,
			parent: parent,
		},

		Name:   name,
		Params: params,
		Result: result,
	}
}

func (s *FunctionStructure) FullName() string {
	return s.BaseStructure.parent.FullName() + ":" + s.Name + " (" + strings.Join(s.Params, ", ") + ") -> " + s.Result
}

func (s *FunctionStructure) AddClass(pkg string, name string) *ClassStructure {
	c := NewClassStructure(s.root, s, pkg, name)
	s.root.AddInternalClass(c)
	return c
}

func (s *FunctionStructure) AddFunction(name string, params []string, result string) *FunctionStructure {
	f := NewFunctionStructure(s.root, s, name, params, result)
	s.root.AddInternalFunctions(f)
	return f
}

type FieldStructure struct {
	BaseStructure

	Name string
	Type string
}

func addIfNotExists[T StructureElement](m map[string]T, e T) {
	if _, ok := m[e.FullName()]; ok {
		panic(fmt.Sprintf("Already exists: %v %T", e, e))
	}

	m[e.FullName()] = e
}
