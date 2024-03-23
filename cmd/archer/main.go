package main

import (
	"github.com/alecthomas/kong"

	"github.com/pescuma/archer/lib/workspace"
)

var cli struct {
	Workspace string `short:"w" help:"Workspace to store data. Default is ./.archer/archer.sqlite or ~/.archer/archer.sqlite if that does not exist." type:"file"`

	Show  ShowCmd  `cmd:"" help:"Show the dependencies of projects inside a json file."`
	Graph GraphCmd `cmd:"" help:"Generate dependencies graph. Requires dot in path."`

	Config struct {
		Set ConfigSetCmd `cmd:"" help:"Set configuration parameters."`
	} `cmd:""`

	Import struct {
		All       ImportAllCmd       `cmd:"" help:"Import all information recursively."`
		Gradle    ImportGradleCmd    `cmd:"" help:"Import information from gradle project."`
		GoMod     ImportGoModCmd     `cmd:"" help:"Import information from go.mod files."`
		Csproj    ImportCsprojCmd    `cmd:"" help:"Import information from csproj files."`
		Hibernate ImportHibernateCmd `cmd:"" help:"Import information from hibernate annotation in classes."`
		Mysql     ImportMySqlCmd     `cmd:"" help:"Import information from MySQL schema."`
		LOC       ImportLOCCmd       `cmd:"" help:"Import counts of lines of code to existing projects."`
		Metrics   ImportMetricsCmd   `cmd:"" help:"Import code metrics to existing projects."`
		Git       struct {
			History ImportGitHistoryCmd `cmd:"" help:"Import history information from git."`
			Blame   ImportGitBlameCmd   `cmd:"" help:"Import blame information from git."`
			People  ImportGitPeopleCmd  `cmd:"" help:"Import only people information from git."`
			Repos   ImportGitReposCmd   `cmd:"" help:"Import only repository information from git."`
		} `cmd:""`
		Owners ImportOwnersCmd `cmd:"" help:"Import file owners."`
	} `cmd:""`

	Compute struct {
		All     ComputeAllCmd     `cmd:"" help:"Compute all based on imported information."`
		LOC     ComputeLOCCmd     `cmd:"" help:"Compute lines of code based on imported files."`
		Metrics ComputeMetricsCmd `cmd:"" help:"Compute code metrics based on imported files."`
		History ComputeHistoryCmd `cmd:"" help:"Compute history based on imported files."`
		Blame   ComputeBlameCmd   `cmd:"" help:"Compute blame based on imported files."`
	} `cmd:""`

	Ignore struct {
		Add struct {
			File   IgnoreAddFileCmd   `cmd:"" help:"Add a file ignore rule."`
			Commit IgnoreAddCommitCmd `cmd:"" help:"Add a commit ignore rule."`
		} `cmd:""`
	} `cmd:""`

	Run struct {
		Git RunGitCmd `cmd:"" help:"Run git commands on all imported repositories."`
	} `cmd:""`

	Server ServerCmd `cmd:"" help:"Start webserver."`
}

type context struct {
	ws *workspace.Workspace
}

func main() {
	ctx := kong.Parse(&cli, kong.ShortUsageOnError())

	err := run(ctx)
	ctx.FatalIfErrorf(err)
}

func run(ctx *kong.Context) error {
	ws, err := workspace.NewWorkspace(cli.Workspace)
	if err != nil {
		return err
	}

	defer func() {
		_ = ws.Close()
	}()

	err = ctx.Run(&context{
		ws: ws,
	})
	if err != nil {
		return err
	}

	err = ws.Write()
	if err != nil {
		return err
	}

	return nil
}
