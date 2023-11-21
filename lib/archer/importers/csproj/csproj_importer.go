package csproj

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/hashicorp/go-set/v2"
	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/importers/common"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
)

type csprojImporter struct {
	rootDir  string
	rootName string
	options  Options
}

type Options struct {
	RespectGitignore bool
}

func NewImporter(rootDir string, rootName string, options Options) archer.Importer {
	return &csprojImporter{
		rootDir:  rootDir,
		rootName: rootName,
		options:  options,
	}
}

func (i *csprojImporter) Import(storage archer.Storage) error {
	fmt.Printf("Loading existing data...\n")

	projsDB, err := storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := storage.LoadFiles()
	if err != nil {
		return err
	}

	err = common.FindAndImportFiles("projects", i.rootDir,
		func(name string) bool {
			return strings.HasSuffix(strings.ToLower(name), ".csproj")
		},
		func(path string) error {
			return i.process(projsDB, filesDB, path)
		})
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

	err = storage.WriteProjects(projsDB)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB)
	if err != nil {
		return err
	}

	return nil
}

func (i *csprojImporter) process(projsDB *model.Projects, filesDB *model.Files, path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var xmlProj xmlProject
	err = xml.Unmarshal(bytes, &xmlProj)
	if err != nil {
		return err
	}

	if xmlProj.Sdk == "" && len(xmlProj.ItemGroups) == 0 {
		fmt.Printf("Ignoring because it is an empty project: %v\n", path)
		return nil
	}

	proj := projsDB.GetOrCreate(i.rootName, i.getProjectName(path))
	proj.Type = model.CodeType
	proj.RootDir = filepath.Dir(path)
	proj.ProjectFile = path
	proj.Dependencies = make(map[string]*model.ProjectDependency)
	proj.SeenAt(time.Now())

	dir := proj.GetDirectory(".")
	dir.Type = model.SourceDir
	dir.SeenAt(time.Now())

	projFile := filesDB.GetOrCreateFile(path)
	projFile.ProjectID = &proj.ID
	projFile.ProjectDirectoryID = &dir.ID
	projFile.SeenAt(time.Now())

	if projFile.RepositoryID != nil {
		proj.RepositoryID = projFile.RepositoryID
	}

	excludes := set.New[string](10)

	filter, err := common.CreateFileFilter(proj.RootDir, i.options.RespectGitignore,
		func(path string) bool {
			return strings.HasSuffix(path, ".csproj") || strings.HasSuffix(path, ".cs")
		},
		excludes.Contains)
	if err != nil {
		return err
	}

	err = common.MarkDeletedFilesAndUnmarkExistingOnes(filesDB, proj, dir, filter)
	if err != nil {
		return err
	}

	for _, item := range xmlProj.ItemGroups {
		for _, pkgRef := range item.PackageReferences {
			i.addPkgDep(projsDB, proj, pkgRef.Include, pkgRef.Version)
		}
		for _, projRef := range item.ProjectReferences {
			err := i.addProjDep(projsDB, proj, projRef.Include)
			if err != nil {
				return err
			}
		}
		for _, ref := range item.References {
			i.addPkgDep(projsDB, proj, ref.Include, "")
		}

		for _, f := range item.Nones {
			if f.Remove != "" {
				remove, err := utils.PathAbs(proj.RootDir, f.Remove)
				if err != nil {
					return err
				}

				excludes.Insert(remove)
			}
		}
	}

	for _, item := range xmlProj.ItemGroups {
		for _, f := range item.Nones {
			err := i.addFiles(filesDB, proj, dir, excludes, f.Include)
			if err != nil {
				return err
			}
		}
		for _, f := range item.EmbeddedResources {
			err := i.addFiles(filesDB, proj, dir, excludes, f.Include)
			if err != nil {
				return err
			}
		}
		for _, f := range item.Compiles {
			err := i.addFiles(filesDB, proj, dir, excludes, f.Include)
			if err != nil {
				return err
			}
		}
		for _, f := range item.ClCompiles {
			err := i.addFiles(filesDB, proj, dir, excludes, f.Include)
			if err != nil {
				return err
			}
		}
		for _, f := range item.ClIncludes {
			err := i.addFiles(filesDB, proj, dir, excludes, f.Include)
			if err != nil {
				return err
			}
		}
	}

	if xmlProj.Sdk == "Microsoft.NET.Sdk" {
		err = common.AddFiles(filesDB, proj, dir, filter)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *csprojImporter) addPkgDep(projsDB *model.Projects, proj *model.Project, pkg string, version string) {
	if pkg == "" {
		return
	}

	dp := projsDB.GetOrCreate(i.rootName, pkg)

	dep := proj.GetOrCreateDependency(dp)
	if version != "" {
		dep.Versions.Insert(version)
	}
}

func (i *csprojImporter) addProjDep(projsDB *model.Projects, proj *model.Project, path string) error {
	if path == "" {
		return nil
	}

	path, err := utils.PathAbs(proj.RootDir, path)
	if err != nil {
		return err
	}

	dp := projsDB.GetOrCreate(i.rootName, i.getProjectName(path))

	proj.GetOrCreateDependency(dp)

	return nil
}

func (i *csprojImporter) addFiles(filesDB *model.Files, proj *model.Project, dir *model.ProjectDirectory, excludes *set.Set[string], path string) error {
	if path == "" {
		return nil
	}

	if strings.Contains(path, "*") || strings.Contains(path, "?") || strings.Contains(path, "[") {
		// '\\' is an escape char
		if filepath.Separator == '\\' {
			path = strings.ReplaceAll(path, "\\", "\\\\")
		}

		g, err := glob.Compile(path, filepath.Separator)
		if err != nil {
			return err
		}

		return filepath.WalkDir(proj.RootDir, func(file string, entry fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return nil

			case entry.IsDir():
				return nil

			default:
				file, err := utils.PathAbs(file)
				if err != nil {
					return err
				}

				end := file[len(proj.RootDir)+1:]

				if g.Match(end) && !excludes.Contains(file) {
					file := filesDB.GetOrCreateFile(file)
					file.ProjectID = &proj.ID
					file.ProjectDirectoryID = &dir.ID
					file.SeenAt(time.Now())
				}
				return nil
			}
		})

	} else {
		path, err := utils.PathAbs(proj.RootDir, path)
		if err != nil {
			return err
		}

		file := filesDB.GetOrCreateFile(path)
		file.ProjectID = &proj.ID
		file.ProjectDirectoryID = &dir.ID
		file.SeenAt(time.Now())
	}

	return nil
}

