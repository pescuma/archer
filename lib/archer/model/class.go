package model

import (
	"strings"
)

type Class struct {
	Package string
	Name    string
	ID      UUID

	Exists  bool
	Size    *Size
	Changes *Changes
	Metrics *Metrics
	Data    map[string]string

	Properties map[string]*Field
	Methods    map[string]*Function
}

func NewClass(pkg string, name string, id *UUID) *Class {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("l")
	} else {
		uuid = *id
	}

	return &Class{
		Package: pkg,
		Name:    name,
		ID:      uuid,
	}
}

func (c *Class) GetOrCreateMethod(name string, args []string) *Function {
	fullName := name + "(" + strings.Join(args, ",") + ")"
	result, ok := c.Methods[fullName]

	if !ok {
		result = NewFunction(name, args, nil)
		c.Methods[fullName] = result
	}

	return result
}

type Field struct {
	Name string
	Type string
	ID   UUID

	Exists  bool
	Size    *Size
	Changes *Changes
	Metrics *Metrics
	Data    map[string]string
}

type Function struct {
	Name string
	Args []string
	ID   UUID

	Exists  bool
	Size    *Size
	Changes *Changes
	Metrics *Metrics
	Data    map[string]string
}

func NewFunction(name string, args []string, id *UUID) *Function {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("u")
	} else {
		uuid = *id
	}

	return &Function{
		Name: name,
		Args: args,
		ID:   uuid,
	}
}
