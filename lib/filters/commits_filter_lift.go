package filters

import (
	"github.com/pescuma/archer/lib/model"
)

func LiftCommitsFilter(filter CommitsFilterBool, usage UsageType) CommitsFilter {
	return &simpleCommitsFilter{filter, usage}
}

type simpleCommitsFilter struct {
	filter CommitsFilterBool
	usage  UsageType
}

func (f *simpleCommitsFilter) Filter(repo *model.Repository, commit *model.RepositoryCommit) UsageType {
	if f.filter(repo, commit) {
		return f.usage
	} else {
		return DontCare
	}
}

func (f *simpleCommitsFilter) Decide(u UsageType) bool {
	switch {
	case u == Include:
		return true
	case u == Exclude:
		return false
	case u == DontCare && f.usage == Exclude:
		return true
	case u == DontCare && f.usage == Include:
		return false
	default:
		return true
	}
}