func (i *csprojImporter) getProjectName(path string) string {
	name := filepath.Base(path)
	name = name[:len(name)-len(filepath.Ext(name))]
	return name
}

type xmlProject struct {
	XMLName    xml.Name       `xml:"Project"`
	Sdk        string         `xml:"Sdk,attr"`
	ItemGroups []xmlItemGroup `xml:"ItemGroup"`
}

type xmlItemGroup struct {
	XMLName           xml.Name              `xml:"ItemGroup"`
	PackageReferences []xmlPackageReference `xml:"PackageReference"`
	ProjectReferences []xmlProjectReference `xml:"ProjectReference"`
	References        []xmlReference        `xml:"Reference"`
	Nones             []xmlNone             `xml:"None"`
	EmbeddedResources []xmlEmbeddedResource `xml:"EmbeddedResource"`
	Compiles          []xmlCompile          `xml:"Compile"`
	ClCompiles        []xmlClCompile        `xml:"ClCompile"`
	ClIncludes        []xmlClInclude        `xml:"ClInclude"`
}

type xmlPackageReference struct {
	XMLName xml.Name `xml:"PackageReference"`
	Include string   `xml:"Include,attr"`
	Version string   `xml:"Version,attr"`
}

type xmlProjectReference struct {
	XMLName xml.Name `xml:"ProjectReference"`
	Include string   `xml:"Include,attr"`
}

type xmlReference struct {
	XMLName xml.Name `xml:"Reference"`
	Include string   `xml:"Include,attr"`
}

type xmlNone struct {
	XMLName xml.Name `xml:"None"`
	Include string   `xml:"Include,attr"`
	Remove  string   `xml:"Remove,attr"`
}

type xmlEmbeddedResource struct {
	XMLName xml.Name `xml:"EmbeddedResource"`
	Include string   `xml:"Include,attr"`
}

type xmlCompile struct {
	XMLName xml.Name `xml:"Compile"`
	Include string   `xml:"Include,attr"`
}

type xmlClCompile struct {
	XMLName xml.Name `xml:"ClCompile"`
	Include string   `xml:"Include,attr"`
}

type xmlClInclude struct {
	XMLName xml.Name `xml:"ClInclude"`
	Include string   `xml:"Include,attr"`
}
