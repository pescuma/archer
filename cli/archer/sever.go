package main

import (
	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/server"
	"github.com/pescuma/archer/lib/storages"
)

type ServerCmd struct {
	Port uint `default:"2724" help:"Port to listen to."`
}

func (c *ServerCmd) Run(ctx *context) error {
	return ctx.ws.Execute(func(console consoles.Console, storage storages.Storage) error {
		return server.Run(console, storage, &server.Options{
			Port: c.Port,
		})
	})
}
