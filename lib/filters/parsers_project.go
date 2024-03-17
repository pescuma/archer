package filters

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func ParseProjectFilterWithUsage(projs *model.Projects, rule string, filterType UsageType) (ProjectFilterWithUsage, error) {
	filter, err := ParseProjectFilter(projs, rule)
	if err != nil {
		return nil, err
	}

	return LiftProjectFilter(filter, filterType), nil
}

func ParseProjectFilter(projs *model.Projects, rule string) (ProjectFilter, error) {
	switch {
	case rule == "":
		return &simpleProjectFilter{}, nil

	case strings.Index(rule, "|") >= 0:
		clauses, err := ParseProjectFilterList(projs, strings.Split(rule, "|"))
		if err != nil {
			return nil, err
		}

		return &simpleProjectFilter{
			func(proj *model.Project) bool {
				result := false
				for _, f := range clauses {
					result = result || f.FilterProject(proj)
				}
				return result
			},
			func(dep *model.ProjectDependency) bool {
				result := false
				for _, f := range clauses {
					result = result || f.FilterDependency(dep)
				}
				return result
			},
		}, nil

	case strings.Index(rule, "&") >= 0:
		clauses, err := ParseProjectFilterList(projs, strings.Split(rule, "&"))
		if err != nil {
			return nil, err
		}

		return &simpleProjectFilter{
			func(proj *model.Project) bool {
				result := true
				for _, f := range clauses {
					result = result && f.FilterProject(proj)
				}
				return result
			},
			func(dep *model.ProjectDependency) bool {
				result := true
				for _, f := range clauses {
					result = result && f.FilterDependency(dep)
				}
				return result
			},
		}, nil

	case strings.Index(rule, "->") >= 0:
		return ParseDependencyFilter(projs, rule)

	default:
		filter, err := ParseOnlyProjsFilter(rule)
		if err != nil {
			return nil, err
		}

		return &simpleProjectFilter{
			filter,
			func(dependency *model.ProjectDependency) bool {
				return filter(dependency.Source) && filter(dependency.Target)
			},
		}, nil
	}
}

func ParseProjectFilterList(projs *model.Projects, rules []string) ([]ProjectFilter, error) {
	result := make([]ProjectFilter, 0, len(rules))

	for _, rule := range rules {
		f, err := ParseProjectFilter(projs, rule)
		if err != nil {
			return nil, err
		}

		result = append(result, f)
	}

	return result, nil
}

func ParseDependencyFilter(projs *model.Projects, rule string) (ProjectFilter, error) {
	re := regexp.MustCompile(`^([^>]*?)\s*(?:-(\d+)?(G)?)?->\s*([^>]*)$`)

	parts := re.FindStringSubmatch(rule)
	if parts == nil {
		return nil, errors.Errorf("invalid edge filter: %v", rule)
	}

	srcFilter, err := ParseOnlyProjsFilter(parts[1])
	if err != nil {
		return nil, err
	}

	maxDepth := math.MaxInt
	if parts[2] != "" {
		maxDepth, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, err
		}
	}

	onlyRequiredEdges := true
	if parts[3] == "G" {
		onlyRequiredEdges = false
	}

	destFilter, _ := ParseOnlyProjsFilter(parts[4])
	if err != nil {
		return nil, err
	}

	matches := map[model.UUID]map[model.UUID]bool{}

	for _, src := range projs.ListProjects(model.FilterExcludeExternal) {
		if !srcFilter(src) {
			continue
		}

		visited := map[model.UUID]bool{}
		findMatchingEdges(matches, visited, destFilter, maxDepth, []*model.Project{src})
	}

	nodes := map[model.UUID]bool{}
	for s, m := range matches {
		nodes[s] = true
		for d := range m {
			nodes[d] = true
		}
	}

	return &simpleProjectFilter{
		func(project *model.Project) bool {
			return nodes[project.ID]
		},
		func(dep *model.ProjectDependency) bool {
			if onlyRequiredEdges {
				return utils.MapMapContains(matches, dep.Source.ID, dep.Target.ID)
			} else {
				return utils.MapContains(nodes, dep.Source.ID) && utils.MapContains(nodes, dep.Target.ID)
			}
		},
	}, nil
}

func findMatchingEdges(
	matches map[model.UUID]map[model.UUID]bool,
	visited map[model.UUID]bool,
	destFilter func(proj *model.Project) bool,
	maxDepth int,
	path []*model.Project,
) {
	dest := utils.Last(path)
	visited[dest.ID] = true

	if destFilter(dest) {
		addPath(matches, path)
	}

	if maxDepth > 0 {
		for _, dep := range dest.ListDependencies(model.FilterExcludeExternal) {
			next := dep.Target

			if visited[next.ID] {
				if utils.MapContains(matches, next.ID) {
					addPath(matches, append(path, next))
				}

			} else {
				findMatchingEdges(matches, visited, destFilter, maxDepth-1, append(path, next))
			}
		}
	}
}

func addEdge(matches map[model.UUID]map[model.UUID]bool, a, b *model.Project) {
	m, ok := matches[a.ID]
	if !ok {
		m = map[model.UUID]bool{}
		matches[a.ID] = m
	}

	m[b.ID] = true
}

func addPath(matches map[model.UUID]map[model.UUID]bool, path []*model.Project) {
	p := path[0]
	for _, n := range path[1:] {
		addEdge(matches, p, n)
		p = n
	}
}

func ParseOnlyProjsFilter(rule string) (func(proj *model.Project) bool, error) {
	rule = strings.TrimSpace(rule)

	if strings.HasPrefix(rule, "!") {
		f, err := ParseOnlyProjsFilter(rule[1:])
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
