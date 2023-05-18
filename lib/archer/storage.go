package archer

import (
	"github.com/Faire/archer/lib/archer/model"
)

type Storage interface {
	LoadProjects(result *model.Projects) error
	WriteProjNames(projRoot string, projNames []string) error
	ReadProjNames() ([]string, error)
	WriteBasicInfo(proj *model.Project) error
	ReadBasicInfo(result *model.Projects, fileName string) error
	WriteDeps(proj *model.Project) error
	ReadDeps(result *model.Projects, fileName string) error
	WriteSize(proj *model.Project) error
	ReadSize(result *model.Projects, fileName string) error
	WriteFiles(proj *model.Project) error
	ReadFiles(result *model.Projects, fileName string) error
	WriteConfig(proj *model.Project) error
	ReadConfig(result *model.Projects, fileName string) error
}

type StorageFactory = func(root string) (Storage, error)
