package main

import (
	"github.com/pescuma/archer/lib/archer/model"
)

type cmdWithFilters struct {
	Root    []string `short:"r" help:"Only show projects from this root(s)."`
	Include []string `short:"i" help:"Filter which projects or dependencies are shown."`
	Exclude []string `short:"e" help:"Filter which projects or dependencies are NOT shown. This has preference over the included ones."`
}

func (c *cmdWithFilters) createFilter(projs *model.Projects) (model.Filter, error) {
	var filters []model.Filter

	for _, f := range c.Include {
		fi, err := model.ParseFilter(projs, f, model.Include)
		if err != nil {
			return nil, err
		}

		filters = append(filters, fi)
	}

	for _, f := range c.Exclude {
		fi, err := model.ParseFilter(projs, f, model.Exclude)
		if err != nil {
			return nil, err
		}

		filters = append(filters, fi)
	}

	filters = append(filters, model.CreateIgnoreFilter())

	if len(c.Root) > 0 {
		f, err := model.CreateRootsFilter(c.Root)
		if err != nil {
			return nil, err
		}

		filters = append(filters, f)
	}

	return model.GroupFilters(filters...), nil
}
