package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/abiosoft/lineprefix"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/importers/blame"
	"github.com/pescuma/archer/lib/importers/csproj"
	"github.com/pescuma/archer/lib/importers/git"
	"github.com/pescuma/archer/lib/importers/gomod"
	"github.com/pescuma/archer/lib/importers/gradle"
	"github.com/pescuma/archer/lib/importers/hibernate"
	"github.com/pescuma/archer/lib/importers/history"
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

	console := consoles.NewStdOutConsole()

	var storage storages.Storage
	var err error
	switch {
	case file == ":memory:":
		storage, err = orm.NewGormStorage(orm.WithSqliteInMemory(), console)

	case strings.HasSuffix(file, ".sqlite"):
		file, err = utils.PathAbs(file)
		if err != nil {
			return nil, err
		}

		err = createWorkspaceDir(file)
		if err != nil {
			return nil, err
		}

		storage, err = orm.NewGormStorage(orm.WithSqlite(file), console)

	default:
		return nil, fmt.Errorf("unknown storage type for file %v", file)
	}
	if err != nil {
		return nil, err
	}

	return &Workspace{
		console: console,
		storage: storage,
	}, nil
}

func createWorkspaceDir(file string) error {
	path := filepath.Dir(file)

	if _, err := os.Stat(path); err != nil {
		fmt.Printf("Creating workspace at %v\n", path)
		err = os.MkdirAll(path, 0o700)
		if err != nil {
			return err
		}
	}

	return nil
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

	return true, nil

}

func (w *Workspace) SetProjectConfig(proj *model.Project, config string, value string) (bool, error) {
	changed := proj.SetData(config, value)

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

func (w *Workspace) ImportGitRepos(dirs []string, opts *git.ReposOptions) error {
	importer := git.NewReposImporter(w.console, w.storage)
	return importer.Import(dirs, opts)
}

func (w *Workspace) ImportGitPeople(dirs []string, opts *git.PeopleOptions) error {
	importer := git.NewPeopleImporter(w.console, w.storage)
	return importer.Import(dirs, opts)
}

func (w *Workspace) ImportGitHistory(dirs []string, opts *git.HistoryOptions) error {
	importer := git.NewHistoryImporter(w.console, w.storage)
	return importer.Import(dirs, opts)
}

func (w *Workspace) ComputeHistory() error {
	computer := history.NewComputer(w.console, w.storage)
	return computer.Compute()
}

func (w *Workspace) ImportGitBlame(dirs []string, opts *git.BlameOptions) error {
	importer := git.NewBlameImporter(w.console, w.storage)
	return importer.Import(dirs, opts)
}

func (w *Workspace) ComputeBlame() error {
	computer := blame.NewComputer(w.console, w.storage)
	return computer.Compute()
}

func (w *Workspace) ImportHibernate(rootDirs, globs []string, opts *hibernate.Options) error {
	importer := hibernate.NewImporter(w.console, w.storage)
	return importer.Import(rootDirs, globs, opts)
}

func (w *Workspace) ImportLOC(filter []string, opts *loc.Options) error {
	importer := loc.NewImporter(w.console, w.storage)
	return importer.Import(filter, opts)
}

func (w *Workspace) ComputeLOC() error {
	computer := loc.NewComputer(w.console, w.storage)
	return computer.Compute()
}

func (w *Workspace) ImportMetrics(filter []string, opts *metrics.Options) error {
	importer := metrics.NewImporter(w.console, w.storage)
	return importer.Import(filter, opts)
}

func (w *Workspace) ComputeMetrics() error {
	computer := metrics.NewComputer(w.console, w.storage)
	return computer.Compute()
}

func (w *Workspace) ImportMySql(connectionString string) error {
	importer := mysql.NewImporter(w.console, w.storage)
	return importer.Import(connectionString)
}

func (w *Workspace) ImportOwners(filter []string, opts *owners.Options) error {
	importer := owners.NewImporter(w.console, w.storage)
	return importer.Import(filter, opts)
}

func (w *Workspace) RunGit(args ...string) error {
	repos, err := w.storage.LoadRepositories()
	if err != nil {
		return err
	}

	for _, repo := range repos.List() {
		if repo.VCS != "git" {
			continue
		}

		cmd := exec.Command("git", args...)
		cmd.Dir = repo.RootDir
		if err != nil {
			return err
		}

		w.console.Printf("%v: Executing '%v'\n", repo.Name, strings.Join(cmd.Args, "' '"))
		w.console.PushPrefix("%v: ", repo.Name)

		prefix := lineprefix.PrefixFunc(func() string {
			return w.console.Prepare("")
		})

		cmd.Stdin = os.Stdin
		cmd.Stdout = lineprefix.New(lineprefix.Writer(os.Stdout), prefix)
		cmd.Stderr = lineprefix.New(lineprefix.Writer(os.Stderr), prefix)

		_ = cmd.Run()

		w.console.PopPrefix()
	}

	return nil
}

func (w *Workspace) Write() error {
	w.console.Printf("Writing results...\n")

	err := w.storage.WriteConfig()
	if err != nil {
		return err
	}

	err = w.storage.WriteProjects()
	if err != nil {
		return err
	}

	err = w.storage.WriteFiles()
	if err != nil {
		return err
	}

	err = w.storage.WritePeople()
	if err != nil {
		return err
	}

	err = w.storage.WritePeopleRelations()
	if err != nil {
		return err
	}

	err = w.storage.WriteRepositories()
	if err != nil {
		return err
	}

	err = w.storage.WriteMonthlyStats()
	if err != nil {
		return err
	}

	return nil
}
