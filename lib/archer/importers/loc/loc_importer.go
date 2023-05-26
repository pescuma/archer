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
	options Options
}

type Options struct {
	Incremental bool
}

func NewImporter(filters []string, options Options) archer.Importer {
	return &locImporter{
		filters: filters,
		options: options,
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

	files := make([]*model.File, 0, len(candidates))

	bar := utils.NewProgressBar(len(candidates))
	for _, file := range candidates {
		stat, err := os.Stat(file.Path)
		if err == nil {
			file.Exists = true

			modTime := stat.ModTime().String()

			if !l.options.Incremental || modTime != file.Data["loc:last_modified"] {
				files = append(files, file)
				file.Data["loc:last_modified"] = modTime

				file.Size.Clear()
				file.Size.Files = 1
				file.Size.Bytes = int(stat.Size())
			}

		} else {
			file.Exists = false

			file.Size.Clear()
		}

		_ = bar.Add(1)
	}

	fmt.Printf("Importing lines of code from %v files...\n", len(files))

	loc, err := l.computeLOC(files)
	if err != nil {
		return err
	}

	for _, file := range files {
		if floc, ok := loc.Files[file.Path]; ok {
			file.Size.Lines = int(floc.Code + floc.Comments + floc.Blanks)
			file.Size.Other["Code"] = int(floc.Code)
			file.Size.Other["Comments"] = int(floc.Comments)
			file.Size.Other["Blanks"] = int(floc.Blanks)
		}
	}

	l.updateParents(ps, candidates)

	fmt.Printf("Writing results...\n")

	err = storage.WriteProjects(projectsDB, archer.ChangedSize)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(filesDB, archer.ChangedData|archer.ChangedSize)
	if err != nil {
		return err
	}

	return nil
}

func (l *locImporter) updateParents(ps []*model.Project, files []*model.File) {
	filesByDir := lo.GroupBy(
		lo.Filter(files, func(f *model.File, _ int) bool { return f.ProjectDirectoryID != nil }),
		func(f *model.File) model.UUID { return *f.ProjectDirectoryID },
	)

	for _, proj := range ps {
		proj.Sizes = map[string]*model.Size{}

		for _, dir := range proj.Dirs {
			dir.Size.Clear()

			for _, file := range filesByDir[dir.ID] {
				dir.Size.Add(file.Size)
			}

			proj.AddSize(dir.Type.String(), dir.Size)
		}
	}
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
