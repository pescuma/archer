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
		Root:  proj.Root,
		Name:  proj.Name,
		Sizes: map[string]map[string]*archer.Size{},
	}

	jps.Sizes[""] = proj.Sizes

	for _, dir := range proj.Dirs {
		if dir.Size.IsEmpty() {
			continue
		}

		jps.Sizes[dir.RelativePath] = map[string]*archer.Size{}
		jps.Sizes[dir.RelativePath][""] = dir.Size

		for _, file := range dir.Files {
			if !file.Size.IsEmpty() {
				jps.Sizes[dir.RelativePath][file.RelativePath] = file.Size
			}
		}
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
				} else {
					file := dir.GetFile(filePath)
					file.Size = size
				}
			}
		}
	}

	return nil
}

func FilesToJson(proj *archer.Project) (string, error) {
	jps := jsonFiles{
		Root:    proj.Root,
		Name:    proj.Name,
		RootDir: proj.RootDir,
	}

	for _, dir := range proj.Dirs {
		jd := jsonDir{
			Path: dir.RelativePath,
			Type: dir.Type,
		}

		for _, file := range dir.Files {
			jd.Files = append(jd.Files, jsonFile{
				Path: file.RelativePath,
			})
		}

		jps.Dirs = append(jps.Dirs, jd)
	}

	marshaled, err := json.Marshal(jps)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

func FilesFromJson(result *archer.Projects, content string) error {
	var jps jsonFiles

	err := json.Unmarshal([]byte(content), &jps)
	if err != nil {
		return err
	}

	proj := result.Get(jps.Root, jps.Name)

	for _, jd := range jps.Dirs {
		dir := proj.GetDirectory(jd.Path)
		dir.Type = jd.Type

		for _, jf := range jd.Files {
			dir.GetFile(jf.Path)
		}
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
	Root  string
	Name  string
	Sizes map[string]map[string]*archer.Size
}

type jsonFiles struct {
	Root    string
	Name    string
	RootDir string
	Dirs    []jsonDir
}

type jsonDir struct {
	Path  string
	Type  archer.ProjectDirectoryType
	Files []jsonFile
}

type jsonFile struct {
	Path string
}

type jsonConfig struct {
	Root   string
	Name   string
	Config map[string]string
}
