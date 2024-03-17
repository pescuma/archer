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
