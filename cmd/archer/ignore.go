package main

type IgnoreAddFileCmd struct {
	Rule string `arg:"" help:"File ignore rule: a file path or a regex."`
}

func (c *IgnoreAddFileCmd) Run(ctx *context) error {
	return ctx.ws.IgnoreAddFileRule(c.Rule)
}

type IgnoreAddCommitCmd struct {
	Rule string `arg:"" help:"Commit ignore rule: a commit hash or a regex."`
}

func (c *IgnoreAddCommitCmd) Run(ctx *context) error {
	return ctx.ws.IgnoreAddCommitRule(c.Rule)
}
