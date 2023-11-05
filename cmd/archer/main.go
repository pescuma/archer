package main

import (
	"github.com/alecthomas/kong"

	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/storage/sqlite"
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
		Gomod     ImportGomodCmd     `cmd:"" help:"Import information from go.mod files."`
		Csproj    ImportCsprojCmd    `cmd:"" help:"Import information from csproj files."`
		Hibernate ImportHibernateCmd `cmd:"" help:"Import information from hibernate annotation in classes."`
		Mysql     ImportMySqlCmd     `cmd:"" help:"Import information from MySQL schema."`
		LOC       ImportLOCCmd       `cmd:"" help:"Import counts of lines of code to existing projects."`
		Metrics   ImportMetricsCmd   `cmd:"" help:"Import code metrics to existing projects."`
		Git       struct {
			People  ImportGitPeopleCmd  `cmd:"" help:"Import people information from git."`
			History ImportGitHistoryCmd `cmd:"" help:"Import history information from git."`
			Blame   ImportGitBlameCmd   `cmd:"" help:"Import blame information from git."`
		} `cmd:""`
		Owners ImportOwnersCmd `cmd:"" help:"Import file owners."`
	} `cmd:""`
}

type context struct {
	ws *archer.Workspace
}

func main() {
	ctx := kong.Parse(&cli, kong.ShortUsageOnError())

	workspace, err := archer.NewWorkspace(sqlite.NewSqliteStorage, cli.Workspace)
	ctx.FatalIfErrorf(err)

	err = ctx.Run(&context{
		ws: workspace,
	})
	ctx.FatalIfErrorf(err)
}
