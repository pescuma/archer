package main

import (
	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/model"
)

type cmdWithFilters struct {
	Include []string `short:"i" help:"Filter which projects or dependencies are shown."`
	Exclude []string `short:"e" help:"Filter which projects or dependencies are NOT shown. This has preference over the included ones."`
}

func (c *cmdWithFilters) createFilter(projs *model.Projects) (filters.Filter, error) {
	var fs []filters.Filter

	for _, f := range c.Include {
		fi, err := filters.ParseFilter(projs, f, filters.Include)
		if err != nil {
			return nil, err
		}

		fs = append(fs, fi)
	}

	for _, f := range c.Exclude {
		fi, err := filters.ParseFilter(projs, f, filters.Exclude)
		if err != nil {
			return nil, err
		}

		fs = append(fs, fi)
	}

	fs = append(fs, filters.CreateIgnoreFilter())
	fs = append(fs, filters.CreateIgnoreExternalDependenciesFilter())

	return filters.GroupFilters(fs...), nil
}
