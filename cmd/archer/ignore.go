package main

type IgnoreAddCommitCmd struct {
	Rule string `arg:"" help:"Commit ignore rule: a commit hash or a regex."`
}

func (c *IgnoreAddCommitCmd) Run(ctx *context) error {
	return ctx.ws.IgnoreAddCommitRule(c.Rule)
}
