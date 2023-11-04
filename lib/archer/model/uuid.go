package model

import (
	"github.com/teris-io/shortid"
	"golang.org/x/exp/rand"
)

type UUID string

func init() {
	sid := shortid.MustNew(0, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_.", rand.Uint64())
	shortid.SetDefault(sid)
}

func NewUUID(t string) UUID {
	return UUID(shortid.MustGenerate() + t)
}
