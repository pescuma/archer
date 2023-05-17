package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
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

func (s *jsonStorage) LoadProjects(result *model.Projects) error {
	return filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		switch d.Name() {
		case basicInfoJson:
			err = s.ReadBasicInfo(result, path)
			if err != nil {
				return err
			}

		case depsJson:
			err = s.ReadDeps(result, path)
			if err != nil {
				return err
			}

		case sizeJson:
			err = s.ReadSize(result, path)
			if err != nil {
				return err
			}

		case filesJson:
			err = s.ReadFiles(result, path)
			if err != nil {
				return err
			}

		case configJson:
			err = s.ReadConfig(result, path)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *jsonStorage) getProjNamesFileName(root string) (string, error) {
	return filepath.Abs(filepath.Join(
		s.root,
		strings.ReplaceAll(root, ":", "_"),
		projNamesJson,
	))
}

func (s *jsonStorage) WriteProjNames(projRoot string, projNames []string) error {
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

func (s *jsonStorage) ReadProjNames() ([]string, error) {
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

func (s *jsonStorage) getDepsFileName(proj *model.Project) (string, error) {
	dataDir, err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, depsJson), nil
}

func (s *jsonStorage) WriteDeps(proj *model.Project) error {
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

func (s *jsonStorage) ReadDeps(result *model.Projects, fileName string) error {
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

func (s *jsonStorage) WriteSize(proj *model.Project) error {
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

func (s *jsonStorage) ReadSize(result *model.Projects, fileName string) error {
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

func (s *jsonStorage) getBasicInfoFileName(proj *model.Project) (string, error) {
	dataDir, err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, basicInfoJson), nil
}

func (s *jsonStorage) WriteBasicInfo(proj *model.Project) error {
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

func (s *jsonStorage) ReadBasicInfo(result *model.Projects, fileName string) error {
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

func (s *jsonStorage) getFilesFileName(proj *model.Project) (string, error) {
	dataDir, err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, filesJson), nil
}

func (s *jsonStorage) WriteFiles(proj *model.Project) error {
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

func (s *jsonStorage) ReadFiles(result *model.Projects, fileName string) error {
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

func (s *jsonStorage) WriteConfig(proj *model.Project) error {
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

func (s *jsonStorage) ReadConfig(result *model.Projects, fileName string) error {
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
