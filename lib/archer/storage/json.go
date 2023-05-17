package storage

import (
	"encoding/json"

	"github.com/Faire/archer/lib/archer"
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

func BasicInfoToJson(proj *archer.Project) (string, error) {
	jps := jsonBasicInfo{
		Root:      proj.Root,
		Name:      proj.Name,
		NameParts: proj.NameParts,
		Type:      proj.Type,

		RootDir:     proj.RootDir,
		Dirs:        proj.Dirs,
		ProjectFile: proj.ProjectFile,
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func BasicInfoFromJson(result *archer.Projects, content string) error {
	var jps jsonBasicInfo

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.Get(jps.Root, jps.Name)
	proj.NameParts = jps.NameParts
	proj.Type = jps.Type
	proj.RootDir = jps.RootDir
	proj.Dirs = jps.Dirs
	proj.ProjectFile = jps.ProjectFile

	return nil
}

func DepsToJson(proj *archer.Project) (string, error) {
	ds := proj.ListDependencies(archer.FilterAll)

	jps := jsonDeps{
		Root: proj.Root,
		Name: proj.Name,
		Deps: make([]jsonDep, 0, len(ds)),
	}

	for _, d := range ds {
		jp := jsonDep{
			TargetRoot: d.Target.Root,
			TargetName: d.Target.Name,
			Config:     d.Data,
		}

		jps.Deps = append(jps.Deps, jp)
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func DepsFromJson(result *archer.Projects, content string) error {
	var jps jsonDeps

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.Get(jps.Root, jps.Name)
	for _, jp := range jps.Deps {
		target := result.Get(jp.TargetRoot, jp.TargetName)

		d := proj.AddDependency(target)
		d.Data = jp.Config
	}

	return nil
}

func SizeToJson(proj *archer.Project) (string, error) {
	jps := jsonSize{
		Root: proj.Root,
		Name: proj.Name,
		Size: proj.Size,
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func SizeFromJson(result *archer.Projects, content string) error {
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

func ConfigToJson(proj *archer.Project) (string, error) {
	jps := jsonConfig{
		Root:   proj.Root,
		Name:   proj.Name,
		Config: proj.Data,
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func ConfigFromJson(result *archer.Projects, content string) error {
	var jps jsonConfig

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.Get(jps.Root, jps.Name)

	for k, v := range jps.Config {
		proj.SetData(k, v)
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
	Type      archer.ProjectType

	RootDir     string
	Dirs        map[string]string
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
	Size map[string]archer.Size
}

type jsonConfig struct {
	Root   string
	Name   string
	Config map[string]string
}
