package filters

import (
	"strings"

	"github.com/pescuma/archer/lib/model"
)

func ParseCommitFilterWithUsage(rule string, filterType UsageType) (CommitFilterWithUsage, error) {
	filter, err := ParseCommitFilter(rule)
	if err != nil {
		return nil, err
	}

	return LiftCommitFilter(filter, filterType), nil
}

func ParseCommitFilter(rule string) (CommitFilter, error) {
	rule = strings.TrimSpace(rule)

	switch {
	case rule == "":
		return func(repository *model.Repository, commit *model.RepositoryCommit) bool {
			return true
		}, nil

	case strings.Index(rule, "|") >= 0:
		clauses, err := ParseCommitFilterList(strings.Split(rule, "|"))
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
		clauses, err := ParseCommitFilterList(strings.Split(rule, "&"))
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
		f, err := ParseCommitFilter(rule[1:])
		if err != nil {
			return nil, err
		}

		return func(repo *model.Repository, commit *model.RepositoryCommit) bool {
			return !f(repo, commit)
		}, nil

	case strings.HasPrefix(rule, "id:"):
		id, err := model.StringToID(rule[3:])
		if err != nil {
			return nil, err
		}

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

func ParseCommitFilterList(rules []string) ([]CommitFilter, error) {
	result := make([]CommitFilter, 0, len(rules))

	for _, rule := range rules {
		f, err := ParseCommitFilter(rule)
		if err != nil {
			return nil, err
		}

		result = append(result, f)
	}

	return result, nil
}
