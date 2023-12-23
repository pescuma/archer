package main

import (
	"strings"
	"time"

	"github.com/pescuma/archer/lib/importers/csproj"
	"github.com/pescuma/archer/lib/importers/git"
	"github.com/pescuma/archer/lib/importers/gomod"
	"github.com/pescuma/archer/lib/importers/hibernate"
	"github.com/pescuma/archer/lib/importers/loc"
	"github.com/pescuma/archer/lib/importers/metrics"
	"github.com/pescuma/archer/lib/importers/owners"
)

type ImportAllCmd struct {
	Paths       []string      `arg:"" help:"Paths to recursively search for data." type:"existingpath"`
	Branch      string        `help:"Git branch to use to import data."`
	Group       string        `help:"Group to use for the projects."`
	Gitignore   bool          `default:"true" help:"Respect .gitignore file when importing files."`
	Incremental bool          `default:"true" negatable:"" help:"Don't import commits already imported."`
	SaveEvery   time.Duration `default:"10m" help:"Save results while processing to avoid losing work."`
}

func (c *ImportAllCmd) Run(ctx *context) error {
	ws := ctx.ws

	ws.Console().PushPrefix("gomod: ")

	err := ws.ImportGoMod(c.Paths, &gomod.Options{
		Groups:           strings.Split(c.Group, ":"),
		RespectGitignore: c.Gitignore,
	})
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()
	ws.Console().PushPrefix("csproj: ")

	err = ws.ImportCsproj(c.Paths, &csproj.Options{
		Groups:           strings.Split(c.Group, ":"),
		RespectGitignore: c.Gitignore,
	})
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()
	ws.Console().PushPrefix("git people: ")

	err = ws.ImportGitPeople(c.Paths, &git.PeopleOptions{
		Branch: c.Branch,
	})
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()
	ws.Console().PushPrefix("git history: ")

	err = ws.ImportGitHistory(c.Paths, &git.HistoryOptions{
		Branch:      c.Branch,
		Incremental: c.Incremental,
		SaveEvery:   toOption(c.SaveEvery),
	})
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()
	ws.Console().PushPrefix("loc: ")

	err = ws.ImportLOC(nil, &loc.Options{
		Incremental: c.Incremental,
	})
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()
	ws.Console().PushPrefix("metrics: ")

	err = ws.ImportMetrics(nil, &metrics.Options{
		Incremental: c.Incremental,
		SaveEvery:   toOption(c.SaveEvery),
	})
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()
	ws.Console().PushPrefix("git blame: ")

	err = ws.ImportGitBlame(c.Paths, &git.BlameOptions{
		Branch:      c.Branch,
		Incremental: c.Incremental,
		SaveEvery:   toOption(c.SaveEvery),
	})
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()

	return nil
}

type ImportGradleCmd struct {
	Path string `arg:"" help:"Path to search for gradle projects." type:"existingpath"`
}

func (c *ImportGradleCmd) Run(ctx *context) error {
	return ctx.ws.ImportGradle(c.Path)
}

type ImportGoModCmd struct {
	Paths     []string `arg:"" help:"Paths to recursively search for go.mod files." type:"existingpath"`
	Group     string   `help:"Group to use for the projects."`
	Gitignore bool     `default:"true" help:"Respect .gitignore file when importing files."`
}

func (c *ImportGoModCmd) Run(ctx *context) error {
	return ctx.ws.ImportGoMod(c.Paths, &gomod.Options{
		Groups:           strings.Split(c.Group, ":"),
		RespectGitignore: c.Gitignore,
	})
}

type ImportCsprojCmd struct {
	Paths     []string `arg:"" help:"Paths to recursively search for csproj files." type:"existingpath"`
	Group     string   `help:"Group to use for the projects."`
	Gitignore bool     `default:"true" help:"Respect .gitignore file when importing files."`
}

func (c *ImportCsprojCmd) Run(ctx *context) error {
	return ctx.ws.ImportCsproj(c.Paths, &csproj.Options{
		Groups:           strings.Split(c.Group, ":"),
		RespectGitignore: c.Gitignore,
	})
}

type ImportHibernateCmd struct {
	Path        []string `arg:"" help:"Path with root of projects to search." type:"existingpath"`
	Group       string   `help:"Group to use for the projects."`
	Glob        []string `help:"Glob limiting files to be processed."`
	Incremental bool     `default:"true" negatable:"" help:"Don't import files already imported."`
}

func (c *ImportHibernateCmd) Run(ctx *context) error {
	return ctx.ws.ImportHibernate(c.Path, c.Glob, &hibernate.Options{
		Groups:      strings.Split(c.Group, ":"),
		Incremental: c.Incremental,
	})
}

