package archer

type Importer interface {
	Import(storage Storage) error
}
