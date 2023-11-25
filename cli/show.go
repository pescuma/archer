package main

import (
	"fmt"

	"github.com/pescuma/archer/lib/archer/filters"
	"github.com/pescuma/archer/lib/archer/model"
)

type ShowCmd struct {
	cmdWithFilters

	Levels int  `short:"l" help:"How many levels of subprojects should be considered."`
	Simple bool `short:"s" help:"Only show project names"`
}

func (c *ShowCmd) Run(ctx *context) error {
	projects, err := ctx.ws.LoadProjects()
	if err != nil {
		return err
	}

	filter, err := c.createFilter(projects)
	if err != nil {
		return err
	}

	c.print(projects, filter)

	return nil
}

func (c *ShowCmd) print(projects *model.Projects, filter filters.Filter) {
	ps := projects.ListProjects(model.FilterExcludeExternal)

	tg := groupByRoot(ps, filter, false, func(p *model.Project) string {
		return p.LevelSimpleName(c.Levels)
	})

	for _, rg := range tg.children {
		c.println("", "Root", rg.name, rg.size.text())

		for _, pg := range rg.children {
			c.println("   ", "Project", pg.name, pg.size.text())

			if !c.Simple {
				for _, dg := range pg.children {
					c.println("      ", "depends on", dg.name, "")
				}
			}
		}

		fmt.Println()
	}

	if !c.Simple && !tg.size.isEmpty() {
		c.println("", "Total", "", tg.size.text())
	}
}

func (c *ShowCmd) println(prefix, category, name, size string) {
	switch {
	case c.Simple:
		fmt.Printf("%v%v\n", prefix, name)

	case size == "":
		fmt.Printf("%v%v %v\n", prefix, category, name)

	case name == "":
		fmt.Printf("%v%v: %v\n", prefix, category, size)

	default:
		fmt.Printf("%v%v %v [%v]\n", prefix, category, name, size)
	}
}
