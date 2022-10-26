package archer

import (
	"encoding/json"
	"path/filepath"
)

func ProjNamesToJson(root string, names []string) (string, error) {
	jps := jsonProjNames{
		Root:  root,
		Names: names,
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func ProjNamesFromJson(content string) (string, []string, error) {
	var jps jsonProjNames

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return "", nil, err
	}

	return jps.Root, jps.Names, nil
}

func BasicInfoToJson(proj *Project) (string, error) {
	jps := jsonBasicInfo{
		Root:      proj.Root,
		Name:      proj.Name,
		NameParts: proj.NameParts,
		Type:      proj.Type,

		RootDir:     proj.RootDir,
		Dir:         proj.Dir,
		ProjectFile: proj.ProjectFile,
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func BasicInfoFromJson(result *Projects, content string, fileName string) error {
	var jps jsonBasicInfo

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.Get(jps.Root, jps.Name)
	proj.NameParts = jps.NameParts
	proj.Type = jps.Type
	proj.RootDir = jps.RootDir
	proj.Dir = jps.Dir
	proj.ProjectFile = jps.ProjectFile
	proj.dataDir = filepath.Dir(fileName)

	return nil
}

func DepsToJson(proj *Project) (string, error) {
	ds := proj.ListDependencies(FilterAll)

	jps := jsonDeps{
		Root: proj.Root,
		Name: proj.Name,
		Deps: make([]jsonDep, 0, len(ds)),
	}

	for _, d := range ds {
		jp := jsonDep{
			TargetRoot: d.Target.Root,
			TargetName: d.Target.Name,
			Config:     d.config,
		}

		jps.Deps = append(jps.Deps, jp)
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func DepsFromJson(result *Projects, content string) error {
	var jps jsonDeps

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.Get(jps.Root, jps.Name)
	for _, jp := range jps.Deps {
		target := result.Get(jp.TargetRoot, jp.TargetName)

		d := proj.AddDependency(target)
		d.config = jp.Config
	}

	return nil
}

func SizeToJson(proj *Project) (string, error) {
	jps := jsonSize{
		Root: proj.Root,
		Name: proj.Name,
		Size: proj.size,
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func SizeFromJson(result *Projects, content string) error {
	var jps jsonSize

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.Get(jps.Root, jps.Name)

	for k, v := range jps.Size {
		proj.AddSize(k, v)
	}

	return nil
}

func ConfigToJson(proj *Project) (string, error) {
	jps := jsonConfig{
		Root:   proj.Root,
		Name:   proj.Name,
		Config: proj.config,
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func ConfigFromJson(result *Projects, content string) error {
	var jps jsonConfig

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.Get(jps.Root, jps.Name)

	for k, v := range jps.Config {
		proj.SetConfig(k, v)
	}

	return nil
}

type jsonProjNames struct {
	Root  string
	Names []string
}

type jsonBasicInfo struct {
	Root      string
	Name      string
	NameParts []string
	Type      ProjectType

	RootDir     string
	Dir         string
	ProjectFile string
}

type jsonDeps struct {
	Root string
	Name string
	Deps []jsonDep
}

type jsonDep struct {
	TargetRoot string
	TargetName string
	Config     map[string]string
}

type jsonSize struct {
	Root string
	Name string
	Size map[string]Size
}

type jsonConfig struct {
	Root   string
	Name   string
	Config map[string]string
}
