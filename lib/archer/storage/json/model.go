package json

import (
	"encoding/json"

	"github.com/Faire/archer/lib/archer/model"
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

func ProjBasicInfoToJson(proj *model.Project) (string, error) {
	jps := jsonBasicInfo{
		Root:      proj.Root,
		Name:      proj.Name,
		ID:        proj.ID,
		NameParts: proj.NameParts,
		Type:      proj.Type,

		RootDir:     proj.RootDir,
		ProjectFile: proj.ProjectFile,
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func ProjBasicInfoFromJson(result *model.Projects, content string) error {
	var jps jsonBasicInfo

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.GetOrCreate(jps.Root, jps.Name)
	proj.ID = jps.ID
	proj.NameParts = jps.NameParts
	proj.Type = jps.Type
	proj.RootDir = jps.RootDir
	proj.ProjectFile = jps.ProjectFile

	return nil
}

func ProjDepsToJson(proj *model.Project) (string, error) {
	ds := proj.ListDependencies(model.FilterAll)

	jps := jsonDeps{
		Root: proj.Root,
		Name: proj.Name,
		Deps: make([]jsonDep, 0, len(ds)),
	}

	for _, d := range ds {
		jp := jsonDep{
			TargetRoot: d.Target.Root,
			TargetName: d.Target.Name,
			ID:         d.ID,
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

func ProjDepsFromJson(result *model.Projects, content string) error {
	var jps jsonDeps

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.GetOrCreate(jps.Root, jps.Name)
	for _, jp := range jps.Deps {
		target := result.GetOrCreate(jp.TargetRoot, jp.TargetName)

		d := proj.GetDependency(target)
		d.ID = jp.ID
		d.Data = jp.Config
	}

	return nil
}

func SizeToJson(proj *model.Project) (string, error) {
	jps := jsonSize{
		Root:  proj.Root,
		Name:  proj.Name,
		Sizes: map[string]map[string]*model.Size{},
	}

	jps.Sizes[""] = proj.Sizes

	for _, dir := range proj.Dirs {
		if dir.Size.IsEmpty() {
			continue
		}

		jps.Sizes[dir.RelativePath] = map[string]*model.Size{}
		jps.Sizes[dir.RelativePath][""] = dir.Size
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func SizeFromJson(result *model.Projects, content string) error {
	var jps jsonSize

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.GetOrCreate(jps.Root, jps.Name)

	for dirPath, files := range jps.Sizes {
		if dirPath == "" {
			for k, v := range files {
				proj.AddSize(k, v)
			}

		} else {
			dir := proj.GetDirectory(dirPath)

			for filePath, size := range files {
				if filePath == "" {
					dir.Size = size
				}
			}
		}
	}

	return nil
}

func FilesToJson(proj *model.Project) (string, error) {
	jps := jsonFiles{
		Root:    proj.Root,
		Name:    proj.Name,
		RootDir: proj.RootDir,
	}

	for _, dir := range proj.Dirs {
		jd := jsonDir{
			Path: dir.RelativePath,
			Type: dir.Type,
			ID:   dir.ID,
		}

		jps.Dirs = append(jps.Dirs, jd)
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func FilesFromJson(result *model.Projects, content string) error {
	var jps jsonFiles

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.GetOrCreate(jps.Root, jps.Name)

	for _, jd := range jps.Dirs {
		dir := proj.GetDirectory(jd.Path)
		dir.Type = jd.Type
	}

	return nil
}

func ConfigToJson(proj *model.Project) (string, error) {
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

func ConfigFromJson(result *model.Projects, content string) error {
	var jps jsonConfig

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.GetOrCreate(jps.Root, jps.Name)

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
	Type      model.ProjectType
	ID        model.UUID

	RootDir     string
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
	ID         model.UUID

	Config map[string]string
}

type jsonSize struct {
	Root  string
	Name  string
	Sizes map[string]map[string]*model.Size
}

type jsonFiles struct {
	Root    string
	Name    string
	RootDir string
	Dirs    []jsonDir
}

type jsonDir struct {
	Path string
	Type model.ProjectDirectoryType
	ID   model.UUID
}

type jsonFile struct {
	Path string
	ID   model.UUID
}

type jsonConfig struct {
	Root   string
	Name   string
	Config map[string]string
}
