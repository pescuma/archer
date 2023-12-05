package filters

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func ParseFilter(projs *model.Projects, filter string, filterType UsageType) (Filter, error) {
	switch {
	case filter == "":
		return &basicFilter{}, nil

	case strings.Index(filter, "|") >= 0:
		result := &orFilter{}

		for _, fi := range strings.Split(filter, "|") {
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

func ParseProjectFilter(projs *model.Projects, filter string, filterType UsageType) (Filter, error) {
	projFilter, err := parseProjectFilterBool(filter)
	if err != nil {
		return nil, err
	}

	return NewProjectFilter(filterType, projFilter), nil
}

func ParseEdgeFilter(projs *model.Projects, filter string, filterType UsageType) (Filter, error) {
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

	for _, src := range projs.ListProjects(model.FilterExcludeExternal) {
		if !srcFilter(src) {
			continue
		}

		visited := map[string]int{}
		findMatchingEdges(matches, visited, maxDepth, []*model.Project{src}, destFilter)
	}

	nodes := map[string]int{}
	for s, m := range matches {
		nodes[s] = 1
		for d := range m {
			nodes[d] = 1
		}
	}

	return &basicFilter{
		filterDependency: func(dep *model.ProjectDependency) UsageType {
			if filterType == Include && !onlyRequiredEdges {
				return utils.IIf(utils.MapContains(nodes, dep.Source.Name) && utils.MapContains(nodes, dep.Target.Name), Include, DontCare)
			} else {
				return utils.IIf(utils.MapMapContains(matches, dep.Source.Name, dep.Target.Name), filterType, DontCare)
			}
		},
		filterType: filterType,
	}, nil
}

func findMatchingEdges(matches map[string]map[string]int, visited map[string]int, maxDepth int, path []*model.Project, destFilter func(proj *model.Project) bool) {
	add := func(a, b *model.Project) {
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

	if dest.Ignore {
		return

	} else if dest != src && destFilter(dest) {
		addPath()

	} else if maxDepth == 0 {
		return

	} else if utils.MapContains(visited, dest.Name) {
		return

	} else {
		visited[dest.Name] = 1

		for _, next := range dest.ListDependencies(model.FilterExcludeExternal) {
			if utils.MapMapContains(matches, dest.Name, next.Target.Name) {
				// This will lead us to a destination, so no need to go down this road
				addPath()

			} else {
				findMatchingEdges(matches, visited, maxDepth-1, append(path, next.Target), destFilter)
			}
		}
	}
}

func parseProjectFilterBool(filter string) (func(proj *model.Project) bool, error) {
	filter = strings.TrimSpace(filter)

	if strings.HasPrefix(filter, "!") {
		f, err := parseProjectFilterBool(filter[1:])
		if err != nil {
			return nil, err
		}

		return func(proj *model.Project) bool {
			return !f(proj)
		}, nil

	} else {
		f, err := ParseStringFilter(filter)
		if err != nil {
			return nil, err
		}

		return func(proj *model.Project) bool {
			return f(proj.Name) || f(proj.SimpleName()) || f(proj.Name)
		}, nil
	}
}

func ParseStringFilter(filter string) (func(string) bool, error) {
	filter = strings.TrimSpace(filter)

	if filter == "" {
		return func(s string) bool {
			return true
		}, nil

	} else if strings.HasPrefix(filter, "re:") {
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
