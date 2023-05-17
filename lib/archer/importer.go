package archer

import (
	"github.com/Faire/archer/lib/archer/model"
)

type Importer interface {
	Import(projs *model.Projects, storage Storage) error
}
