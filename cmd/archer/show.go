package main

import (
	"fmt"

	"github.com/Faire/archer/lib/archer"
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

func (c *ShowCmd) print(projects *archer.Projects, filter archer.Filter) {
	ps := projects.ListProjects(archer.FilterExcludeExternal)

	tg := groupByRoot(ps, filter, false, func(p *archer.Project) string {
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

func (c *ShowCmd) computeNodesShow(ps []*archer.Project, filter archer.Filter) map[string]bool {
	show := map[string]bool{}

	for _, p := range ps {
		show[p.Name] = false
	}

	for _, p := range ps {
		if !show[p.Name] {
			show[p.Name] = filter.Decide(filter.FilterProject(p)) != archer.Exclude
		}

		for _, d := range p.ListDependencies(archer.FilterExcludeExternal) {
			if filter.Decide(filter.FilterDependency(d)) == archer.Exclude {
				continue
			}

			show[p.Name] = true
		}
	}

	return show
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
