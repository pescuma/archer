package archer

type Storage interface {
	LoadProjects(result *Projects) error
	WriteProjNames(projRoot string, projNames []string) error
	ReadProjNames() ([]string, error)
	WriteDeps(proj *Project) error
	ReadDeps(result *Projects, fileName string) error
	WriteSize(proj *Project) error
	ReadSize(result *Projects, fileName string) error
	WriteBasicInfo(proj *Project) error
	ReadBasicInfo(result *Projects, fileName string) error
	WriteConfig(proj *Project) error
	ReadConfig(result *Projects, fileName string) error
}

type StorageFactory = func(root string) (Storage, error)
