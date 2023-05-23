package main

import (
	"time"

	"github.com/Faire/archer/lib/archer/importers/git"
	"github.com/Faire/archer/lib/archer/importers/gradle"
	"github.com/Faire/archer/lib/archer/importers/hibernate"
	"github.com/Faire/archer/lib/archer/importers/loc"
	"github.com/Faire/archer/lib/archer/importers/metrics"
	"github.com/Faire/archer/lib/archer/importers/mysql"
)

type ImportGradleCmd struct {
	Path string `arg:"" help:"Path with root of gradle project." type:"existingpath"`
}

func (c *ImportGradleCmd) Run(ctx *context) error {
	g := gradle.NewImporter(c.Path)

	return ctx.ws.Import(g)
}

type ImportHibernateCmd struct {
	Path []string `arg:"" help:"Path with root of projects to search." type:"existingpath"`
	Glob []string `help:"Glob limiting files to be processed."`
	Root string   `default:"db" help:"Root name for the projects."`
}

func (c *ImportHibernateCmd) Run(ctx *context) error {
	g := hibernate.NewImporter(c.Path, c.Glob, c.Root)

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
	Filters []string `default:"" help:"Filters to be applied to the projects. Empty means all."`
}

func (c *ImportLOCCmd) Run(ctx *context) error {
	g := loc.NewImporter(c.Filters)

	return ctx.ws.Import(g)
}

type ImportMetricsCmd struct {
	Filters       []string `default:"" help:"Filters to be applied to the projects. Empty means all."`
	Incremental   bool     `negatable:"" help:"Don't import metrics already imported'."`
	LimitImported int      `help:"Limit the number of imported files. Can be used to incrementally import data. Counted randomly."`
}

func (c *ImportMetricsCmd) Run(ctx *context) error {
	limits := metrics.Limits{
		Incremental: c.Incremental,
	}
	if c.LimitImported != 0 {
		limits.MaxImportedFiles = &c.LimitImported
	}

	g := metrics.NewImporter(c.Filters, limits)

	return ctx.ws.Import(g)
}

type ImportGitCmd struct {
	Path          string        `arg:"" help:"Path with root of git repository." type:"existingpath"`
	Incremental   bool          `default:"true" negatable:"" help:"Don't import commits already imported'."`
	LimitImported int           `help:"Limit the number of imported commits. Can be used to incrementally import data. Counted from the latest commit."`
	LimitCommits  int           `help:"Limit the number of commits to be imported. Counted from the latest commit."`
	LimitDuration time.Duration `help:"Import commits only in this duration. Counted from current time."`
	After         time.Time     `help:"Import commits after this date (inclusive)."`
	Before        time.Time     `help:"Import commits before this date (exclusive)."`
}

func (c *ImportGitCmd) Run(ctx *context) error {
	limits := git.Limits{
		Incremental: c.Incremental,
	}

	if c.LimitImported != 0 {
		limits.MaxImportedCommits = &c.LimitImported
	}
	if c.LimitCommits != 0 {
		limits.MaxCommits = &c.LimitCommits
	}

	emptyTime := time.Time{}
	if c.After != emptyTime {
		limits.After = &c.After
	}
	if c.Before != emptyTime {
		limits.Before = &c.Before
	}

	if c.LimitDuration != 0 {
		before := time.Now()
		after := before.Add(-c.LimitDuration)

		if limits.After == nil || limits.After.Before(after) {
			limits.After = &after
		}
		if limits.Before == nil || limits.Before.After(before) {
			limits.Before = &before
		}
	}

	g := git.NewImporter(c.Path, limits)

	return ctx.ws.Import(g)
}
