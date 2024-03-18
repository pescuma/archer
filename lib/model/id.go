package model

import (
	"fmt"
	"strconv"

	"github.com/teris-io/shortid"
	"golang.org/x/exp/rand"
)

type ID int

func (i ID) String() string {
	return fmt.Sprintf("%v", int(i))
}

func StringToID(id string) (ID, error) {
	r, err := strconv.ParseInt(id, 10, 32)
	return ID(r), err
}

type UUID string

func NewUUID(t string) UUID {
	return UUID(shortid.MustGenerate() + t)
}

func init() {
	sid := shortid.MustNew(0, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_.", rand.Uint64())
	shortid.SetDefault(sid)
}
