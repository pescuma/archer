package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Faire/archer/lib/archer/utils"
)

type Project struct {
	Root      string
	Name      string
	NameParts []string
	Type      ProjectType
	ID        UUID

	RootDir     string
	ProjectFile string

	Dirs         map[string]*ProjectDirectory
	Dependencies map[string]*ProjectDependency
	Sizes        map[string]*Size
	Changes      *Changes
	Metrics      *Metrics
	Data         map[string]string
}

func NewProject(root, name string, id *UUID) *Project {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("p")
	} else {
		uuid = *id
	}

	return &Project{
		Root:         root,
		Name:         name,
		ID:           uuid,
		Dirs:         map[string]*ProjectDirectory{},
		Dependencies: map[string]*ProjectDependency{},
		Sizes:        map[string]*Size{},
		Changes:      NewChanges(),
		Metrics:      NewMetrics(),
		Data:         map[string]string{},
	}
}

func (p *Project) String() string {
	return fmt.Sprintf("%v:%v[%v]", p.Root, p.Name, p.Type)
}

func (p *Project) GetOrCreateDependency(d *Project) *ProjectDependency {
	result, ok := p.Dependencies[d.Name]

	if !ok {
		result = NewDependency(p, d)
		p.Dependencies[d.Name] = result
	}

	return result
}

func (p *Project) GetSize() *Size {
	result := NewSize()

	for _, v := range p.Sizes {
		result.Add(v)
	}

	return result
}

func (p *Project) GetSizeOf(name string) *Size {
	result, ok := p.Sizes[name]

	if !ok {
		result = NewSize()
	}

	return result
}

func (p *Project) AddSize(name string, size *Size) {
	old, ok := p.Sizes[name]
	if !ok {
		old = NewSize()
		p.Sizes[name] = old
	}

	old.Add(size)
}

func (p *Project) GetMetrics() *Metrics {
	result := NewMetrics()

	for _, v := range p.Dirs {
		result.Add(v.Metrics)
	}

	return result
}

func (p *Project) GetDirectory(relativePath string) *ProjectDirectory {
	result, ok := p.Dirs[relativePath]

	if !ok {
		result = NewProjectDirectory(relativePath)
		p.Dirs[relativePath] = result
	}

	return result
}

func (p *Project) FullName() string {
	return p.Root + ":" + p.Name
}

func (p *Project) SimpleName() string {
	return p.LevelSimpleName(0)
}

func (p *Project) LevelSimpleName(level int) string {
	if len(p.NameParts) == 0 {
		return p.Name
	}

	parts := p.NameParts

	if level > 0 {
		parts = utils.Take(parts, level)
	}

	parts = simplifyPrefixes(parts)

	result := strings.Join(parts, ":")

	if len(p.Name) <= len(result) {
		result = p.Name
	}

	return result
}

func simplifyPrefixes(parts []string) []string {
	for len(parts) > 1 && strings.HasPrefix(parts[1], parts[0]) {
		parts = parts[1:]
	}
	return parts
}

func (p *Project) IsIgnored() bool {
	return utils.IsTrue(p.GetData("ignore"))
}

func (p *Project) IsCode() bool {
	return p.Type == CodeType
}

func (p *Project) IsExternalDependency() bool {
	return p.Type == Library
}

func (p *Project) ListDependencies(filter FilterType) []*ProjectDependency {
	result := make([]*ProjectDependency, 0, len(p.Dependencies))

	for _, v := range p.Dependencies {
		if filter == FilterExcludeExternal && v.Target.IsExternalDependency() {
			continue
		}

		result = append(result, v)
	}

	sortDependencies(result)

	return result
}

func sortDependencies(result []*ProjectDependency) {
	sort.Slice(result, func(i, j int) bool {
		pi := result[i].Source
		pj := result[j].Source

		if pi.Name == pj.Name {
			pi = result[i].Target
			pj = result[j].Target
		}

		if pi.IsCode() && pj.IsExternalDependency() {
			return true
		}

		if pi.IsExternalDependency() && pj.IsCode() {
			return false
		}

		return strings.TrimLeft(pi.Name, ":") < strings.TrimLeft(pj.Name, ":")
	})
}

func (p *Project) SetData(name string, value string) bool {
	if p.GetData(name) == value {
		return false
	}

	if value == "" {
		delete(p.Data, name)
	} else {
		p.Data[name] = value
	}

	return true
}

func (p *Project) GetData(name string) string {
	v, _ := p.Data[name]
	return v
}
