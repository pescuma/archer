package main

import (
	"time"

	"github.com/pescuma/archer/lib/archer/importers/csproj"
	"github.com/pescuma/archer/lib/archer/importers/git"
	"github.com/pescuma/archer/lib/archer/importers/gomod"
	"github.com/pescuma/archer/lib/archer/importers/gradle"
	"github.com/pescuma/archer/lib/archer/importers/hibernate"
	"github.com/pescuma/archer/lib/archer/importers/loc"
	"github.com/pescuma/archer/lib/archer/importers/metrics"
	"github.com/pescuma/archer/lib/archer/importers/mysql"
	"github.com/pescuma/archer/lib/archer/importers/owners"
)

type ImportGradleCmd struct {
	Path string `arg:"" help:"Path with root of gradle project." type:"existingpath"`
}

func (c *ImportGradleCmd) Run(ctx *context) error {
	g := gradle.NewImporter(c.Path)

	return ctx.ws.Import(g)
}

type ImportGomodCmd struct {
	Path      string `arg:"" help:"Path to recursively search for go.mod files." type:"existingpath"`
	Root      string `default:"go" help:"Root name for the projects."`
	Gitignore bool   `default:"true" help:"Respect .gitignore file when importing files."`
}

func (c *ImportGomodCmd) Run(ctx *context) error {
	g := gomod.NewImporter(c.Path, c.Root, gomod.Options{
		RespectGitignore: c.Gitignore,
	})

	return ctx.ws.Import(g)
}

type ImportCsprojCmd struct {
	Path      string `arg:"" help:"Path to recursively search for csproj files." type:"existingpath"`
	Root      string `default:"cs" help:"Root name for the projects."`
	Gitignore bool   `default:"true" help:"Respect .gitignore file when importing files."`
}

func (c *ImportCsprojCmd) Run(ctx *context) error {
	g := csproj.NewImporter(c.Path, c.Root, csproj.Options{
		RespectGitignore: c.Gitignore,
	})

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

type ImportGitPeopleCmd struct {
	Paths []string `arg:"" help:"Paths with the roots of git repositories." type:"existingpath"`
}

func (c *ImportGitPeopleCmd) Run(ctx *context) error {
	g := git.NewPeopleImporter(c.Paths)

	return ctx.ws.Import(g)
}

type ImportGitHistoryCmd struct {
	Paths         []string      `arg:"" help:"Paths with the roots of git repositories." type:"existingpath"`
	Incremental   bool          `default:"true" negatable:"" help:"Don't import commits already imported."`
	LimitImported int           `help:"Limit the number of imported commits. Can be used to incrementally import data. Counted from the latest commit."`
	LimitCommits  int           `help:"Limit the number of commits to be imported. Counted from the latest commit."`
	LimitDuration time.Duration `help:"Import commits only in this duration. Counted from current time."`
	After         time.Time     `help:"Import commits after this date (inclusive)."`
	Before        time.Time     `help:"Import commits before this date (exclusive)."`
	SaveEvery     int           `help:"Save results after some number of commits."`
}

func (c *ImportGitHistoryCmd) Run(ctx *context) error {
	options := git.HistoryOptions{
		Incremental: c.Incremental,
	}

	if c.LimitImported != 0 {
		options.MaxImportedCommits = &c.LimitImported
	}
	if c.LimitCommits != 0 {
		options.MaxCommits = &c.LimitCommits
	}

	emptyTime := time.Time{}
	if c.After != emptyTime {
		options.After = &c.After
	}
	if c.Before != emptyTime {
		options.Before = &c.Before
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

	if c.SaveEvery != 0 {
		options.SaveEvery = &c.SaveEvery
	}

	g := git.NewHistoryImporter(c.Paths, options)

	return ctx.ws.Import(g)
}

type ImportGitBlameCmd struct {
	Paths         []string `arg:"" help:"Paths with the roots of git repositories." type:"existingpath"`
	Incremental   bool     `default:"true" negatable:"" help:"Don't import files already imported."`
	LimitImported int      `help:"Limit the number of imported files. Can be used to incrementally import data. Counted by file name."`
	SaveEvery     int      `help:"Save results after some number of files."`
}

func (c *ImportGitBlameCmd) Run(ctx *context) error {
	options := git.BlameOptions{
		Incremental: c.Incremental,
	}
	if c.LimitImported != 0 {
		options.MaxImportedFiles = &c.LimitImported
	}
	if c.SaveEvery != 0 {
		options.SaveEvery = &c.SaveEvery
	}

	g := git.NewBlameImporter(c.Paths, options)

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
