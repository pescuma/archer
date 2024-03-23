package metrics

import (
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/languages/kotlin"
	"github.com/pescuma/archer/lib/languages/kotlin_parser"
	"github.com/pescuma/archer/lib/metrics/complexity"
	"github.com/pescuma/archer/lib/metrics/dependencies"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
)

type Importer struct {
	console consoles.Console
	storage storages.Storage
}

type Options struct {
	Incremental      bool
	MaxImportedFiles *int
	SaveEvery        *time.Duration
}

func NewImporter(console consoles.Console, storage storages.Storage) *Importer {
	return &Importer{
		console: console,
		storage: storage,
	}
}

func (i *Importer) Import(filter []string, opts *Options) error {
	projectsDB, err := i.storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := i.storage.LoadFiles()
	if err != nil {
		return err
	}

	ps, err := filters.ParseAndFilterProjects(projectsDB, filter, model.FilterExcludeExternal)
	if err != nil {
		return err
	}

	ps = lo.Filter(ps, func(p *model.Project, _ int) bool { return len(p.Dirs) > 0 })

	var candidates []*model.File
	if len(filter) == 0 {
		candidates = filesDB.List()
	} else {
		candidates = filesDB.ListByProjects(ps)
	}

	type work struct {
		file    *model.File
		modTime string
	}
	ws := map[string]*work{}

	for _, file := range candidates {
		if !opts.ShouldContinue(len(ws)) {
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

		stat, err := os.Stat(file.Path)
		if err != nil {
			file.Exists = false
			continue
		} else {
			file.SeenAt(time.Now(), stat.ModTime())
		}

		modTime := stat.ModTime().String()

		if opts.Incremental && file.Metrics.GuiceDependencies >= 0 {
			if modTime == file.Data["metrics:last_modified"] {
				continue
			}
		}

		ws[file.Path] = &work{
			file:    file,
			modTime: modTime,
		}
	}

	i.console.Printf("Importing metrics from %v files...\n", len(ws))

	start := time.Now()
	return kotlin.ProcessFiles(lo.Keys(ws),
		func(path string, content kotlin_parser.IKotlinFileContext) error {
			w := ws[path]
			file := w.file

			structure := kotlin.ImportStructure(w.file.Path, content)

			file.Metrics.GuiceDependencies = dependencies.ComputeKotlinGuiceDependencies(file.Path, structure, content)
			file.Metrics.Abstracts = dependencies.ComputeKotlinAbstracts(file.Path, structure, content)

			c := complexity.ComputeKotlinComplexity(file.Path, content)
			file.Metrics.CyclomaticComplexity = c.CyclomaticComplexity
			file.Metrics.CognitiveComplexity = c.CognitiveComplexity

			file.Data["metrics:last_modified"] = w.modTime

			return nil
		},
		func(bar *progressbar.ProgressBar, index int, path string) error {
			if opts.SaveEvery != nil && time.Since(start) > *opts.SaveEvery {
				_ = bar.Clear()
				i.console.Printf("Writing metrics for files...\n")

				err = i.storage.WriteFiles()
				if err != nil {
					return err
				}

				start = time.Now()
			}
			return nil
		},
		func(bar *progressbar.ProgressBar, index int, path string, err error) error {
			file := ws[path].file

			if errors.Is(err, fs.ErrNotExist) {
				file.Exists = false

			} else {
				_ = bar.Clear()
				i.console.Printf("Error processing file %v: %v\n", file.Path, err)
			}

			return nil
		},
	)
}

func (l *Options) ShouldContinue(imported int) bool {
	if l.MaxImportedFiles != nil && imported >= *l.MaxImportedFiles {
		return false
	}

	return true
}
