package archer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

	Dirs         map[string]*ProjectDirectory
	Dependencies map[string]*Dependency
	Sizes        map[string]*Size
	Data         map[string]string
}

func NewProject(root, name string) *Project {
	return &Project{
		Root:         root,
		Name:         name,
		Dirs:         map[string]*ProjectDirectory{},
		Dependencies: map[string]*Dependency{},
		Sizes:        map[string]*Size{},
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

func (p *Project) AddSize(name string, size *Size) {
	old, ok := p.Sizes[name]
	if !ok {
		old = NewSize()
		p.Sizes[name] = old
	}

	old.Add(size)
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

func (p *Project) SetDirectoryAndFiles(root string, rootType ProjectDirectoryType, recursive bool) (*ProjectDirectory, error) {
	// TODO Delete old files

	root, err := utils.PathAbs(root)
	if err != nil {
		return nil, nil
	}

	_, err = os.Stat(root)
	if err != nil {
		return nil, nil
	}

	var dir *ProjectDirectory
	if recursive {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return filepath.SkipDir
			}

			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				} else {
					return nil
				}
			}

			if dir == nil {
				rootRel, err := filepath.Rel(p.RootDir, root)
				if err != nil {
					return err
				}

				dir = p.GetDirectory(rootRel)
				dir.Type = rootType
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}

			dir.GetFile(rel)
			if err != nil {
				return err
			}

			return nil
		})
		if err == nil {
			return nil, nil
		}

	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if dir == nil {
				rootRel, err := filepath.Rel(p.RootDir, root)
				if err != nil {
					return nil, err
				}

				dir = p.GetDirectory(rootRel)
				dir.Type = rootType
			}

			dir.GetFile(entry.Name())
			if err != nil {
				return nil, err
			}
		}
	}

	return dir, nil
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
	for p := range matched {
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

func NewSize() *Size {
	return &Size{
		Other: map[string]int{},
	}
}

func (s *Size) Add(other *Size) {
	s.Lines += other.Lines
	s.Files += other.Files
	s.Bytes += other.Bytes

	for k, v := range other.Other {
		o, _ := s.Other[k]
		s.Other[k] = o + v
	}
}

func (s *Size) IsEmpty() bool {
	return s.Lines == 0 && s.Files == 0 && s.Bytes == 0 && len(s.Other) == 0
}

func (s *Size) Clear() {
	s.Lines = 0
	s.Files = 0
	s.Bytes = 0
	s.Other = map[string]int{}
}

type ProjectDirectory struct {
	RelativePath string
	Type         ProjectDirectoryType
	Files        map[string]*ProjectFile
	Size         *Size
}

func NewProjectDirectory(relativePath string) *ProjectDirectory {
	return &ProjectDirectory{
		RelativePath: relativePath,
		Size:         NewSize(),
		Files:        map[string]*ProjectFile{},
	}
}

func (d *ProjectDirectory) GetFile(relativePath string) *ProjectFile {
	result, ok := d.Files[relativePath]

	if !ok {
		result = NewProjectFile(relativePath)
		d.Files[relativePath] = result
	}

	return result
}

type ProjectFile struct {
	RelativePath string
	Size         *Size
}

func NewProjectFile(relativePath string) *ProjectFile {
	return &ProjectFile{
		RelativePath: relativePath,
		Size:         NewSize(),
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

type ProjectDirectoryType int

const (
	SourceDir ProjectDirectoryType = iota
	TestsDir
	ConfigDir
)

func (t ProjectDirectoryType) String() string {
	switch t {
	case TestsDir:
		return "tests"
	case ConfigDir:
		return "config"
	default:
		return "source"
	}
}
