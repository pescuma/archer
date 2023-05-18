package gradle

import (
	"fmt"
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

func (g *gradleImporter) Import(projs *model.Projects, storage archer.Storage) error {
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

	all := projs.ListProjects(model.FilterAll)
	all = lo.Filter(all, func(p *model.Project, _ int) bool { return p.Root == rootProj })

	err = storage.WriteProjNames(rootProj, lo.Map(all, func(p *model.Project, _ int) string { return p.Name }))
	if err != nil {
		return err
	}

	for _, proj := range all {
		err = storage.WriteBasicInfo(proj)
		if err != nil {
			return err
		}
	}

	for _, proj := range all {
		err = storage.WriteFiles(proj)
		if err != nil {
			return err
		}
	}

	for _, proj := range all {
		err = storage.WriteDeps(proj)
		if err != nil {
			return err
		}
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

func (g *gradleImporter) importBasicInfo(projs *model.Projects, projName string, rootProj string) (bool, error) {
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
	proj.Type = model.CodeType
	proj.RootDir = projDir
	proj.ProjectFile = projFile

	_, err = proj.SetDirectoryAndFiles(projDir, model.ConfigDir, false)
	if err != nil {
		return false, err
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
		_, err = proj.SetDirectoryAndFiles(filepath.Join(projDir, c.Path), c.Type, true)
		if err != nil {
			return false, err
		}
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

func (g *gradleImporter) loadDependencies(projs *model.Projects, projNamesToImport []string, rootProj string) error {
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
