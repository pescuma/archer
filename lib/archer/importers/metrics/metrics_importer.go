package metrics

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/languages"
	"github.com/Faire/archer/lib/archer/languages/kotlin_parser"
	"github.com/Faire/archer/lib/archer/metrics/complexity"
	"github.com/Faire/archer/lib/archer/metrics/dependencies"
	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type metricsImporter struct {
	filters []string
	options Options
}

type Options struct {
	Incremental      bool
	MaxImportedFiles *int
	SaveEvery        *int
}

func NewImporter(filters []string, options Options) archer.Importer {
	return &metricsImporter{
		filters: filters,
		options: options,
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

	reposDB, err := storage.LoadRepositories()
	if err != nil {
		return err
	}

	peopleDB, err := storage.LoadPeople()
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
	modTimes := map[string]string{}
	for _, file := range candidates {
		if !m.options.ShouldContinue(len(files)) {
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
		}

		modTime := stat.ModTime().String()

		if m.options.Incremental && file.Metrics.GuiceDependencies >= 0 {
			if modTime == file.Data["metrics:last_modified"] {
				continue
			}
		}

		files[file.Path] = file
		modTimes[file.Path] = modTime
	}

	fmt.Printf("Importing metrics from %v files...\n", len(files))

	lastSave := 0
	err = languages.ProcessKotlinFiles(lo.Keys(files),
		func(path string, content kotlin_parser.IKotlinFileContext) error {
			file := files[path]

			file.Metrics.GuiceDependencies = dependencies.ComputeKotlinGuiceDependencies(file.Path, content)

			c := complexity.ComputeKotlinComplexity(file.Path, content)
			file.Metrics.CyclomaticComplexity = c.CyclomaticComplexity
			file.Metrics.CognitiveComplexity = c.CognitiveComplexity

			file.Data["metrics:last_modified"] = modTimes[path]

			return nil
		},
		func(bar *progressbar.ProgressBar, index int, path string) error {
			if m.options.SaveEvery != nil && (index+1)-lastSave >= *m.options.SaveEvery {
				lastSave = index

				_ = bar.Clear()
				fmt.Print("Writing metrics for files...")

				err = storage.WriteFiles(filesDB, archer.ChangedMetrics)
				if err != nil {
					return err
				}

				fmt.Print("\r")
				_ = bar.RenderBlank()
			}
			return nil
		},
		func(bar *progressbar.ProgressBar, index int, path string, err error) error {
			file := files[path]

			if errors.Is(err, fs.ErrNotExist) {
				file.Exists = false

			} else {
				_ = bar.Clear()
				fmt.Printf("Error procesing file %v: %v\n", file.Path, err)
			}

			return nil
		})
	if err != nil {
		return err
	}

	fmt.Printf("Importing changes from %v files...\n", len(candidates))

	dirsByIDs := map[model.UUID]*model.ProjectDirectory{}
	for _, p := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		p.Metrics.Clear()
		for _, d := range p.Dirs {
			d.Metrics.Clear()
			dirsByIDs[d.ID] = d
		}
	}
	for _, p := range peopleDB.ListPeople() {
		p.Metrics.Clear()
	}
	for _, t := range peopleDB.ListTeams() {
		t.Metrics.Clear()
	}

	now := time.Now()
	for _, repo := range reposDB.List() {
		for _, c := range repo.ListCommits() {
			inLast6Months := now.Sub(c.Date) < 6*30*24*time.Hour
			addTo := func(metrics *model.Metrics) {
				metrics.ChangesIn6Months += utils.IIf(inLast6Months, 1, 0)
				metrics.ChangesTotal++
			}

			projs := map[model.UUID]bool{}
			dirs := map[model.UUID]bool{}
			teams := map[model.UUID]bool{}
			for _, cf := range c.Files {
				f := filesDB.GetByID(cf.FileID)
				addTo(f.Metrics)

				if f.ProjectID != nil {
					projs[*f.ProjectID] = true
				}
				if f.ProjectDirectoryID != nil {
					teams[*f.ProjectDirectoryID] = true
				}
				if f.TeamID != nil {
					teams[*f.TeamID] = true
				}
			}

			for _, cp := range lo.Keys(projs) {
				addTo(projectsDB.GetByID(cp).Metrics)
			}

			for _, cd := range lo.Keys(dirs) {
				addTo(dirsByIDs[cd].Metrics)
			}

			for _, ct := range lo.Keys(teams) {
				addTo(peopleDB.GetTeamByID(ct).Metrics)
			}

			addTo(peopleDB.GetTeamByID(c.AuthorID).Metrics)
		}
	}

	updateParentMetrics(projectsDB, filesDB, peopleDB)

	fmt.Printf("Writing results...\n")

	err = storage.WriteProjects(projectsDB, archer.ChangedMetrics)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB, archer.ChangedData|archer.ChangedMetrics)
	if err != nil {
		return err
	}

	err = storage.WritePeople(peopleDB, archer.ChangedMetrics)
	if err != nil {
		return err
	}

	return nil
}

func updateParentMetrics(projectsDB *model.Projects, filesDB *model.Files, peopleDB *model.People) {
	filesByDir := filesDB.GroupByDirectory()

	for _, proj := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		for _, dir := range proj.Dirs {
			for _, file := range filesByDir[dir.ID] {
				dir.Metrics.AddIgnoringChanges(file.Metrics)

				if file.TeamID != nil {
					team := peopleDB.GetTeamByID(*file.TeamID)
					team.Metrics.AddIgnoringChanges(file.Metrics)
				}
			}

			proj.Metrics.AddIgnoringChanges(dir.Metrics)
		}
	}
}

func (l *Options) ShouldContinue(imported int) bool {
	if l.MaxImportedFiles != nil && imported >= *l.MaxImportedFiles {
		return false
	}

	return true
}
