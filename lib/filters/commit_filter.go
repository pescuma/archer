package filters

import "github.com/pescuma/archer/lib/model"

type CommitFilterWithUsage interface {
	Filter(*model.Repository, *model.RepositoryCommit) UsageType

	// Decide does not return DontCase, so it should decide what to do in this case
	Decide(u UsageType) bool
}

type CommitFilter func(*model.Repository, *model.RepositoryCommit) bool
