package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/importers/csproj"
	"github.com/pescuma/archer/lib/importers/git"
	"github.com/pescuma/archer/lib/importers/gomod"
	"github.com/pescuma/archer/lib/importers/gradle"
	"github.com/pescuma/archer/lib/importers/hibernate"
	"github.com/pescuma/archer/lib/importers/loc"
	"github.com/pescuma/archer/lib/importers/metrics"
	"github.com/pescuma/archer/lib/importers/mysql"
	"github.com/pescuma/archer/lib/importers/owners"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/storages/orm"
	"github.com/pescuma/archer/lib/utils"
)

type Workspace struct {
	console consoles.Console
	storage storages.Storage
}

func NewWorkspace(file string) (*Workspace, error) {
	if file == "" {
		if _, err := os.Stat("./.archer"); err == nil {
			file = "./.archer/archer.sqlite"
		} else {
			file = "~/.archer/archer.sqlite"
		}
	}

	var storage storages.Storage
	var err error
	switch {
	case file == ":memory:":
		storage, err = orm.NewGormStorage(orm.WithSqliteInMemory())

	case strings.HasSuffix(file, ".sqlite"):
		file, err = utils.PathAbs(file)
		if err != nil {
			return nil, err
		}

		path := filepath.Dir(file)
		if _, err := os.Stat(path); err != nil {
			fmt.Printf("Creating workspace at %v\n", path)
			err = os.MkdirAll(path, 0o700)
			if err != nil {
				return nil, err
			}
		}

		storage, err = orm.NewGormStorage(orm.WithSqlite(file))

	default:
		return nil, fmt.Errorf("unknown storage type for file %v", file)
	}
	if err != nil {
		return nil, err
	}

	return &Workspace{
		console: consoles.NewStdOutConsole(),
		storage: storage,
	}, nil
}

func (w *Workspace) Close() error {
	return w.storage.Close()
}

func (w *Workspace) Console() consoles.Console {
	return w.console
}

func (w *Workspace) LoadProjects() (*model.Projects, error) {
	return w.storage.LoadProjects()
}

func (w *Workspace) Execute(f func(consoles.Console, storages.Storage) error) error {
	return f(consoles.NewStdOutConsole(), w.storage)
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

	err = w.storage.WriteConfig()
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

func (w *Workspace) ImportCsproj(dirs []string, opts *csproj.Options) error {
	importer := csproj.NewImporter(w.console, w.storage)
	return importer.Import(dirs, opts)
}

func (w *Workspace) ImportGoMod(dirs []string, opts *gomod.Options) error {
	importer := gomod.NewImporter(w.console, w.storage)
	return importer.Import(dirs, opts)
}

func (w *Workspace) ImportGradle(dir string) error {
	importer := gradle.NewImporter(w.console, w.storage)
	return importer.Import(dir)
}

func (w *Workspace) ImportGitPeople(dirs []string) error {
	importer := git.NewPeopleImporter(w.console, w.storage)
	return importer.Import(dirs)
}

func (w *Workspace) ImportGitHistory(dirs []string, opts *git.HistoryOptions) error {
	importer := git.NewHistoryImporter(w.console, w.storage)
	return importer.Import(dirs, opts)
}

func (w *Workspace) ImportGitBlame(dirs []string, opts *git.BlameOptions) error {
	importer := git.NewBlameImporter(w.console, w.storage)
	return importer.Import(dirs, opts)
}

func (w *Workspace) ImportHibernate(rootDirs, globs []string, opts *hibernate.Options) error {
	importer := hibernate.NewImporter(w.console, w.storage)
	return importer.Import(rootDirs, globs, opts)
}

func (w *Workspace) ImportLOC(filter []string, opts *loc.Options) error {
	importer := loc.NewImporter(w.console, w.storage)
	return importer.Import(filter, opts)
}

func (w *Workspace) ImportMetrics(filter []string, opts *metrics.Options) error {
	importer := metrics.NewImporter(w.console, w.storage)
	return importer.Import(filter, opts)
}

func (w *Workspace) ImportMySql(connectionString string) error {
	importer := mysql.NewImporter(w.console, w.storage)
	return importer.Import(connectionString)
}

func (w *Workspace) ImportOwners(filter []string, opts *owners.Options) error {
	importer := owners.NewImporter(w.console, w.storage)
	return importer.Import(filter, opts)
}
