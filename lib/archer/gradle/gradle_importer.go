package gradle

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hhatto/gocloc"
	"github.com/pkg/errors"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/utils"
)

type gradleImporter struct {
	rootDir string
	storage archer.Storage
}

func NewImporter(rootDir string) archer.Importer {
	return &gradleImporter{
		rootDir: rootDir,
	}
}

func (g *gradleImporter) Import(projs *archer.Projects, storage archer.Storage) error {
	g.storage = storage

	fmt.Printf("Listing projects...\n")

	queue, err := g.importProjectNames()
	if err != nil {
		return err
	}

	fmt.Printf("Going to import basic info from %v projects...\n", len(queue))

	rootProj := queue[0]

	for i, p := range queue {
		changed, err := g.importBasicInfo(projs, p, rootProj)
		if err != nil {
			return err
		}

		prefix := fmt.Sprintf("[%v / %v] ", i, len(queue))

		if !changed {
			fmt.Printf("%vSkipped import of basic info from '%v'\n", prefix, p)
		} else {
			fmt.Printf("%vImported basic info from '%v'\n", prefix, p)
		}
	}

	fmt.Printf("Going to import dependencies from %v projects...\n", len(queue))

	block := 100
	for i := 0; i < len(queue); {
		piece := utils.Take(queue[i:], block)

		for _, p := range piece {
			i++
			prefix := fmt.Sprintf("[%v / %v] ", i, len(queue))
			fmt.Printf("%vImporting dependencies from '%v' ...\n", prefix, p)
		}

		err = g.loadDependencies(projs, piece, rootProj)
		if err != nil {
			return err
		}
	}

	for _, p := range queue {
		proj := projs.Get(rootProj, p)

		for _, d := range proj.ListDependencies(archer.FilterAll) {
			if d.Source.IsCode() && strings.HasSuffix(d.Source.Name, "-api") {
				d.SetConfig("source", strings.TrimSuffix(d.Source.Name, "-api"))
			}
			if d.Target.IsCode() && strings.HasSuffix(d.Target.Name, "-api") {
				d.SetConfig("target", strings.TrimSuffix(d.Target.Name, "-api"))
				d.SetConfig("type", "api")
				d.SetConfig("style", "dashed")
			}
		}

		err = g.storage.WriteDeps(proj)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Going to import lines of code from %v projects...\n", len(queue))

	for i, p := range queue {
		changed, err := g.importSize(projs, rootProj, p)
		if err != nil {
			return err
		}

		prefix := fmt.Sprintf("[%v / %v] ", i, len(queue))

		if !changed {
			fmt.Printf("%vSkipped import of lines of code from '%v'\n", prefix, p)
		} else {
			fmt.Printf("%vImported lines of code from '%v'\n", prefix, p)
		}
	}

	return nil
}

func (g *gradleImporter) importProjectNames() ([]string, error) {
	projNames, err := listProjects(g.rootDir)
	if err != nil {
		return nil, err
	}

	err = g.storage.WriteProjNames(projNames[0], projNames)
	if err != nil {
		return nil, err
	}

	return projNames, nil
}

func (g *gradleImporter) importBasicInfo(projs *archer.Projects, projName string, rootProj string) (bool, error) {
	projDir, err := g.getProjectDir(projName)
	if err != nil {
		return false, err
	}

	projFile, err := g.getProjectFile(projName)
	if err != nil {
		return false, err
	}

	proj := projs.Get(rootProj, projName)
	proj.NameParts = utils.IIf(projName == rootProj, []string{rootProj}, strings.Split(projName[1:], ":"))
	proj.Root = rootProj
	proj.Type = archer.CodeType
	proj.RootDir = g.rootDir
	proj.ProjectFile = projFile

	proj.Dir = filepath.Join(projDir, "src")
	if _, err = os.Stat(proj.Dir); err != nil {
		proj.Dir = ""
	}

	err = g.storage.WriteBasicInfo(proj)
	if err != nil {
		return false, err
	}

	return true, nil
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

func (g *gradleImporter) loadDependencies(projs *archer.Projects, projNamesToImport []string, rootProj string) error {
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
		err = parseDeps(projs, o, rootProj)
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

func (g *gradleImporter) importSize(projs *archer.Projects, rootProj, projName string) (bool, error) {
	proj := projs.Get(rootProj, projName)

	if proj.Dir == "" {
		return false, nil
	}

	es, err := os.ReadDir(proj.Dir)
	if err != nil {
		return false, err
	}
	for _, e := range es {
		if !e.IsDir() {
			continue
		}

		size, err := g.computeCLOC(filepath.Join(proj.Dir, e.Name()))
		if err != nil {
			return false, err
		}

		proj.AddSize(e.Name(), *size)
	}

	err = g.storage.WriteSize(proj)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (g *gradleImporter) computeCLOC(path string) (*archer.Size, error) {
	result := archer.Size{
		Other: map[string]int{},
	}

	_, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	}

	languages := gocloc.NewDefinedLanguages()
	options := gocloc.NewClocOptions()
	paths := []string{
		path,
	}

	processor := gocloc.NewProcessor(languages, options)
	loc, err := processor.Analyze(paths)
	if err != nil {
		return nil, errors.Wrapf(err, "error computing lones of code")
	}

	files := 0
	bytes := 0
	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}

			files += 1
			bytes += int(info.Size())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	result.Bytes = bytes
	result.Files = files
	result.Other["Code"] = int(loc.Total.Code)
	result.Other["Comments"] = int(loc.Total.Comments)
	result.Other["Blanks"] = int(loc.Total.Blanks)
	result.Lines = int(loc.Total.Code + loc.Total.Comments + loc.Total.Blanks)

	return &result, nil
}
