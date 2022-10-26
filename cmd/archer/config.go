package main

import (
	"fmt"

	"github.com/Faire/archer/lib/archer"
)

type ConfigSetCmd struct {
	Project string `arg:"" help:"Project filter to configure."`
	Config  string `arg:"" help:"Configuration name to change."`
	Value   string `arg:"" help:"Configuration value to set."`
}

func (c *ConfigSetCmd) Run(ctx *context) error {
	projects, err := ctx.ws.LoadProjects()
	if err != nil {
		return err
	}

	filter, err := archer.ParseFilter(projects, c.Project, archer.Include)
	if err != nil {
		return err
	}

	for _, p := range projects.ListProjects(archer.FilterExcludeExternal) {
		if filter.FilterProject(p) != archer.Include {
			continue
		}

		fmt.Printf("Seting '%v' '%v' = '%v'\n", p.Name, c.Config, c.Value)

		_, err := ctx.ws.SetConfigParameter(p, c.Config, c.Value)
		if err != nil {
			return err
		}
	}

	return nil
}
