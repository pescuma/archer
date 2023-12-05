package gradle

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/storages"

	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

type Importer struct {
	console consoles.Console
	storage storages.Storage
}

func NewImporter(console consoles.Console, storage storages.Storage) *Importer {
	return &Importer{
		console: console,
		storage: storage,
	}
}

func (i *Importer) Import(rootDir string) error {
	projsDB, err := i.storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := i.storage.LoadFiles()
	if err != nil {
		return err
	}

	fmt.Printf("Listing projects...\n")

	queue, err := listProjects(rootDir)
	if err != nil {
		return err
	}

	fmt.Printf("Importing basic info from %v projects...\n", len(queue))

	rootProj := queue[0]

	bar := utils.NewProgressBar(len(queue))
	for _, p := range queue {
		err = i.importBasicInfo(rootDir, projsDB, filesDB, p, rootProj)
		if err != nil {
			return err
		}

		_ = bar.Add(1)
	}

	fmt.Printf("Importing files from %v projects...\n", len(queue))

	bar = utils.NewProgressBar(len(queue))
	for _, p := range queue {
		proj := projsDB.GetOrCreate(p)

		err = i.importDirectories(filesDB, proj)
		if err != nil {
			return err
		}

		_ = bar.Add(1)
	}

	fmt.Printf("Importing dependencies from %v projects...\n", len(queue))

	bar = utils.NewProgressBar(len(queue))
	block := 100
	projsInsideRoot := lo.Associate(queue, func(s string) (string, bool) { return s, true })
	for j := 0; j < len(queue); j += block {
		piece := utils.Take(queue[j:], block)

		err = i.loadDependencies(rootDir, projsDB, piece, rootProj, projsInsideRoot)
		if err != nil {
			return err
		}

		_ = bar.Add(len(piece))
	}

	for _, p := range queue {
		proj := projsDB.GetOrCreate(p)

		for _, d := range proj.ListDependencies(model.FilterAll) {
			if d.Source.IsCode() && strings.HasSuffix(d.Source.Name, "-api") {
				d.SetData("source", strings.TrimSuffix(d.Source.Name, "-api"))
			}
			if d.Target.IsCode() && strings.HasSuffix(d.Target.Name, "-api") {
				d.SetData("target", strings.TrimSuffix(d.Target.Name, "-api"))
				d.SetData("type", "api")
				d.SetData("style", "dashed")
			}
		}
	}

	fmt.Printf("Writing results...\n")

	err = i.storage.WriteProjects()
	if err != nil {
		return err
	}

	err = i.storage.WriteFiles()
	if err != nil {
		return err
	}

	return nil
}

func (i *Importer) importBasicInfo(rootDir string, projsDB *model.Projects, filesDB *model.Files, projName string, rootProj string) error {
	projDir, err := i.getProjectDir(rootDir, projName)
	if err != nil {
		return err
	}

	projFileName, err := i.getProjectFile(rootDir, projName)
	if err != nil {
		return err
	}

	proj := projsDB.GetOrCreate(projName)
	proj.Groups = utils.IIf(projName == rootProj,
		[]string{rootProj},
		simplifyPrefixes(append([]string{rootProj}, strings.Split(projName[1:], ":")...)))
	proj.Type = model.CodeType
	proj.RootDir = projDir
	proj.ProjectFile = projFileName
	proj.SeenAt(time.Now())

	projFile := filesDB.GetOrCreateFile(projFileName)
	projFile.ProjectID = &proj.ID
	projFile.SeenAt(time.Now())

	if projFile.RepositoryID != nil {
		proj.RepositoryID = projFile.RepositoryID
	}

	return nil
}

