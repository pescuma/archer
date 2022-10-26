package archer

type Importer interface {
	Import(projs *Projects, storage *Storage) error
}
