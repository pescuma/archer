package model

import (
	"fmt"
)

type ProjectDependency struct {
	Source *Project
	Target *Project
	ID     UUID

	Data map[string]string
}

func NewDependency(source *Project, target *Project) *ProjectDependency {
	return &ProjectDependency{
		Source: source,
		Target: target,
		ID:     NewUUID("q"),
		Data:   map[string]string{},
	}
}

func (d *ProjectDependency) String() string {
	return fmt.Sprintf("%v -> %v", d.Source, d.Target)
}

func (d *ProjectDependency) SetData(name string, value string) bool {
	if d.GetData(name) == value {
		return false
	}

	if value == "" {
		delete(d.Data, name)
	} else {
		d.Data[name] = value
	}

	return true
}

func (d *ProjectDependency) GetData(name string) string {
	v, _ := d.Data[name]
	return v
}
