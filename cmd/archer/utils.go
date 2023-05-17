package main

import (
	"sort"

	"github.com/samber/lo"

	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

func groupByRoot(ps []*model.Project, filter model.Filter, forceShowDependentProjects bool, projGrouping func(project *model.Project) string) *group {
	show := computeNodesShow(ps, filter, forceShowDependentProjects)

	ps = lo.Filter(ps, func(p *model.Project, _ int) bool { return show[p.FullName()] })

	rs := lo.GroupBy(ps, func(p *model.Project) string { return p.Root })

	keys := lo.Keys(rs)
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	tg := group{
		category: TotalCategory,
	}

	for _, k := range keys {
		rps := rs[k]

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
			pgfn := p.Root + ":" + pgn

			pg, ok := pgs[pgfn]
			if !ok {
				pg = &group{
					category: ProjectCategory,
					name:     pgn,
					fullName: pgfn,
					proj:     p,
				}
				pgs[pgfn] = pg
				rg.children = append(rg.children, pg)
			}

			size := p.GetSize()
			pg.size.add(size)
			rg.size.add(size)
			tg.size.add(size)

			for _, d := range p.ListDependencies(model.FilterExcludeExternal) {
				if filter.Decide(filter.FilterDependency(d)) == model.Exclude {
					continue
				}

				dgn := projGrouping(d.Target)
				dgfn := d.Target.Root + ":" + dgn

				if pgfn == dgfn {
					continue
				}

				dg, ok := dgs[pgfn+"\n"+dgfn]
				if !ok {
					dg = &group{
						category: DependencyCategory,
						name:     utils.IIf(p.Root == d.Target.Root, dgn, dgfn),
						fullName: dgfn,
						dep:      d,
					}
					dgs[pgfn+"\n"+dgfn] = dg
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

func computeNodesShow(ps []*model.Project, filter model.Filter, forceShowDependentProjects bool) map[string]bool {
	show := map[string]bool{}

	for _, p := range ps {
		show[p.FullName()] = false
	}

	for _, p := range ps {
		if !show[p.FullName()] {
			show[p.FullName()] = filter.Decide(filter.FilterProject(p)) != model.Exclude
		}

		for _, d := range p.ListDependencies(model.FilterExcludeExternal) {
			if filter.Decide(filter.FilterDependency(d)) == model.Exclude {
				continue
			}

			show[d.Source.FullName()] = true

			if forceShowDependentProjects {
				show[d.Target.FullName()] = true
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