type ImportMySqlCmd struct {
	ConnectionString string `arg:"" help:"MySQL connection string."`
}

func (c *ImportMySqlCmd) Run(ctx *context) error {
	return ctx.ws.ImportMySql(c.ConnectionString)
}

type ImportLOCCmd struct {
	Filters     []string `default:"" help:"Filters to be applied to the projects. Empty means all."`
	Incremental bool     `default:"true" negatable:"" help:"Don't import files already imported."`
}

func (c *ImportLOCCmd) Run(ctx *context) error {
	return ctx.ws.ImportLOC(c.Filters, &loc.Options{
		Incremental: c.Incremental,
	})
}

type ImportMetricsCmd struct {
	Filters       []string      `default:"" help:"Filters to be applied to the projects. Empty means all."`
	Incremental   bool          `default:"true" negatable:"" help:"Don't import files already imported."`
	LimitImported int           `help:"Limit the number of imported files. Can be used to incrementally import data."`
	SaveEvery     time.Duration `default:"10m" help:"Save results while processing to avoid losing work."`
}

func (c *ImportMetricsCmd) Run(ctx *context) error {
	return ctx.ws.ImportMetrics(c.Filters, &metrics.Options{
		Incremental:      c.Incremental,
		MaxImportedFiles: toOption(c.LimitImported),
		SaveEvery:        toOption(c.SaveEvery),
	})
}

type ImportGitPeopleCmd struct {
	Paths  []string `arg:"" help:"Paths with the roots of git repositories." type:"existingpath"`
	Branch string   `help:"Git branch to use to import data."`
}

func (c *ImportGitPeopleCmd) Run(ctx *context) error {
	return ctx.ws.ImportGitPeople(c.Paths, &git.PeopleOptions{
		Branch: c.Branch,
	})
}

type ImportGitHistoryCmd struct {
	Paths         []string      `arg:"" help:"Paths with the roots of git repositories." type:"existingpath"`
	Branch        string        `help:"Git branch to use to import data."`
	Incremental   bool          `default:"true" negatable:"" help:"Don't import commits already imported."`
	LimitImported int           `help:"Limit the number of imported commits. Can be used to incrementally import data. Counted from the latest commit."`
	LimitCommits  int           `help:"Limit the number of commits to be imported. Counted from the latest commit."`
	After         time.Time     `help:"Import commits after this date (inclusive)."`
	Before        time.Time     `help:"Import commits before this date (exclusive)."`
	SaveEvery     time.Duration `default:"10m" help:"Save results while processing to avoid losing work."`
}

func (c *ImportGitHistoryCmd) Run(ctx *context) error {
	return ctx.ws.ImportGitHistory(c.Paths, &git.HistoryOptions{
		Branch:             c.Branch,
		Incremental:        c.Incremental,
		MaxImportedCommits: toOption(c.LimitImported),
		MaxCommits:         toOption(c.LimitCommits),
		After:              toOption(c.After),
		Before:             toOption(c.Before),
		SaveEvery:          toOption(c.SaveEvery),
	})
}

type ImportGitBlameCmd struct {
	Paths         []string      `arg:"" help:"Paths with the roots of git repositories." type:"existingpath"`
	Branch        string        `help:"Git branch to use to import data."`
	Incremental   bool          `default:"true" negatable:"" help:"Don't import files already imported."`
	LimitImported int           `help:"Limit the number of imported files. Can be used to incrementally import data. Counted by file name."`
	SaveEvery     time.Duration `default:"10m" help:"Save results while processing to avoid losing work."`
}

func (c *ImportGitBlameCmd) Run(ctx *context) error {
	return ctx.ws.ImportGitBlame(c.Paths, &git.BlameOptions{
		Branch:           c.Branch,
		Incremental:      c.Incremental,
		MaxImportedFiles: toOption(c.LimitImported),
		SaveEvery:        toOption(c.SaveEvery),
	})
}

type ImportOwnersCmd struct {
	Filters       []string `default:"" help:"Filters to be applied to the projects. Empty means all."`
	Incremental   bool     `default:"true" negatable:"" help:"Don't import files already imported."`
	LimitImported int      `help:"Limit the number of imported files. Can be used to incrementally import data."`
	SaveEvery     int      `help:"Save results after some number of files."`
}

func (c *ImportOwnersCmd) Run(ctx *context) error {
	return ctx.ws.ImportOwners(c.Filters, &owners.Options{
		Incremental:      c.Incremental,
		MaxImportedFiles: toOption(c.LimitImported),
		SaveEvery:        toOption(c.SaveEvery),
	})
}

func toOption[T comparable](d T) *T {
	var def T

	if d == def {
		return nil
	} else {
		return &d
	}
}
