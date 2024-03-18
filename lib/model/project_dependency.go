package model

import (
	"fmt"

	"github.com/hashicorp/go-set/v2"
)

type ProjectDependency struct {
	Source *Project
	Target *Project
	ID     ID

	Versions *set.Set[string]
	Data     map[string]string
}

func NewDependency(id ID, source *Project, target *Project) *ProjectDependency {
	return &ProjectDependency{
		Source:   source,
		Target:   target,
		ID:       id,
		Versions: set.New[string](10),
		Data:     map[string]string{},
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
