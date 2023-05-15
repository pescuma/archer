package archer

import (
	"fmt"
	"os"

	"github.com/Faire/archer/lib/archer/utils"
)

type Workspace struct {
	storage Storage
}

func NewWorkspace(factory StorageFactory, root string) (*Workspace, error) {
	if root == "" {
		if _, err := os.Stat("./.archer"); err == nil {
			root = "./.archer"
		} else {
			root = "~/.archer"
		}
	}

	root, err := utils.PathAbs(root)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(root); err != nil {
		fmt.Printf("Creating workspace at %v\n", root)
		err := os.MkdirAll(root, 0o700)
		if err != nil {
			return nil, err
		}
	}

	storage, err := factory(root)
	if err != nil {
		return nil, err
	}

	return &Workspace{
		storage: storage,
	}, nil
}

func (w *Workspace) LoadProjects() (*Projects, error) {
	result := NewProjects()

	err := w.storage.LoadProjects(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (w *Workspace) Import(importer Importer) error {
	projs := NewProjects()

	err := w.storage.LoadProjects(projs)
	if err != nil {
		return err
	}

	return importer.Import(projs, w.storage)
}

func (w *Workspace) SetConfigParameter(proj *Project, config string, value string) (bool, error) {
	changed := proj.SetConfig(config, value)

	if changed {
		err := w.storage.WriteConfig(proj)
		if err != nil {
			return false, err
		}
	}

	return changed, nil
}
