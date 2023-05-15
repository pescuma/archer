package storage

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Faire/archer/lib/archer"
)

const (
	projNamesJson = "projNames.json"
	depsJson      = "deps.json"
	sizeJson      = "size.json"
	basicInfoJson = "proj.json"
	configJson    = "config.json"
)

type FilesStorage struct {
	root string
}

func NewFilesStorage(root string) (archer.Storage, error) {
	return &FilesStorage{
		root: root,
	}, nil
}

func (s *FilesStorage) LoadProjects(result *archer.Projects) error {
	return filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		switch d.Name() {
		case basicInfoJson:
			err = s.ReadBasicInfo(result, path)
			if err != nil {
				return err
			}

		case configJson:
			err = s.ReadConfig(result, path)
			if err != nil {
				return err
			}

		case depsJson:
			err = s.ReadDeps(result, path)
			if err != nil {
				return err
			}

		case sizeJson:
			err := s.ReadSize(result, path)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *FilesStorage) getProjNamesFileName(root string) (string, error) {
	return filepath.Abs(filepath.Join(
		s.root,
		strings.ReplaceAll(root, ":", "_"),
		projNamesJson,
	))
}

func (s *FilesStorage) WriteProjNames(projRoot string, projNames []string) error {
	fileName, err := s.getProjNamesFileName(s.root)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	content, err := archer.ProjNamesToJson(projRoot, projNames)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(content), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *FilesStorage) ReadProjNames() ([]string, error) {
	fileName, err := s.getProjNamesFileName(s.root)
	if err != nil {
		return nil, err
	}

	contents, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	_, result, err := archer.ProjNamesFromJson(string(contents))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *FilesStorage) computeDataDir(proj *archer.Project) error {
	dir, err := filepath.Abs(filepath.Join(
		s.root,
		proj.Root,
		strings.TrimLeft(strings.ReplaceAll(proj.Name, ":", string(os.PathSeparator)), string(os.PathSeparator)),
	))
	if err != nil {
		return err
	}

	proj.DataDir = dir

	return nil
}

func (s *FilesStorage) getDepsFileName(proj *archer.Project) (string, error) {
	err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(proj.DataDir, depsJson), nil
}

func (s *FilesStorage) WriteDeps(proj *archer.Project) error {
	fileName, err := s.getDepsFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := archer.DepsToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *FilesStorage) ReadDeps(result *archer.Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = archer.DepsFromJson(result, string(contents))
	if err != nil {
		return err
	}

	return nil
}

func (s *FilesStorage) getSizeFileName(proj *archer.Project) (string, error) {
	err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(proj.DataDir, sizeJson), nil
}

func (s *FilesStorage) WriteSize(proj *archer.Project) error {
	fileName, err := s.getSizeFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := archer.SizeToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *FilesStorage) ReadSize(result *archer.Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = archer.SizeFromJson(result, string(contents))
	if err != nil {
		return err
	}

	return nil
}

func (s *FilesStorage) getBasicInfoFileName(proj *archer.Project) (string, error) {
	err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(proj.DataDir, basicInfoJson), nil
}

func (s *FilesStorage) WriteBasicInfo(proj *archer.Project) error {
	fileName, err := s.getBasicInfoFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := archer.BasicInfoToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *FilesStorage) ReadBasicInfo(result *archer.Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = archer.BasicInfoFromJson(result, string(contents), fileName)
	if err != nil {
		return err
	}

	return nil
}

func (s *FilesStorage) getConfigFileName(proj *archer.Project) (string, error) {
	err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(proj.DataDir, configJson), nil
}

func (s *FilesStorage) WriteConfig(proj *archer.Project) error {
	fileName, err := s.getConfigFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := archer.ConfigToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *FilesStorage) ReadConfig(result *archer.Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = archer.ConfigFromJson(result, string(contents))
	if err != nil {
		return err
	}

	return nil
}
