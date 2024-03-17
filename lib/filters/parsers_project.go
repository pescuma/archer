package filters

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func ParseProjsAndDepsFilter(projs *model.Projects, rule string, filterType UsageType) (ProjsAndDepsFilter, error) {
	switch {
	case rule == "":
		return &simpleProjectFilter{}, nil

	case strings.Index(rule, "|") >= 0:
		result := &orProjsAndDepsFilter{}

		for _, fi := range strings.Split(rule, "|") {
			fi = strings.TrimSpace(fi)
			if fi == "" {
				continue
			}

			f, err := ParseProjsAndDepsFilter(projs, fi, filterType)
			if err != nil {
				return nil, err
			}

			result.filters = append(result.filters, f)
		}

		return result, nil

	case strings.Index(rule, "&") >= 0:
		result := &andProjsAndDepsFilter{}

		for _, fi := range strings.Split(rule, "&") {
			fi = strings.TrimSpace(fi)
			if fi == "" {
				continue
			}

			f, err := ParseProjsAndDepsFilter(projs, fi, filterType)
			if err != nil {
				return nil, err
			}

			result.filters = append(result.filters, f)
		}

		return result, nil

	case strings.Index(rule, "->") >= 0:
		return ParseOnlyDepsFilter(projs, rule, filterType)

	default:
		return ParseOnlyProjsFilter(projs, rule, filterType)
	}
}

func ParseOnlyProjsFilter(projs *model.Projects, rule string, filterType UsageType) (ProjsAndDepsFilter, error) {
	projFilter, err := ParseOnlyProjsFilterBool(rule)
	if err != nil {
		return nil, err
	}

	return NewSimpleProjectFilter(filterType, projFilter), nil
}

func ParseOnlyDepsFilter(projs *model.Projects, rule string, filterType UsageType) (ProjsAndDepsFilter, error) {
	re := regexp.MustCompile(`^([^>]*?)\s*(?:-(\d+)?(R)?)?->\s*([^>]*)$`)

	parts := re.FindStringSubmatch(rule)
	if parts == nil {
		return nil, errors.Errorf("invalid edge filter: %v", rule)
	}

	srcFilter, err := ParseOnlyProjsFilterBool(parts[1])
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

	destFilter, _ := ParseOnlyProjsFilterBool(parts[4])
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

	return &simpleProjectFilter{
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

func ParseOnlyProjsFilterBool(rule string) (func(proj *model.Project) bool, error) {
	rule = strings.TrimSpace(rule)

	if strings.HasPrefix(rule, "!") {
		f, err := ParseOnlyProjsFilterBool(rule[1:])
		if err != nil {
			return nil, err
		}

		return func(proj *model.Project) bool {
			return !f(proj)
		}, nil

	} else if strings.HasPrefix(rule, "id:") {
		id := model.UUID(rule[3:])

		return func(proj *model.Project) bool {
			return proj.ID == id
		}, nil
	} else {
		f, err := ParseStringFilter(rule)
		if err != nil {
			return nil, err
		}

		return func(proj *model.Project) bool {
			return f(proj.Name) || f(proj.SimpleName()) || f(proj.Name)
		}, nil
	}
}
