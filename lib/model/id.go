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

func MustStringToID(id string) ID {
	r, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		panic(err)
	}
	return ID(r)
}

func createID(maxID *ID, id *ID) ID {
	if id == nil {
		*maxID++
		return *maxID

	} else {
		if *id > *maxID {
			*maxID = *id
		}

		return *id
	}
}

type UUID string

func NewUUID(t string) UUID {
	return UUID(shortid.MustGenerate() + t)
}

func init() {
	sid := shortid.MustNew(0, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_.", rand.Uint64())
	shortid.SetDefault(sid)
}
