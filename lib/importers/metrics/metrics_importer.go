package metrics

import (
	"io/fs"
	"math"
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
	"github.com/pescuma/archer/lib/utils"
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

	peopleDB, err := i.storage.LoadPeople()
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
		candidates = filesDB.ListFiles()
	} else {
		candidates = filesDB.ListFilesByProjects(ps)
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
	err = kotlin.ProcessFiles(lo.Keys(ws),
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
		})
	if err != nil {
		return err
	}

	i.console.Printf("Propagating changes to parents...\n")

	updateParentMetrics(projectsDB, filesDB, peopleDB)

	i.console.Printf("Writing results...\n")

	err = i.storage.WriteProjects()
	if err != nil {
		return err
	}

	err = i.storage.WriteFiles()
	if err != nil {
		return err
	}

	err = i.storage.WritePeople()
	if err != nil {
		return err
	}

	return nil
}

func updateParentMetrics(projectsDB *model.Projects, filesDB *model.Files, peopleDB *model.People) {
	for _, p := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		p.Metrics.Clear()
		for _, d := range p.Dirs {
			d.Metrics.Clear()
		}
	}

	for _, a := range peopleDB.ListProductAreas() {
		a.Metrics.Clear()
	}

	filesByDir := filesDB.GroupFilesByDirectory()

	for _, proj := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		for _, dir := range proj.Dirs {
			for _, file := range filesByDir[dir.ID] {
				file.Metrics.FocusedComplexity = computeFocusedComplexity(file.Size, file.Changes, file.Metrics)

				proj.SeenAt(file.FirstSeen, file.LastSeen)
				dir.SeenAt(file.FirstSeen, file.LastSeen)

				dir.Metrics.Add(file.Metrics)

				if file.ProductAreaID != nil {
					peopleDB.GetProductAreaByID(*file.ProductAreaID).Metrics.Add(file.Metrics)
				}
			}

			proj.Metrics.Add(dir.Metrics)
		}
	}
}

func computeFocusedComplexity(size *model.Size, changes *model.Changes, metrics *model.Metrics) int {
	if size.Lines == 0 {
		return 0
	}

	complexityBase := float64(utils.Max(metrics.CognitiveComplexity, 1))

	deps := utils.Max(metrics.GuiceDependencies, 0)
	depsLimit := 6.
	depsExp := 0.3
	depsFactor := math.Max(math.Pow(float64(deps)/depsLimit, depsExp), 0.1)

	chs := changes.In6Months
	chsLimit := 10.
	chsExp := 0.2
	chsFactor := math.Max(math.Pow(float64(chs)/chsLimit, chsExp), 0.1)

	return int(math.Round(complexityBase * depsFactor * chsFactor))
}

func (l *Options) ShouldContinue(imported int) bool {
	if l.MaxImportedFiles != nil && imported >= *l.MaxImportedFiles {
		return false
	}

	return true
}
