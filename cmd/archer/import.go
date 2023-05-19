package main

import (
	"github.com/Faire/archer/lib/archer/importers/git"
	"github.com/Faire/archer/lib/archer/importers/gradle"
	"github.com/Faire/archer/lib/archer/importers/hibernate"
	"github.com/Faire/archer/lib/archer/importers/loc"
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

type ImportGitCmd struct {
	Path string `arg:"" help:"Path with root of git repository." type:"existingpath"`
}

func (c *ImportGitCmd) Run(ctx *context) error {
	g := git.NewImporter(c.Path)

	return ctx.ws.Import(g)
}
