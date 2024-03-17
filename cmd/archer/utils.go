package main

import (
	"sort"

	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func groupByGroups(ps []*model.Project, filter filters.ProjectFilter, forceShowDependentProjects bool, projGrouping func(project *model.Project) string) *group {
	show := computeNodesShow(ps, filter, forceShowDependentProjects)

	ps = lo.Filter(ps, func(p *model.Project, _ int) bool { return show[p.Name] })

	gs := lo.GroupBy(ps, func(p *model.Project) string { return p.FullGroup() })

	keys := lo.Keys(gs)
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	tg := group{
		category: TotalCategory,
	}

	for _, k := range keys {
		rps := gs[k]

		rg := &group{
			category: RootCategory,
			name:     k,
			fullName: k,
		}
		tg.children = append(tg.children, rg)

		pgs := map[string]*group{}
		dgs := map[string]*group{}

		for _, p := range rps {
			pgn := projGrouping(p)

			pg, ok := pgs[pgn]
			if !ok {
				pg = &group{
					category: ProjectCategory,
					name:     pgn,
					fullName: pgn,
					proj:     p,
				}
				pgs[pgn] = pg
				rg.children = append(rg.children, pg)
			}

			size := p.Size
			pg.size.add(size)
			rg.size.add(size)
			tg.size.add(size)

			for _, d := range filters.FilterDependencies(filter, p.Dependencies) {
				dgn := projGrouping(d.Target)
				dgfn := dgn

				if pgn == dgfn {
					continue
				}

				dg, ok := dgs[pgn+"\n"+dgfn]
				if !ok {
					dg = &group{
						category: DependencyCategory,
						name:     utils.IIf(p.FullGroup() == d.Target.FullGroup(), dgn, dgfn),
						fullName: dgfn,
						dep:      d,
					}
					dgs[pgn+"\n"+dgfn] = dg
					pg.children = append(pg.children, dg)
				}

				dg.size.lines++
			}
		}
	}

	sort.Slice(tg.children, func(i, j int) bool {
		return tg.children[i].fullName < tg.children[j].fullName
	})
	for _, rg := range tg.children {
		sort.Slice(rg.children, func(i, j int) bool {
			return rg.children[i].fullName < rg.children[j].fullName
		})

		for _, pg := range rg.children {
			sort.Slice(pg.children, func(i, j int) bool {
				return pg.children[i].fullName < pg.children[j].fullName
			})
		}
	}

	return &tg
}

func computeNodesShow(ps []*model.Project, filter filters.ProjectFilter, forceShowDependentProjects bool) map[string]bool {
	show := map[string]bool{}

	for _, p := range ps {
		if filter.FilterProject(p) {
			show[p.Name] = true
		}

		for _, d := range p.ListDependencies(model.FilterExcludeExternal) {
			if !filter.FilterDependency(d) {
				continue
			}

			show[d.Source.Name] = true

			if forceShowDependentProjects {
				show[d.Target.Name] = true
			}
		}
	}

	return show
}

type group struct {
	category groupCategory
	name     string
	fullName string
	size     sizes
	children []*group

	proj *model.Project
	dep  *model.ProjectDependency
}

type groupCategory int

const (
	TotalCategory groupCategory = iota
	RootCategory
	ProjectCategory
	DependencyCategory
)
