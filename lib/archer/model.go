package archer

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

	RootDir     string
	ProjectFile string

	Dirs         map[string]string
	Dependencies map[string]*Dependency
	Size         map[string]Size
	Data         map[string]string
}

func NewProject(root, name string) *Project {
	return &Project{
		Root:         root,
		Name:         name,
		Dependencies: map[string]*Dependency{},
		Size:         map[string]Size{},
		Data:         map[string]string{},
	}
}

func (p *Project) String() string {
	return fmt.Sprintf("%v:%v[%v]", p.Root, p.Name, p.Type)
}

func (p *Project) AddDependency(d *Project) *Dependency {
	result := &Dependency{
		Source: p,
		Target: d,
		Data:   map[string]string{},
	}

	p.Dependencies[d.Name] = result

	return result
}

func (p *Project) AddSize(name string, size Size) {
	p.Size[name] = size
}

func (p *Project) GetSize() Size {
	result := Size{
		Other: map[string]int{},
	}

	for _, v := range p.Size {
		result.Add(v)
	}

	return result
}

func (p *Project) GetSizeOf(name string) Size {
	result, ok := p.Size[name]

	if !ok {
		result = Size{
			Other: map[string]int{},
		}
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
	return p.Type == ExternalDependencyType
}

func (p *Project) ListDependencies(filter FilterType) []*Dependency {
	result := make([]*Dependency, 0, len(p.Dependencies))

	for _, v := range p.Dependencies {
		if filter == FilterExcludeExternal && v.Target.IsExternalDependency() {
			continue
		}

		result = append(result, v)
	}

	sortDependencies(result)

	return result
}

func sortDependencies(result []*Dependency) {
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

type Dependency struct {
	Source *Project
	Target *Project
	Data   map[string]string
}

func (d *Dependency) String() string {
	return fmt.Sprintf("%v -> %v", d.Source, d.Target)
}

func (d *Dependency) SetData(name string, value string) bool {
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

func (d *Dependency) GetData(name string) string {
	v, _ := d.Data[name]
	return v
}

type Projects struct {
	all map[string]*Project
}

func NewProjects() *Projects {
	return &Projects{
		all: map[string]*Project{},
	}
}

func (ps *Projects) GetOrNil(name string) *Project {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.all[name]
	if !ok {
		return nil
	}

	return result
}

func (ps *Projects) Get(root, name string) *Project {
	if len(root) == 0 {
		panic("empty root not supported")
	}
	if len(name) == 0 {
		panic("empty name not supported")
	}

	key := root + "\n" + name
	result, ok := ps.all[key]

	if !ok {
		result = NewProject(root, name)
		ps.all[key] = result
	}

	return result
}

func (ps *Projects) FilterProjects(filters []string, ft FilterType) ([]*Project, error) {
	if len(filters) == 0 {
		return ps.ListProjects(ft), nil
	}

	matched := map[*Project]bool{}
	for _, fe := range filters {
		filter, err := ParseFilter(ps, fe, Include)
		if err != nil {
			return nil, err
		}

		for _, p := range ps.ListProjects(ft) {
			if filter.Decide(filter.FilterProject(p)) == Include {
				matched[p] = true
			}

			for _, d := range p.ListDependencies(ft) {
				if filter.Decide(filter.FilterDependency(d)) == Include {
					matched[d.Source] = true
					matched[d.Target] = true
				}
			}
		}
	}

	result := make([]*Project, 0, len(matched))
	for p, _ := range matched {
		result = append(result, p)
	}

	sortProjects(result)

	return result, nil
}

func (ps *Projects) ListProjects(ft FilterType) []*Project {
	result := make([]*Project, 0, len(ps.all))

	for _, v := range ps.all {
		if ft == FilterExcludeExternal && v.IsExternalDependency() {
			continue
		}

		result = append(result, v)
	}

	sortProjects(result)

	return result
}

func sortProjects(result []*Project) {
	sort.Slice(result, func(i, j int) bool {
		pi := result[i]
		pj := result[j]

		if pi.IsCode() && pj.IsExternalDependency() {
			return true
		}

		if pi.IsExternalDependency() && pj.IsCode() {
			return false
		}

		return strings.TrimLeft(pi.Name, ":") < strings.TrimLeft(pj.Name, ":")
	})
}

type Size struct {
	Lines int
	Files int
	Bytes int
	Other map[string]int
}

func (l *Size) Add(other Size) {
	l.Lines += other.Lines
	l.Files += other.Files
	l.Bytes += other.Bytes

	for k, v := range other.Other {
		o, _ := l.Other[k]
		l.Other[k] = o + v
	}
}

type FilterType int

const (
	FilterAll FilterType = iota
	FilterExcludeExternal
)

type ProjectType int

const (
	ExternalDependencyType ProjectType = iota
	CodeType
	DatabaseType
)
