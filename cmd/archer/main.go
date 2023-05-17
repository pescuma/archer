package main

import (
	"github.com/alecthomas/kong"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/storage"
)

var cli struct {
	Workspace string `short:"w" help:"Workspace to store data. Default is ./.archer or ~/.archer if that does not exist." type:"path"`

	Show  ShowCmd  `cmd:"" help:"Show the dependencies of projects inside a json file."`
	Graph GraphCmd `cmd:"" help:"Generate dependencies graph. Requires dot in path."`

	Config struct {
		Set ConfigSetCmd `cmd:"" help:"Set configuration parameters."`
	} `cmd:""`

	Import struct {
		Gradle    ImportGradleCmd    `cmd:"" help:"Import information from gradle project."`
		Hibernate ImportHibernateCmd `cmd:"" help:"Import information from hibernate annotation in classes."`
		Mysql     ImportMySqlCmd     `cmd:"" help:"Import information from MySQL schema."`
		Size      ImportSizeCmd      `cmd:"" help:"Import size information to existing projects."`
	} `cmd:""`
}

type context struct {
	ws *archer.Workspace
}

func main() {
	ctx := kong.Parse(&cli, kong.ShortUsageOnError())

	workspace, err := archer.NewWorkspace(storage.NewJsonStorage, cli.Workspace)
	ctx.FatalIfErrorf(err)

	err = ctx.Run(&context{
		ws: workspace,
	})
	ctx.FatalIfErrorf(err)
}
