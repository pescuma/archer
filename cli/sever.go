package main

import (
	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/server"
)

type ServerCmd struct {
	Port uint `default:"2724" help:"Port to listen to."`
}

func (c *ServerCmd) Run(ctx *context) error {
	return ctx.ws.Execute(func(storage archer.Storage) error {
		return server.Run(storage, &server.Options{
			Port: c.Port,
		})
	})
}
