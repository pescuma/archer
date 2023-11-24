package filters

import "github.com/pescuma/archer/lib/archer/model"

func CreateIgnoreFilter() Filter {
	return NewProjectFilter(Exclude, func(proj *model.Project) bool { return proj.IsIgnored() })
}

func CreateIgnoreExternalDependenciesFilter() Filter {
	return NewProjectFilter(Exclude, func(proj *model.Project) bool { return proj.IsExternalDependency() })
}

func CreateRootsFilter(roots []string) (Filter, error) {
	var fs []func(string) bool

	for _, r := range roots {
		f, err := ParseStringFilter(r)
		if err != nil {
			return nil, err
		}

		fs = append(fs, f)
	}

	ignore := func(proj *model.Project) bool {
		for _, f := range fs {
			if f(proj.Root) {
				return false
			}
		}

		return true
	}

	return NewProjectFilter(Exclude, ignore), nil
}
