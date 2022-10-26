package main

import "github.com/Faire/archer/lib/archer"

type cmdWithFilters struct {
	Root    []string `short:"r" help:"Only show projects from this root(s)."`
	Include []string `short:"i" help:"Filter which projects or dependencies are shown."`
	Exclude []string `short:"e" help:"Filter which projects or dependencies are NOT shown. This has preference over the included ones."`
}

func (c *cmdWithFilters) createFilter(projs *archer.Projects) (archer.Filter, error) {
	var filters []archer.Filter

	for _, f := range c.Include {
		fi, err := archer.ParseFilter(projs, f, archer.Include)
		if err != nil {
			return nil, err
		}

		filters = append(filters, fi)
	}

	for _, f := range c.Exclude {
		fi, err := archer.ParseFilter(projs, f, archer.Exclude)
		if err != nil {
			return nil, err
		}

		filters = append(filters, fi)
	}

	filters = append(filters, archer.CreateIgnoreFilter())

	if len(c.Root) > 0 {
		f, err := archer.CreateRootsFilter(c.Root)
		if err != nil {
			return nil, err
		}

		filters = append(filters, f)
	}

	return archer.GroupFilters(filters...), nil
}
