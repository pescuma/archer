package filters

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/pescuma/archer/lib/model"
)

func ParseCommitsFilter(rule string, filterType UsageType) (CommitsFilter, error) {
	filter, err := ParseCommitsFilterBool(rule)
	if err != nil {
		return nil, err
	}

	return LiftCommitsFilter(filter, filterType), nil
}

func ParseCommitsFilterBool(rule string) (CommitsFilterBool, error) {
	rule = strings.TrimSpace(rule)

	switch {
	case rule == "":
		return func(repository *model.Repository, commit *model.RepositoryCommit) bool {
			return true
		}, nil

	case strings.Index(rule, "|") >= 0:
		clauses, err := parseClauses(strings.Split(rule, "|"))
		if err != nil {
			return nil, err
		}

		return func(repo *model.Repository, commit *model.RepositoryCommit) bool {
			result := false
			for _, f := range clauses {
				result = result || f(repo, commit)
			}
			return result
		}, nil

	case strings.Index(rule, "&") >= 0:
		clauses, err := parseClauses(strings.Split(rule, "&"))
		if err != nil {
			return nil, err
		}

		return func(repo *model.Repository, commit *model.RepositoryCommit) bool {
			result := true
			for _, f := range clauses {
				result = result && f(repo, commit)
			}
			return result
		}, nil

	case strings.HasPrefix(rule, "!"):
		f, err := ParseCommitsFilterBool(rule[1:])
		if err != nil {
			return nil, err
		}

		return func(repo *model.Repository, commit *model.RepositoryCommit) bool {
			return !f(repo, commit)
		}, nil

	case strings.HasPrefix(rule, "id:"):
		id := model.UUID(rule[3:])

		return func(repo *model.Repository, commit *model.RepositoryCommit) bool {
			return commit.ID == id
		}, nil

	default:
		f, err := ParseStringFilter(rule)
		if err != nil {
			return nil, err
		}

		return func(repo *model.Repository, commit *model.RepositoryCommit) bool {
			return f(commit.Hash)
		}, nil
	}
}

func parseClauses(split []string) ([]CommitsFilterBool, error) {
	result := make([]CommitsFilterBool, 0, len(split))

	for _, fi := range split {
		fi = strings.TrimSpace(fi)
		if fi == "" {
			return nil, errors.New("empty clause")
		}

		f, err := ParseCommitsFilterBool(fi)
		if err != nil {
			return nil, err
		}

		result = append(result, f)
	}

	return result, nil
}
