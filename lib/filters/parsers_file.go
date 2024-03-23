package filters

import (
	"fmt"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/pescuma/archer/lib/model"
)

func ParseFileFilterWithUsage(rule string, filterType UsageType) (FileFilterWithUsage, error) {
	filter, err := ParseFileFilter(rule)
	if err != nil {
		return nil, err
	}

	return LiftFileFilter(filter, filterType), nil
}

func ParseFileFilter(rule string) (FileFilter, error) {
	rule = strings.TrimSpace(rule)

	switch {
	case rule == "":
		return func(file *model.File) bool {
			return true
		}, nil

	case strings.Index(rule, "|") >= 0:
		clauses, err := ParseFileFilterList(strings.Split(rule, "|"))
		if err != nil {
			return nil, err
		}

		return func(file *model.File) bool {
			result := false
			for _, f := range clauses {
				result = result || f(file)
			}
			return result
		}, nil

	case strings.Index(rule, "&") >= 0:
		clauses, err := ParseFileFilterList(strings.Split(rule, "&"))
		if err != nil {
			return nil, err
		}

		return func(file *model.File) bool {
			result := true
			for _, f := range clauses {
				result = result && f(file)
			}
			return result
		}, nil

	case strings.HasPrefix(rule, "!"):
		f, err := ParseFileFilter(rule[1:])
		if err != nil {
			return nil, err
		}

		return func(file *model.File) bool {
			return !f(file)
		}, nil

	case strings.HasPrefix(rule, "id:"):
		id, err := model.StringToID(rule[3:])
		if err != nil {
			return nil, err
		}

		return func(file *model.File) bool {
			return file.ID == id
		}, nil

	default:
		if !doublestar.ValidatePathPattern(rule) {
			return nil, fmt.Errorf("invalid file glob: %v", rule)
		}

		return func(file *model.File) bool {
			m, err := doublestar.PathMatch(rule, file.Path)
			return err == nil && m
		}, nil
	}
}

func ParseFileFilterList(rules []string) ([]FileFilter, error) {
	result := make([]FileFilter, 0, len(rules))

	for _, rule := range rules {
		f, err := ParseFileFilter(rule)
		if err != nil {
			return nil, err
		}

		result = append(result, f)
	}

	return result, nil
}
