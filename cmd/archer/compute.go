package main

type ComputeAllCmd struct {
}

func (c *ComputeAllCmd) Run(ctx *context) error {
	ws := ctx.ws

	ws.Console().PushPrefix("loc: ")

	err := ws.ComputeLOC()
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()
	ws.Console().PushPrefix("metrics: ")

	err = ws.ComputeMetrics()
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()
	ws.Console().PushPrefix("history: ")

	err = ws.ComputeHistory()
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()
	ws.Console().PushPrefix("blame: ")

	err = ws.ComputeBlame()
	if err != nil {
		return err
	}

	ws.Console().PopPrefix()

	return nil
}

type ComputeLOCCmd struct {
}

func (c *ComputeLOCCmd) Run(ctx *context) error {
	return ctx.ws.ComputeLOC()
}

type ComputeMetricsCmd struct {
}

func (c *ComputeMetricsCmd) Run(ctx *context) error {
	return ctx.ws.ComputeMetrics()
}

type ComputeHistoryCmd struct {
}

func (c *ComputeHistoryCmd) Run(ctx *context) error {
	return ctx.ws.ComputeHistory()
}

type ComputeBlameCmd struct {
}

func (c *ComputeBlameCmd) Run(ctx *context) error {
	return ctx.ws.ComputeBlame()
}
