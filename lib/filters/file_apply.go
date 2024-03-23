package filters

import (
	"github.com/pescuma/archer/lib/model"
)

func LiftFileFilter(filter FileFilter, usage UsageType) FileFilterWithUsage {
	return &simpleFileFilterWithUsage{filter, usage}
}

func UnliftFileFilter(filter FileFilterWithUsage) FileFilter {
	return func(file *model.File) bool {
		return filter.Decide(filter.Filter(file))
	}
}

func GroupFileFilters(filters ...FileFilterWithUsage) FileFilterWithUsage {
	return &fileFilterWithUsageGroup{filters}
}
