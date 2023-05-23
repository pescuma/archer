package loc

import (
	"fmt"
	"os"

	"github.com/hhatto/gocloc"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type locImporter struct {
	filters []string
}

func NewImporter(filters []string) archer.Importer {
	return &locImporter{
		filters: filters,
	}
}

func (l *locImporter) Import(storage archer.Storage) error {
	projectsDB, err := storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := storage.LoadFiles()
	if err != nil {
		return err
	}

	ps, err := projectsDB.FilterProjects(l.filters, model.FilterExcludeExternal)
	if err != nil {
		return err
	}

	ps = lo.Filter(ps, func(p *model.Project, _ int) bool { return len(p.Dirs) > 0 })

	var candidates []*model.File
	if len(l.filters) == 0 {
		candidates = filesDB.List()
	} else {
		candidates = filesDB.ListByProjects(ps)
	}

	fmt.Printf("Importing size from %v files...\n", len(candidates))

	bar := utils.NewProgressBar(len(candidates))
	var fs = make([]*model.File, 0, len(candidates))
	for _, c := range candidates {
		info, err := os.Stat(c.Path)
		if err == nil {
			c.Exists = true

			c.Size.Clear()
			c.Size.Files = 1
			c.Size.Bytes = int(info.Size())

			fs = append(fs, c)
		} else {
			c.Exists = false

			c.Size.Clear()
		}

		_ = bar.Add(1)
	}

	fmt.Printf("Importing lines of code from %v files...\n", len(fs))

	loc, err := l.computeLOC(fs)
	if err != nil {
		return err
	}

	dirsByID := map[model.UUID]*model.ProjectDirectory{}

	for _, proj := range ps {
		for _, dir := range proj.Dirs {
			dir.Size.Clear()
			dirsByID[dir.ID] = dir
		}
	}

	for _, file := range fs {
		if floc, ok := loc.Files[file.Path]; ok {
			file.Size.Lines = int(floc.Code + floc.Comments + floc.Blanks)
			file.Size.Other["Code"] = int(floc.Code)
			file.Size.Other["Comments"] = int(floc.Comments)
			file.Size.Other["Blanks"] = int(floc.Blanks)
		}

		if file.ProjectDirectoryID != nil {
			if dir, ok := dirsByID[*file.ProjectDirectoryID]; ok {
				dir.Size.Files += 1
				dir.Size.Lines += file.Size.Lines
				dir.Size.Other["Code"] += file.Size.Other["Code"]
				dir.Size.Other["Comments"] += file.Size.Other["Comments"]
				dir.Size.Other["Blanks"] += file.Size.Other["Blanks"]
			}
		}
	}

	for _, proj := range ps {
		proj.Sizes = map[string]*model.Size{}

		for _, dir := range proj.Dirs {
			proj.AddSize(dir.Type.String(), dir.Size)
		}
	}

	fmt.Printf("Writing results...\n")

	err = storage.WriteProjects(projectsDB, archer.ChangedSize)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB, archer.ChangedSize)
	if err != nil {
		return err
	}

	return nil
}

func (l *locImporter) computeLOC(files []*model.File) (*gocloc.Result, error) {
	languages := gocloc.NewDefinedLanguages()
	options := gocloc.NewClocOptions()

	paths := lo.Map(files, func(f *model.File, _ int) string { return f.Path })

	processor := gocloc.NewProcessor(languages, options)
	loc, err := processor.Analyze(paths)
	if err != nil {
		return nil, errors.Wrapf(err, "error computing lines of code")
	}

	return loc, nil
}
