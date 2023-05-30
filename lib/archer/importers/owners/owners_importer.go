package owners

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type ownersImporter struct {
	filters []string
	options Options
}

type Options struct {
	Incremental      bool
	MaxImportedFiles *int
	SaveEvery        *int
}

func NewImporter(filters []string, options Options) archer.Importer {
	return &ownersImporter{
		filters: filters,
		options: options,
	}
}

func (m *ownersImporter) Import(storage archer.Storage) error {
	config, err := storage.LoadConfig()
	if err != nil {
		return err
	}

	projectsDB, err := storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := storage.LoadFiles()
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

	type work struct {
		file    *model.File
		re      *regexp.Regexp
		modTime string
	}

	var ws []*work
	hasAnyRE := false
	for _, file := range candidates {
		if !m.options.ShouldContinue(len(ws)) {
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
		}

		modTime := stat.ModTime().String()

		if m.options.Incremental && modTime == file.Data["owners:last_modified"] {
			continue
		}

		ws = append(ws, &work{
			file:    file,
			re:      re,
			modTime: modTime,
		})
	}

	if !hasAnyRE {
		fmt.Printf("You need to configure an regexp to find the owners using the ownerRE config key.\n")
		return nil
	}

	fmt.Printf("Importing owners from %v files...\n", len(ws))

	bar := utils.NewProgressBar(len(ws))
	for i, w := range ws {
		bytes, err := os.ReadFile(w.file.Path)
		if err != nil {
			return err
		}

		contents := string(bytes)
		ms := lo.Uniq(lo.Map(w.re.FindAllStringSubmatch(contents, -1), func(m []string, _ int) string { return m[1] }))
		if len(ms) == 1 {
			team := peopleDB.GetOrCreateTeam(ms[0])
			w.file.TeamID = &team.ID

		} else if len(ms) > 1 {
			_ = bar.Clear()
			fmt.Printf("Multiple owners found for '%v': %v\n", w.file.Path, ms)
			_ = bar.RenderBlank()
		}

		w.file.Data["owners:last_modified"] = w.modTime

		if m.options.SaveEvery != nil && (i+1)%*m.options.SaveEvery == 0 {
			_ = bar.Clear()
			fmt.Printf("Writing results...")

			err = storage.WritePeople(peopleDB, archer.ChangedBasicInfo)
			if err != nil {
				return err
			}

			err = storage.WriteFiles(filesDB, archer.ChangedTeams)
			if err != nil {
				return err
			}
		}

		_ = bar.Add(1)
	}

	fmt.Printf("Writing results...\n")

	err = storage.WritePeople(peopleDB, archer.ChangedBasicInfo)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB, archer.ChangedTeams)
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
