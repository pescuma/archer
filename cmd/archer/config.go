package main

import (
	"fmt"

	"github.com/Faire/archer/lib/archer/model"
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

	filter, err := model.ParseFilter(projects, c.Project, model.Include)
	if err != nil {
		return err
	}

	for _, p := range projects.ListProjects(model.FilterExcludeExternal) {
		if filter.FilterProject(p) != model.Include {
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