func (i *Importer) importDirectories(files *model.Files, proj *model.Project) error {
	err := i.importDirectory(files, proj, proj.RootDir, model.ConfigDir, false)
	if err != nil {
		return err
	}

	var candidates = []struct {
		Path string
		Type model.ProjectDirectoryType
	}{
		{"config", model.ConfigDir},
		{"src/main/kotlin", model.SourceDir},
		{"src/test/kotlin", model.TestsDir},
		{"src/main/java", model.SourceDir},
		{"src/test/java", model.TestsDir},
	}

	for _, c := range candidates {
		dirPath, err := utils.PathAbs(proj.RootDir, c.Path)
		if err != nil {
			return err
		}

		err = i.importDirectory(files, proj, dirPath, c.Type, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Importer) importDirectory(files *model.Files, proj *model.Project, dirPath string, dirType model.ProjectDirectoryType, recursive bool) error {
	dirPath, err := utils.PathAbs(dirPath)
	if err != nil {
		return nil
	}

	_, err = os.Stat(dirPath)
	if err != nil {
		return nil
	}

	var dir *model.ProjectDirectory
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			} else {
				return utils.IIf(recursive, nil, filepath.SkipDir)
			}
		}

		if dir == nil {
			rootRel, err := filepath.Rel(proj.RootDir, dirPath)
			if err != nil {
				return err
			}

			dir = proj.GetDirectory(rootRel)
			dir.Type = dirType
			dir.SeenAt(time.Now())
		}

		file := files.GetOrCreateFile(path)
		file.ProjectID = &proj.ID
		file.ProjectDirectoryID = &dir.ID
		file.SeenAt(time.Now())

		return nil
	})
}

func (i *Importer) getProjectDir(rootDir, projName string) (string, error) {
	if projName[0] == ':' {
		return utils.PathAbs(
			rootDir,
			strings.ReplaceAll(projName[1:], ":", string(os.PathSeparator)),
		)

	} else {
		return rootDir, nil
	}
}

func (i *Importer) getProjectFile(rootDir, projName string) (string, error) {
	dir, err := i.getProjectDir(rootDir, projName)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "build.gradle.") {
			return utils.PathAbs(dir, e.Name())
		}
	}

	return "", nil
}

func (i *Importer) loadDependencies(rootDir string, projs *model.Projects, projNamesToImport []string, rootProj string, projsInsideRoot map[string]bool) error {
	args := make([]string, 0, 3*len(projNamesToImport))
	for _, projName := range projNamesToImport {
		var target string
		if strings.HasPrefix(projName, ":") {
			target = projName + ":dependencies"
		} else {
			target = "dependencies"
		}

		args = append(args, target, "--configuration", "compileClasspath")
	}

	cmd := exec.Command(filepath.Join(rootDir, "gradlew"), args...)
	cmd.Dir = rootDir

	output, err := cmd.Output()
	if err != nil {
		return err
	}

	opt := splitOutputPerTarget(string(output))

	for _, o := range opt {
		err = parseDeps(projs, o, projsInsideRoot)
		if err != nil {
			return err
		}
	}

	return nil
}

func splitOutputPerTarget(output string) map[string]string {
	result := map[string]string{}

	re := regexp.MustCompile(`^> Task (:[^ ]+)$`)
	task := ""
	start := -1

	lines := strings.Split(output, "\n")
	for i, l := range lines {
		m := re.FindStringSubmatch(l)
		if m != nil {
			if start != -1 {
				result[task] = strings.Join(lines[start:i], "\n")
			}

			task = strings.TrimSuffix(m[1], ":dependencies")
			start = i + 1
			continue
		}
	}

	if start != -1 {
		result[task] = strings.Join(lines[start:], "\n")
	}

	return result
}

func (i *Importer) needsUpdate(projFile string, depsJson string) (bool, error) {
	sp, err := os.Stat(projFile)
	if err != nil {
		// No project file means no deps
		return false, nil
	}

	sd, err := os.Stat(depsJson)

	return err != nil || sp.ModTime().After(sd.ModTime()), nil
}

func simplifyPrefixes(parts []string) []string {
	for len(parts) > 1 && strings.HasPrefix(parts[1], parts[0]) {
		parts = parts[1:]
	}
	return parts
}
