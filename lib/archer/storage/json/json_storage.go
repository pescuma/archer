package json

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/model"
)

const (
	projNamesJson = "projNames.json"
	depsJson      = "deps.json"
	sizeJson      = "size.json"
	basicInfoJson = "proj.json"
	configJson    = "config.json"
	filesJson     = "files.json"
)

type jsonStorage struct {
	root string
}

func NewJsonStorage(root string) (archer.Storage, error) {
	if _, err := os.Stat(root); err != nil {
		fmt.Printf("Creating workspace at %v\n", root)
		err := os.MkdirAll(root, 0o700)
		if err != nil {
			return nil, err
		}
	}

	return &jsonStorage{
		root: root,
	}, nil
}

func (s *jsonStorage) LoadProjects() (*model.Projects, error) {
	result := model.NewProjects()

	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		switch d.Name() {
		case basicInfoJson:
			err = s.readBasicInfo(result, path)
			if err != nil {
				return err
			}

		case depsJson:
			err = s.readDeps(result, path)
			if err != nil {
				return err
			}

		case sizeJson:
			err = s.readSize(result, path)
			if err != nil {
				return err
			}

		case filesJson:
			err = s.readFiles(result, path)
			if err != nil {
				return err
			}

		case configJson:
			err = s.readConfig(result, path)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *jsonStorage) WriteProjects(projs *model.Projects, changes archer.StorageChanges) error {
	byRoot := lo.GroupBy(projs.ListProjects(model.FilterAll), func(p *model.Project) string { return p.RootDir })

	for root, projs := range byRoot {
		err := s.writeProjNames(root, lo.Map(projs, func(p *model.Project, _ int) string { return p.Name }))
		if err != nil {
			return err
		}

		for _, proj := range projs {
			err = s.WriteProject(proj, changes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *jsonStorage) WriteProject(proj *model.Project, changes archer.StorageChanges) error {
	if changes&archer.ChangedBasicInfo != 0 {
		err := s.writeBasicInfo(proj)
		if err != nil {
			return err
		}
	}

	if changes&archer.ChangedDependencies != 0 {
		err := s.writeDeps(proj)
		if err != nil {
			return err
		}
	}

	if changes&archer.ChangedSize != 0 {
		err := s.writeSize(proj)
		if err != nil {
			return err
		}
	}

	if changes&archer.ChangedData != 0 {
		err := s.writeConfig(proj)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *jsonStorage) getProjNamesFileName(root string) (string, error) {
	return filepath.Abs(filepath.Join(
		s.root,
		strings.ReplaceAll(root, ":", "_"),
		projNamesJson,
	))
}

func (s *jsonStorage) writeProjNames(projRoot string, projNames []string) error {
	fileName, err := s.getProjNamesFileName(projRoot)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	content, err := ProjNamesToJson(projRoot, projNames)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(content), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) readProjNames() ([]string, error) {
	fileName, err := s.getProjNamesFileName(s.root)
	if err != nil {
		return nil, err
	}

	contents, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	_, result, err := ProjNamesFromJson(string(contents))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *jsonStorage) computeDataDir(proj *model.Project) (string, error) {
	dir, err := filepath.Abs(filepath.Join(
		s.root,
		proj.Root,
		strings.TrimLeft(strings.ReplaceAll(proj.Name, ":", string(os.PathSeparator)), string(os.PathSeparator)),
	))
	if err != nil {
		return "", err
	}

	return dir, nil
}

func (s *jsonStorage) getBasicInfoFileName(proj *model.Project) (string, error) {
	dataDir, err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, basicInfoJson), nil
}

func (s *jsonStorage) writeBasicInfo(proj *model.Project) error {
	fileName, err := s.getBasicInfoFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := ProjBasicInfoToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) readBasicInfo(result *model.Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = ProjBasicInfoFromJson(result, string(contents))
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) getDepsFileName(proj *model.Project) (string, error) {
	dataDir, err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, depsJson), nil
}

func (s *jsonStorage) writeDeps(proj *model.Project) error {
	fileName, err := s.getDepsFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := ProjDepsToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) readDeps(result *model.Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = ProjDepsFromJson(result, string(contents))
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) getSizeFileName(proj *model.Project) (string, error) {
	dataDir, err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, sizeJson), nil
}

func (s *jsonStorage) writeSize(proj *model.Project) error {
	fileName, err := s.getSizeFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := SizeToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) readSize(result *model.Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = SizeFromJson(result, string(contents))
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) getFilesFileName(proj *model.Project) (string, error) {
	dataDir, err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, filesJson), nil
}

func (s *jsonStorage) writeFiles(proj *model.Project) error {
	fileName, err := s.getFilesFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := FilesToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) WriteFile(file *model.File, changes archer.StorageChanges) error {
	//TODO implement me
	panic("implement me")
}

func (s *jsonStorage) readFiles(result *model.Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = FilesFromJson(result, string(contents))
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) getConfigFileName(proj *model.Project) (string, error) {
	dataDir, err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, configJson), nil
}

func (s *jsonStorage) writeConfig(proj *model.Project) error {
	fileName, err := s.getConfigFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := ConfigToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) readConfig(result *model.Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = ConfigFromJson(result, string(contents))
	if err != nil {
		return err
	}

	return nil
}

func (s *jsonStorage) LoadFiles() (*model.Files, error) {
	// TODO implement me
	panic("implement me")
}

func (s *jsonStorage) WriteFiles(files *model.Files, changes archer.StorageChanges) error {
	// TODO implement me
	panic("implement me")
}

func (s *jsonStorage) LoadFileContents(file model.UUID) (*model.FileContents, error) {
	//TODO implement me
	panic("implement me")
}

func (s *jsonStorage) WriteFileContents(contents *model.FileContents, changes archer.StorageChanges) error {
	//TODO implement me
	panic("implement me")
}

func (s *jsonStorage) ComputeBlamePerAuthor() ([]*archer.BlamePerAuthor, error) {
	//TODO implement me
	panic("implement me")
}

func (s *jsonStorage) ComputeSurvivedLines() ([]*archer.SurvivedLineCount, error) {
	//TODO implement me
	panic("implement me")
}

func (s *jsonStorage) LoadPeople() (*model.People, error) {
	// TODO implement me
	panic("implement me")
}

func (s *jsonStorage) WritePeople(people *model.People, changes archer.StorageChanges) error {
	// TODO implement me
	panic("implement me")
}

func (s *jsonStorage) LoadRepositories() (repos *model.Repositories, err error) {
	// TODO implement me
	panic("implement me")
}

func (s *jsonStorage) LoadRepository(rootDir string) (*model.Repository, error) {
	// TODO implement me
	panic("implement me")
}

func (s *jsonStorage) WriteRepository(repo *model.Repository, changes archer.StorageChanges) error {
	// TODO implement me
	panic("implement me")
}

func (s *jsonStorage) LoadConfig() (*map[string]string, error) {
	// TODO implement me
	panic("implement me")
}

func (s *jsonStorage) WriteConfig(m *map[string]string) error {
	// TODO implement me
	panic("implement me")
}
