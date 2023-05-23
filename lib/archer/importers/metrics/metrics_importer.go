package metrics

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/samber/lo"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/importers"
	"github.com/Faire/archer/lib/archer/languages/kotlin_parser"
	"github.com/Faire/archer/lib/archer/metrics"
	"github.com/Faire/archer/lib/archer/model"
)

type metricsImporter struct {
	filters []string
	limits  Limits
}

type Limits struct {
	Incremental      bool
	MaxImportedFiles *int
}

func NewImporter(filters []string, limits Limits) archer.Importer {
	return &metricsImporter{
		filters: filters,
		limits:  limits,
	}
}

func (m *metricsImporter) Import(storage archer.Storage) error {
	projectsDB, err := storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := storage.LoadFiles()
	if err != nil {
		return err
	}

	ps, err := projectsDB.FilterProjects(m.filters, model.FilterExcludeExternal)
	if err != nil {
		return err
	}

	ps = lo.Filter(ps, func(p *model.Project, _ int) bool { return len(p.Dirs) > 0 })

	var candidates []*model.File
	if len(m.filters) == 0 {
		candidates = filesDB.List()
	} else {
		candidates = filesDB.ListByProjects(ps)
	}

	files := map[string]*model.File{}
	for _, file := range candidates {
		if !m.limits.ShouldContinue(len(files)) {
			break
		}

		if !file.Exists {
			continue
		}

		if strings.Contains(file.Path, "/.idea/") {
			continue
		}

		if !strings.HasSuffix(file.Path, ".kt") {
			continue
		}

		if m.limits.Incremental && file.Metrics.GuiceDependencies >= 0 {
			continue
		}

		files[file.Path] = file
	}

	fmt.Printf("Importing metrics from %v files...\n", len(files))

	_, err = importers.ProcessKotlinFiles(lo.Keys(files),
		func(path string, content kotlin_parser.IKotlinFileContext) (int, error) {
			file := files[path]

			file.Metrics.GuiceDependencies = metrics.ComputeKotlinGuiceDependencies(file.Path, content)

			return 0, nil
		},
		func(path string, err error) error {
			file := files[path]

			if err == fs.ErrNotExist {
				file.Exists = false

			} else if err != nil {
				fmt.Printf("Error procesing file %v: %v\n", file.Path, err)
			}

			return nil
		})
	if err != nil {
		return err
	}

	for _, proj := range ps {
		proj.Metrics.Clear()

		for _, dir := range proj.Dirs {
			dir.Metrics.Clear()

			for _, file := range filesDB.ListByProjectDirectory(dir) {
				dir.Metrics.Add(file.Metrics)
			}

			proj.Metrics.Add(dir.Metrics)
		}
	}

	err = storage.WriteProjects(projectsDB, archer.ChangedMetrics)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB, archer.ChangedMetrics)
	if err != nil {
		return err
	}

	return nil
}

func (l *Limits) ShouldContinue(imported int) bool {
	if l.MaxImportedFiles != nil && imported >= *l.MaxImportedFiles {
		return false
	}

	return true
}
