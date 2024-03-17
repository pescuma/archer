package main

import (
	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/model"
)

type cmdWithFilters struct {
	Include []string `short:"i" help:"Filter which projects or dependencies are shown."`
	Exclude []string `short:"e" help:"Filter which projects or dependencies are NOT shown. This has preference over the included ones."`
}

func (c *cmdWithFilters) createFilter(projs *model.Projects) (filters.ProjsAndDepsFilter, error) {
	var fs []filters.ProjsAndDepsFilter

	for _, f := range c.Include {
		fi, err := filters.ParseProjsAndDepsFilter(projs, f, filters.Include)
		if err != nil {
			return nil, err
		}

		fs = append(fs, fi)
	}

	for _, f := range c.Exclude {
		fi, err := filters.ParseProjsAndDepsFilter(projs, f, filters.Exclude)
		if err != nil {
			return nil, err
		}

		fs = append(fs, fi)
	}

	fs = append(fs, filters.CreateProjsAndDepsIgnoreFilter())
	fs = append(fs, filters.CreateProjsAndDepsIgnoreExternalDependenciesFilter())

	return filters.GroupProjsAnDepsFilters(fs...), nil
}
