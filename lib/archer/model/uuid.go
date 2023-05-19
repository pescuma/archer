package model

import (
	"github.com/lithammer/shortuuid/v4"
)

type UUID string

func NewUUID(t string) UUID {
	return UUID(shortuuid.New() + t)
}
