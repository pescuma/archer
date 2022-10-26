package archer

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	projNamesJson = "projNames.json"
	depsJson      = "deps.json"
	sizeJson      = "size.json"
	basicInfoJson = "proj.json"
	configJson    = "config.json"
)

type Storage struct {
	root string
}

func NewStorage(root string) (*Storage, error) {
	return &Storage{
		root: root,
	}, nil
}

func (s *Storage) LoadProjects(result *Projects) error {
	return filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		switch d.Name() {
		case basicInfoJson:
			err = s.ReadBasicInfoFile(result, path)
			if err != nil {
				return err
			}

		case configJson:
			err = s.ReadConfigFile(result, path)
			if err != nil {
				return err
			}

		case depsJson:
			err = s.ReadDepsFile(result, path)
			if err != nil {
				return err
			}

		case sizeJson:
			err := s.ReadSizeFile(result, path)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Storage) GetProjNamesFileName(root string) (string, error) {
	return filepath.Abs(filepath.Join(
		s.root,
		strings.ReplaceAll(root, ":", "_"),
		projNamesJson,
	))
}

func (s *Storage) WriteProjNamesFile(fileName string, projRoot string, projNames []string) error {
	err := os.MkdirAll(filepath.Dir(fileName), 0o700)
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

func (s *Storage) ReadProjNamesFile(fileName string) ([]string, error) {
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

func (s *Storage) computeDataDir(proj *Project) error {
	dir, err := filepath.Abs(filepath.Join(
		s.root,
		proj.Root,
		strings.TrimLeft(strings.ReplaceAll(proj.Name, ":", string(os.PathSeparator)), string(os.PathSeparator)),
	))
	if err != nil {
		return err
	}

	proj.dataDir = dir

	return nil
}

func (s *Storage) GetDepsFileName(proj *Project) (string, error) {
	err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(proj.dataDir, depsJson), nil
}

func (s *Storage) WriteDepsFile(proj *Project) error {
	fileName, err := s.GetDepsFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := DepsToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) ReadDepsFile(result *Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = DepsFromJson(result, string(contents))
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetSizeFileName(proj *Project) (string, error) {
	err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(proj.dataDir, sizeJson), nil
}

func (s *Storage) WriteSizeFile(proj *Project) error {
	fileName, err := s.GetSizeFileName(proj)
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

func (s *Storage) ReadSizeFile(result *Projects, fileName string) error {
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

func (s *Storage) GetBasicInfoFileName(proj *Project) (string, error) {
	err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(proj.dataDir, basicInfoJson), nil
}

func (s *Storage) WriteBasicInfoFile(proj *Project) error {
	fileName, err := s.GetBasicInfoFileName(proj)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0o700)
	if err != nil {
		return err
	}

	jc, err := BasicInfoToJson(proj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, []byte(jc), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) ReadBasicInfoFile(result *Projects, fileName string) error {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = BasicInfoFromJson(result, string(contents), fileName)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetConfigFileName(proj *Project) (string, error) {
	err := s.computeDataDir(proj)
	if err != nil {
		return "", err
	}

	return filepath.Join(proj.dataDir, configJson), nil
}

func (s *Storage) WriteConfigFile(proj *Project) error {
	fileName, err := s.GetConfigFileName(proj)
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

func (s *Storage) ReadConfigFile(result *Projects, fileName string) error {
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
