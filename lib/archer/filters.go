package archer

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/Faire/archer/lib/archer/utils"
)

func CreateIgnoreFilter() Filter {
	return &basicFilter{
		filterProject: func(proj *Project) UsageType {
			return utils.IIf(proj.IsIgnored(), Exclude, DontCare)
		},
		filterDependency: func(dep *ProjectDependency) UsageType {
			return utils.IIf(dep.Source.IsIgnored() || dep.Target.IsIgnored(), Exclude, DontCare)
		},
		filterType: Exclude,
	}
}

func CreateRootsFilter(roots []string) (Filter, error) {
	var fs []func(string) bool

	for _, r := range roots {
		f, err := parseFilterBool(r)
		if err != nil {
			return nil, err
		}

		fs = append(fs, f)
	}

	matches := func(proj *Project) bool {
		for _, f := range fs {
			if f(proj.Root) {
				return true
			}
		}

		return false
	}

	return &basicFilter{
		filterProject: func(proj *Project) UsageType {
			return utils.IIf(matches(proj), DontCare, Exclude)
		},
		filterDependency: func(dep *ProjectDependency) UsageType {
			return utils.IIf(matches(dep.Source) && matches(dep.Target), DontCare, Exclude)
		},
		filterType: Exclude,
	}, nil
}

func ParseFilter(projs *Projects, filter string, filterType UsageType) (Filter, error) {
	switch {
	case filter == "":
		return &basicFilter{}, nil

	case strings.Index(filter, "&") >= 0:
		result := &andFilter{}

		for _, fi := range strings.Split(filter, "&") {
			fi = strings.TrimSpace(fi)
			if fi == "" {
				continue
			}

			f, err := ParseFilter(projs, fi, filterType)
			if err != nil {
				return nil, err
			}

			result.filters = append(result.filters, f)
		}

		return result, nil

	case strings.Index(filter, "->") >= 0:
		return ParseEdgeFilter(projs, filter, filterType)

	default:
		return ParseProjectFilter(projs, filter, filterType)
	}
}

func ParseProjectFilter(projs *Projects, filter string, filterType UsageType) (Filter, error) {
	projFilter, err := parseProjectFilterBool(filter)
	if err != nil {
		return nil, err
	}

	return &basicFilter{
		filterProject: func(src *Project) UsageType {
			if !projFilter(src) {
				return DontCare
			}

			return filterType
		},
		filterDependency: func(dep *ProjectDependency) UsageType {
			sm := projFilter(dep.Source)
			dm := projFilter(dep.Target)

			if filterType == Include {
				return utils.IIf(sm && dm, Include, DontCare)

			} else {
				return utils.IIf(sm || dm, Exclude, DontCare)
			}
		},
		filterType: filterType,
	}, nil
}

func ParseEdgeFilter(projs *Projects, filter string, filterType UsageType) (Filter, error) {
	re := regexp.MustCompile(`^([^>]*?)\s*(?:-(\d+)?(R)?)?->\s*([^>]*)$`)

	parts := re.FindStringSubmatch(filter)
	if parts == nil {
		return nil, errors.Errorf("invalid edge filter: %v", filter)
	}

	srcFilter, err := parseProjectFilterBool(parts[1])
	if err != nil {
		return nil, err
	}

	maxDepth := -1
	if parts[2] != "" {
		maxDepth, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, err
		}
	}

	onlyRequiredEdges := false
	if parts[3] == "R" {
		onlyRequiredEdges = true
	}

	destFilter, _ := parseProjectFilterBool(parts[4])
	if err != nil {
		return nil, err
	}

	matches := map[string]map[string]int{}

	for _, src := range projs.ListProjects(FilterExcludeExternal) {
		if !srcFilter(src) {
			continue
		}

		visited := map[string]int{}
		findMatchingEdges(matches, visited, maxDepth, []*Project{src}, destFilter)
	}

	nodes := map[string]int{}
	for s, m := range matches {
		nodes[s] = 1
		for d := range m {
			nodes[d] = 1
		}
	}

	return &basicFilter{
		filterDependency: func(dep *ProjectDependency) UsageType {
			if filterType == Include && !onlyRequiredEdges {
				return utils.IIf(utils.MapContains(nodes, dep.Source.Name) && utils.MapContains(nodes, dep.Target.Name), Include, DontCare)
			} else {
				return utils.IIf(utils.MapMapContains(matches, dep.Source.Name, dep.Target.Name), filterType, DontCare)
			}
		},
		filterType: filterType,
	}, nil
}

func findMatchingEdges(matches map[string]map[string]int, visited map[string]int, maxDepth int, path []*Project, destFilter func(proj *Project) bool) {
	add := func(a, b *Project) {
		m, ok := matches[a.Name]
		if !ok {
			m = map[string]int{}
			matches[a.Name] = m
		}

		m[b.Name] = 1
	}

	addPath := func() {
		p := path[0]
		for _, n := range path[1:] {
			add(p, n)
			p = n
		}
	}

	src := utils.First(path)
	dest := utils.Last(path)

	if dest.IsIgnored() {
		return

	} else if dest != src && destFilter(dest) {
		addPath()

	} else if maxDepth == 0 {
		return

	} else if utils.MapContains(visited, dest.Name) {
		return

	} else {
		visited[dest.Name] = 1

		for _, next := range dest.ListDependencies(FilterExcludeExternal) {
			if utils.MapMapContains(matches, dest.Name, next.Target.Name) {
				// This will lead us to a destination, so no need to go down this road
				addPath()

			} else {
				findMatchingEdges(matches, visited, maxDepth-1, append(path, next.Target), destFilter)
			}
		}
	}
}

