package gradle

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type gradleImporter struct {
	rootDir string
}

func NewImporter(rootDir string) archer.Importer {
	return &gradleImporter{
		rootDir: rootDir,
	}
}

func (g *gradleImporter) Import(storage archer.Storage) error {
	projs, err := storage.LoadProjects()
	if err != nil {
		return err
	}

	files, err := storage.LoadFiles()
	if err != nil {
		return err
	}

	fmt.Printf("Listing projects...\n")

	queue, err := g.importProjectNames()
	if err != nil {
		return err
	}

	fmt.Printf("Importing basic info from %v projects...\n", len(queue))

	rootProj := queue[0]

	bar := utils.NewProgressBar(len(queue))
	for _, p := range queue {
		err = g.importBasicInfo(projs, p, rootProj)
		if err != nil {
			return err
		}

		_ = bar.Add(1)
	}

	fmt.Printf("Importing files from %v projects...\n", len(queue))

	bar = utils.NewProgressBar(len(queue))
	for _, p := range queue {
		proj := projs.GetOrCreate(rootProj, p)

		err = g.importDirectories(files, proj)
		if err != nil {
			return err
		}

		_ = bar.Add(1)
	}

	fmt.Printf("Importing dependencies from %v projects...\n", len(queue))

	bar = utils.NewProgressBar(len(queue))
	block := 100
	projsInsideRoot := lo.Associate(queue, func(s string) (string, bool) { return s, true })
	for i := 0; i < len(queue); i += block {
		piece := utils.Take(queue[i:], block)

		err = g.loadDependencies(projs, piece, rootProj, projsInsideRoot)
		if err != nil {
			return err
		}

		_ = bar.Add(len(piece))
	}

	for _, p := range queue {
		proj := projs.GetOrCreate(rootProj, p)

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

	err = storage.WriteProjects(projs, archer.ChangedBasicInfo|archer.ChangedData|archer.ChangedDependencies)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(files, archer.ChangedBasicInfo)
	if err != nil {
		return err
	}

	return nil
}

func (g *gradleImporter) importProjectNames() ([]string, error) {
	projNames, err := listProjects(g.rootDir)
	if err != nil {
		return nil, err
	}

	return projNames, nil
}

func (g *gradleImporter) importBasicInfo(projs *model.Projects, projName string, rootProj string) error {
	projDir, err := g.getProjectDir(projName)
	if err != nil {
		return err
	}

	projFile, err := g.getProjectFile(projName)
	if err != nil {
		return err
	}

	proj := projs.GetOrCreate(rootProj, projName)
	proj.NameParts = utils.IIf(projName == rootProj, []string{rootProj}, strings.Split(projName[1:], ":"))
	proj.Root = rootProj
	proj.Type = model.CodeType
	proj.RootDir = projDir
	proj.ProjectFile = projFile

	return nil
}

func (g *gradleImporter) importDirectories(files *model.Files, proj *model.Project) error {
	err := g.importDirectory(files, proj, proj.RootDir, model.ConfigDir, false)
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
		err = g.importDirectory(files, proj, filepath.Join(proj.RootDir, c.Path), c.Type, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *gradleImporter) importDirectory(files *model.Files, proj *model.Project, dirPath string, dirType model.ProjectDirectoryType, recursive bool) error {
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
		}

		file := files.GetOrCreate(path)
		file.ProjectID = &proj.ID
		file.ProjectDirectoryID = &dir.ID

		return nil
	})
}

func (g *gradleImporter) getProjectDir(projName string) (string, error) {
	if projName[0] == ':' {
		return filepath.Abs(filepath.Join(
			g.rootDir,
			strings.ReplaceAll(projName[1:], ":", string(os.PathSeparator)),
		))

	} else {
		return g.rootDir, nil
	}
}

func (g *gradleImporter) getProjectFile(projName string) (string, error) {
	dir, err := g.getProjectDir(projName)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "build.gradle.") {
			return filepath.Join(dir, e.Name()), nil
		}
	}

	return "", nil
}

func (g *gradleImporter) loadDependencies(projs *model.Projects, projNamesToImport []string, rootProj string, projsInsideRoot map[string]bool) error {
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

	cmd := exec.Command(filepath.Join(g.rootDir, "gradlew"), args...)
	cmd.Dir = g.rootDir

	output, err := cmd.Output()
	if err != nil {
		return err
	}

	opt := splitOutputPerTarget(string(output))

	for _, o := range opt {
		err = parseDeps(projs, o, rootProj, projsInsideRoot)
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

func (g *gradleImporter) needsUpdate(projFile string, depsJson string) (bool, error) {
	sp, err := os.Stat(projFile)
	if err != nil {
		// No project file means no deps
		return false, nil
	}

	sd, err := os.Stat(depsJson)

	return err != nil || sp.ModTime().After(sd.ModTime()), nil
}
