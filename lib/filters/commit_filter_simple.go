package filters

import (
	"github.com/pescuma/archer/lib/model"
)

type simpleCommitFilterWithUsage struct {
	filter CommitFilter
	usage  UsageType
}

func (s *simpleCommitFilterWithUsage) Filter(repo *model.Repository, commit *model.RepositoryCommit) UsageType {
	if s.filter(repo, commit) {
		return s.usage
	} else {
		return DontCare
	}
}

func (s *simpleCommitFilterWithUsage) Decide(u UsageType) bool {
	return u.DecideFor(s.usage)
}

type commitFilterWithUsageGroup struct {
	filters []CommitFilterWithUsage
}

func (g *commitFilterWithUsageGroup) Filter(repo *model.Repository, commit *model.RepositoryCommit) UsageType {
	result := DontCare
	for _, f := range g.filters {
		result = result.Merge(f.Filter(repo, commit))
	}
	return result
}

func (g *commitFilterWithUsageGroup) Decide(u UsageType) bool {
	switch u {
	case Include:
		return true
	case Exclude:
		return false
	default:
		result := true
		for _, f := range g.filters {
			result = result && f.Decide(u)
		}
		return result
	}
}