func parseProjectFilterBool(filter string) (func(proj *Project) bool, error) {
	filter = strings.TrimSpace(filter)

	if strings.HasPrefix(filter, "!") {
		f, err := parseProjectFilterBool(filter[1:])
		if err != nil {
			return nil, err
		}

		return func(proj *Project) bool {
			return !f(proj)
		}, nil

	} else if filter == "" {
		return func(proj *Project) bool {
			return true
		}, nil

	} else if strings.HasPrefix(filter, "root:") {
		f, err := parseFilterBool(strings.TrimPrefix(filter, "root:"))
		if err != nil {
			return nil, err
		}

		return func(proj *Project) bool {
			return f(proj.Root)
		}, nil

	} else {
		f, err := parseFilterBool(filter)
		if err != nil {
			return nil, err
		}

		return func(proj *Project) bool {
			return f(proj.Name) || f(proj.SimpleName()) || f(proj.FullName())
		}, nil
	}
}

func parseFilterBool(filter string) (func(string) bool, error) {
	if strings.HasPrefix(filter, "re:") {
		re, err := regexp.Compile("(?i)" + strings.TrimPrefix(filter, "re:"))
		if err != nil {
			return nil, errors.Wrapf(err, "invalid project RE: %v", filter)
		}

		return re.MatchString, nil

	} else if strings.Index(filter, "*") >= 0 {
		filterRE := strings.ReplaceAll(filter, `\`, `\\`)
		filterRE = strings.ReplaceAll(filterRE, `.`, `\.`)
		filterRE = strings.ReplaceAll(filterRE, `(`, `\(`)
		filterRE = strings.ReplaceAll(filterRE, `)`, `\)`)
		filterRE = strings.ReplaceAll(filterRE, `^`, `\^`)
		filterRE = strings.ReplaceAll(filterRE, `$`, `\$`)
		filterRE = strings.ReplaceAll(filterRE, `*`, `.*`)

		re, err := regexp.Compile("(?i)^" + filterRE + "$")
		if err != nil {
			return nil, errors.Wrapf(err, "invalid project filter: %v", filter)
		}

		return re.MatchString, nil

	} else {
		return func(s string) bool {
			return strings.EqualFold(s, filter)
		}, nil
	}
}

func GroupFilters(filters ...Filter) Filter {
	return &multipleFilter{filters}
}

type Filter interface {
	FilterProject(proj *Project) UsageType

	FilterDependency(dep *ProjectDependency) UsageType

	// Decide does not return DontCase, so it should decide what to do in this case
	Decide(u UsageType) UsageType
}

type UsageType int

const (
	DontCare UsageType = iota
	Include
	Exclude // Exclude has preference over Include
)

func (u UsageType) Merge(other UsageType) UsageType {
	switch {
	case u == other:
		return u
	case u == Exclude || other == Exclude:
		return Exclude
	default: // One of them is Include, because they have 2 different values
		return Include
	}
}

type basicFilter struct {
	filterProject    func(proj *Project) UsageType
	filterDependency func(dep *ProjectDependency) UsageType
	filterType       UsageType
}

func (b *basicFilter) FilterProject(proj *Project) UsageType {
	if b.filterProject == nil {
		return DontCare
	}

	return b.filterProject(proj)
}

func (b *basicFilter) FilterDependency(dep *ProjectDependency) UsageType {
	if b.filterDependency == nil {
		return DontCare
	}

	return b.filterDependency(dep)
}

func (b *basicFilter) Decide(u UsageType) UsageType {
	switch {
	case u == DontCare && b.filterType == Exclude:
		return Include
	case u == DontCare && b.filterType == Include:
		return Exclude
	default:
		return u
	}
}

type andFilter struct {
	filters []Filter
}

func (m *andFilter) FilterProject(proj *Project) UsageType {
	result := lo.Map(m.filters, func(f Filter, _ int) UsageType { return f.FilterProject(proj) })
	result = lo.Uniq(result)

	if len(result) != 1 {
		return DontCare
	} else {
		return result[0]
	}
}

func (m *andFilter) FilterDependency(dep *ProjectDependency) UsageType {
	result := lo.Map(m.filters, func(f Filter, _ int) UsageType { return f.FilterDependency(dep) })
	result = lo.Uniq(result)

	if len(result) != 1 {
		return DontCare
	} else {
		return result[0]
	}
}

func (m *andFilter) Decide(u UsageType) UsageType {
	result := utils.IIf(u == DontCare, Include, u)
	for _, f := range m.filters {
		result = result.Merge(f.Decide(u))
	}
	return result
}

type multipleFilter struct {
	filters []Filter
}

func (m *multipleFilter) FilterProject(proj *Project) UsageType {
	result := DontCare
	for _, f := range m.filters {
		result = result.Merge(f.FilterProject(proj))
	}
	return result
}

func (m *multipleFilter) FilterDependency(dep *ProjectDependency) UsageType {
	result := DontCare
	for _, f := range m.filters {
		result = result.Merge(f.FilterDependency(dep))
	}
	return result
}

func (m *multipleFilter) Decide(u UsageType) UsageType {
	result := utils.IIf(u == DontCare, Include, u)
	for _, f := range m.filters {
		result = result.Merge(f.Decide(u))
	}
	return result
}
