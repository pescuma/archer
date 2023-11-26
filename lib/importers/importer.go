package importers

import (
	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/storages"
)

type Importer interface {
	Import(console consoles.Console, storage storages.Storage) error
}
