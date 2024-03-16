package main

type ComputeLOCCmd struct {
}

func (c *ComputeLOCCmd) Run(ctx *context) error {
	return ctx.ws.ComputeLOC()
}
