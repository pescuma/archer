package owners

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/filters"
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
	SaveEvery        *int
}

func NewImporter(console consoles.Console, storage storages.Storage) *Importer {
	return &Importer{
		console: console,
		storage: storage,
	}
}

func (i *Importer) Import(filter []string, opts *Options) error {
	config, err := i.storage.LoadConfig()
	if err != nil {
		return err
	}

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
		re      *regexp.Regexp
	}

	var ws []*work
	hasAnyRE := false
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

		var proj *model.Project
		if file.ProjectID != nil {
			proj = projectsDB.GetByID(*file.ProjectID)
		}

		re, err := findRE(config, proj, filepath.Ext(file.Path))
		if err != nil {
			return err
		}

		if re == nil {
			continue
		}

		hasAnyRE = true

		stat, err := os.Stat(file.Path)
		if err != nil {
			file.Exists = false
			continue
		} else {
			file.SeenAt(time.Now(), stat.ModTime())
		}

		modTime := stat.ModTime().String()

		if opts.Incremental && modTime == file.Data["owners:last_modified"] {
			continue
		}

		ws = append(ws, &work{
			file:    file,
			modTime: modTime,
			re:      re,
		})
	}

	if !hasAnyRE {
		fmt.Printf("You need to configure an regexp to find the owners using the ownerRE config key.\n")
		return nil
	}

	fmt.Printf("Importing owners from %v files...\n", len(ws))

	bar := utils.NewProgressBar(len(ws))
	for j, w := range ws {
		bytes, err := os.ReadFile(w.file.Path)
		if err != nil {
			return err
		}

		contents := string(bytes)
		ms := lo.Uniq(lo.Map(w.re.FindAllStringSubmatch(contents, -1), func(m []string, _ int) string { return m[1] }))
		if len(ms) == 1 {
			area := peopleDB.GetOrCreateProductArea(ms[0])
			w.file.ProductAreaID = &area.ID

		} else if len(ms) > 1 {
			_ = bar.Clear()
			fmt.Printf("Multiple owners found for '%v': %v\n", w.file.Path, ms)
			_ = bar.RenderBlank()
		}

		w.file.Data["owners:last_modified"] = w.modTime

		if opts.SaveEvery != nil && (j+1)%*opts.SaveEvery == 0 {
			_ = bar.Clear()
			fmt.Printf("Writing results...")

			err = i.storage.WritePeople()
			if err != nil {
				return err
			}

			err = i.storage.WriteFiles()
			if err != nil {
				return err
			}
		}

		_ = bar.Add(1)
	}

	fmt.Printf("Writing results...\n")

	err = i.storage.WritePeople()
	if err != nil {
		return err
	}

	err = i.storage.WriteFiles()
	if err != nil {
		return err
	}

	return nil
}

func findRE(config *map[string]string, proj *model.Project, ext string) (*regexp.Regexp, error) {
	if proj != nil {
		if re, ok := proj.Data["ownerRE:"+ext]; ok {
			return regexp.Compile(re)
		}

		if re, ok := proj.Data["ownerRE"]; ok {
			return regexp.Compile(re)
		}
	}

	if re, ok := (*config)["ownerRE:"+ext]; ok {
		return regexp.Compile(re)
	}

	if re, ok := (*config)["ownerRE"]; ok {
		return regexp.Compile(re)
	}

	return nil, nil
}

func (l *Options) ShouldContinue(imported int) bool {
	if l.MaxImportedFiles != nil && imported >= *l.MaxImportedFiles {
		return false
	}

	return true
}
