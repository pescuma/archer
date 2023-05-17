package archer

import (
	"os"

	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type Workspace struct {
	storage Storage
}

func NewWorkspace(factory StorageFactory, root string) (*Workspace, error) {
	if root == "" {
		if _, err := os.Stat("./.archer"); err == nil {
			root = "./.archer/"
		} else {
			root = "~/.archer/"
		}
	}

	root, err := utils.PathAbs(root)
	if err != nil {
		return nil, err
	}

	storage, err := factory(root)
	if err != nil {
		return nil, err
	}

	return &Workspace{
		storage: storage,
	}, nil
}

func (w *Workspace) LoadProjects() (*model.Projects, error) {
	result := model.NewProjects()

	err := w.storage.LoadProjects(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (w *Workspace) Import(importer Importer) error {
	projs := model.NewProjects()

	err := w.storage.LoadProjects(projs)
	if err != nil {
		return err
	}

	return importer.Import(projs, w.storage)
}

func (w *Workspace) SetConfigParameter(proj *model.Project, config string, value string) (bool, error) {
	changed := proj.SetData(config, value)

	if changed {
		err := w.storage.WriteConfig(proj)
		if err != nil {
			return false, err
		}
	}

	return changed, nil
}
