package main

type RunGitCmd struct {
	Args []string `arg:"" help:"Arguments to pass to git command. This requires git to be in path."`
}

func (c *RunGitCmd) Run(ctx *context) error {
	return ctx.ws.RunGit(c.Args...)
}
