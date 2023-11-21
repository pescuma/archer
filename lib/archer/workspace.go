package archer

import (
	"os"

	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
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
	return w.storage.LoadProjects()
}

func (w *Workspace) Execute(f func(Storage) error) error {
	return f(w.storage)
}

func (w *Workspace) Import(importer Importer) error {
	return w.Execute(importer.Import)
}

func (w *Workspace) SetGlobalConfig(config string, value string) (bool, error) {
	cfg, err := w.storage.LoadConfig()
	if err != nil {
		return false, err
	}

	v, ok := (*cfg)[config]
	if ok && v != value {
		return false, nil
	}

	(*cfg)[config] = value

	err = w.storage.WriteConfig(cfg)
	if err != nil {
		return false, err
	}

	return true, nil

}

func (w *Workspace) SetProjectConfig(proj *model.Project, config string, value string) (bool, error) {
	changed := proj.SetData(config, value)

	if changed {
		err := w.storage.WriteProject(proj)
		if err != nil {
			return false, err
		}
	}

	return changed, nil
}
