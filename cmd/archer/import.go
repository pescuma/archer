package main

import (
	"time"

	"github.com/Faire/archer/lib/archer/importers/git"
	"github.com/Faire/archer/lib/archer/importers/gradle"
	"github.com/Faire/archer/lib/archer/importers/hibernate"
	"github.com/Faire/archer/lib/archer/importers/loc"
	"github.com/Faire/archer/lib/archer/importers/metrics"
	"github.com/Faire/archer/lib/archer/importers/mysql"
	"github.com/Faire/archer/lib/archer/importers/owners"
)

type ImportGradleCmd struct {
	Path string `arg:"" help:"Path with root of gradle project." type:"existingpath"`
}

func (c *ImportGradleCmd) Run(ctx *context) error {
	g := gradle.NewImporter(c.Path)

	return ctx.ws.Import(g)
}

type ImportHibernateCmd struct {
	Path        []string `arg:"" help:"Path with root of projects to search." type:"existingpath"`
	Glob        []string `help:"Glob limiting files to be processed."`
	Root        string   `default:"db" help:"Root name for the projects."`
	Incremental bool     `default:"true" negatable:"" help:"Don't import files already imported."`
}

func (c *ImportHibernateCmd) Run(ctx *context) error {
	g := hibernate.NewImporter(c.Path, c.Glob, c.Root, hibernate.Options{
		Incremental: c.Incremental,
	})

	return ctx.ws.Import(g)
}

type ImportMySqlCmd struct {
	ConnectionString string `arg:"" help:"MySQL connection string."`
}

func (c *ImportMySqlCmd) Run(ctx *context) error {
	g := mysql.NewImporter(c.ConnectionString)

	return ctx.ws.Import(g)
}

type ImportLOCCmd struct {
	Filters     []string `default:"" help:"Filters to be applied to the projects. Empty means all."`
	Incremental bool     `default:"true" negatable:"" help:"Don't import files already imported."`
}

func (c *ImportLOCCmd) Run(ctx *context) error {
	g := loc.NewImporter(c.Filters, loc.Options{
		Incremental: c.Incremental,
	})

	return ctx.ws.Import(g)
}

type ImportMetricsCmd struct {
	Filters       []string `default:"" help:"Filters to be applied to the projects. Empty means all."`
	Incremental   bool     `default:"true" negatable:"" help:"Don't import files already imported."`
	LimitImported int      `help:"Limit the number of imported files. Can be used to incrementally import data."`
	SaveEvery     int      `help:"Save results after some number of files."`
}

func (c *ImportMetricsCmd) Run(ctx *context) error {
	options := metrics.Options{
		Incremental: c.Incremental,
	}
	if c.LimitImported != 0 {
		options.MaxImportedFiles = &c.LimitImported
	}
	if c.SaveEvery != 0 {
		options.SaveEvery = &c.SaveEvery
	}

	g := metrics.NewImporter(c.Filters, options)

	return ctx.ws.Import(g)
}

type ImportGitCmd struct {
	Paths         []string      `arg:"" help:"Paths with the roots of git repositories." type:"existingpath"`
	Incremental   bool          `default:"true" negatable:"" help:"Don't import commits already imported."`
	LimitImported int           `help:"Limit the number of imported commits. Can be used to incrementally import data. Counted from the latest commit."`
	LimitCommits  int           `help:"Limit the number of commits to be imported. Counted from the latest commit."`
	LimitDuration time.Duration `help:"Import commits only in this duration. Counted from current time."`
	After         time.Time     `help:"Import commits after this date (inclusive)."`
	Before        time.Time     `help:"Import commits before this date (exclusive)."`
	SaveEvery     int           `help:"Save results after some number of commits."`
}

func (c *ImportGitCmd) Run(ctx *context) error {
	options := git.Options{
		Incremental: c.Incremental,
	}

	if c.LimitImported != 0 {
		options.MaxImportedCommits = &c.LimitImported
	}
	if c.LimitCommits != 0 {
		options.MaxCommits = &c.LimitCommits
	}

	if c.LimitDuration != 0 {
		before := time.Now()
		after := before.Add(-c.LimitDuration)

		if options.After == nil || options.After.Before(after) {
			options.After = &after
		}
		if options.Before == nil || options.Before.After(before) {
			options.Before = &before
		}
	}

	emptyTime := time.Time{}
	if c.After != emptyTime {
		options.After = &c.After
	}
	if c.Before != emptyTime {
		options.Before = &c.Before
	}

	if c.SaveEvery != 0 {
		options.SaveEvery = &c.SaveEvery
	}

	g := git.NewImporter(c.Paths, options)

	return ctx.ws.Import(g)
}

type ImportOwnersCmd struct {
	Filters       []string `default:"" help:"Filters to be applied to the projects. Empty means all."`
	Incremental   bool     `default:"true" negatable:"" help:"Don't import files already imported."`
	LimitImported int      `help:"Limit the number of imported files. Can be used to incrementally import data."`
	SaveEvery     int      `help:"Save results after some number of files."`
}

func (c *ImportOwnersCmd) Run(ctx *context) error {
	options := owners.Options{
		Incremental: c.Incremental,
	}
	if c.LimitImported != 0 {
		options.MaxImportedFiles = &c.LimitImported
	}
	if c.SaveEvery != 0 {
		options.SaveEvery = &c.SaveEvery
	}

	g := owners.NewImporter(c.Filters, options)

	return ctx.ws.Import(g)
}
