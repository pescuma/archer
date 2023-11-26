package workspace

import (
	"os"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/importers"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
)

type Workspace struct {
	storage storages.Storage
}

func NewWorkspace(factory storages.Factory, root string) (*Workspace, error) {
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

func (w *Workspace) Close() error {
	return w.storage.Close()
}

func (w *Workspace) LoadProjects() (*model.Projects, error) {
	return w.storage.LoadProjects()
}

func (w *Workspace) Execute(f func(consoles.Console, storages.Storage) error) error {
	return f(consoles.NewStdOutConsole(), w.storage)
}

func (w *Workspace) Import(importer importers.Importer) error {
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
