package main

import (
	"fmt"

	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/model"
)

type ConfigSetCmd struct {
	Project string `arg:"" help:"Project filter to configure or global for all."`
	Config  string `arg:"" help:"Configuration name to change."`
	Value   string `arg:"" help:"Configuration value to set."`
}

func (c *ConfigSetCmd) Run(ctx *context) error {
	if c.Project == "global" {
		fmt.Printf("Seting '%v' = '%v'\n", c.Config, c.Value)

		_, err := ctx.ws.SetGlobalConfig(c.Config, c.Value)
		if err != nil {
			return err
		}

	} else {
		projects, err := ctx.ws.LoadProjects()
		if err != nil {
			return err
		}

		filter, err := filters.ParseProjsAndDepsFilter(projects, c.Project, filters.Include)
		if err != nil {
			return err
		}

		for _, p := range projects.ListProjects(model.FilterExcludeExternal) {
			if filter.FilterProject(p) != filters.Include {
				continue
			}

			fmt.Printf("Seting '%v' '%v' = '%v'\n", p.Name, c.Config, c.Value)

			_, err := ctx.ws.SetProjectConfig(p, c.Config, c.Value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
