package filters

import (
	"github.com/pescuma/archer/lib/model"
)

func LiftCommitFilter(filter CommitFilter, usage UsageType) CommitFilterWithUsage {
	return &simpleCommitFilterWithUsage{filter, usage}
}

func UnliftCommitFilter(filter CommitFilterWithUsage) CommitFilter {
	return func(repo *model.Repository, commit *model.RepositoryCommit) bool {
		return filter.Decide(filter.Filter(repo, commit))
	}
}

func GroupCommitFilters(filters ...CommitFilterWithUsage) CommitFilterWithUsage {
	return &commitFilterWithUsageGroup{filters}
}
