package loc

import (
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/hhatto/gocloc"
	"github.com/pkg/errors"
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
	Incremental bool
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
		candidates = filesDB.ListFiles()
	} else {
		candidates = filesDB.ListFilesByProjects(ps)
	}

	i.console.Printf("Importing size from %v files...\n", len(candidates))

	files := make([]*model.File, 0, len(candidates))

	bar := utils.NewProgressBar(len(candidates))
	for _, file := range candidates {
		stat, err := os.Stat(file.Path)
		if err == nil && !stat.IsDir() {
			file.Exists = true
			file.SeenAt(time.Now(), stat.ModTime())

			modTime := stat.ModTime().String()

			if !opts.Incremental || modTime != file.Data["loc:last_modified"] {
				file.Data["loc:last_modified"] = modTime
				files = append(files, file)

				file.Size.Clear()
				file.Size.Files = 1
				file.Size.Bytes = int(stat.Size())
			}

		} else {
			file.Exists = false
		}

		_ = bar.Add(1)
	}

	i.console.Printf("Importing lines of code from %v files...\n", len(files))

	loc, err := i.computeLOC(files)
	if err != nil {
		return err
	}

	for _, file := range files {
		if floc, ok := loc.Files[file.Path]; ok {
			file.Size.Lines = int(floc.Code + floc.Comments + floc.Blanks)
			file.Size.Other["Code"] = int(floc.Code)
			file.Size.Other["Comments"] = int(floc.Comments)
			file.Size.Other["Blanks"] = int(floc.Blanks)

		} else if isText, err := utils.IsTextFile(file.Path); err == nil && isText {
			text, blanks, err := countLines(file.Path)
			if err == nil {
				file.Size.Lines = text + blanks
				file.Size.Other["Code"] = text
				file.Size.Other["Blanks"] = blanks
			}
		}
	}

	return nil
}

func countLines(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}

	defer file.Close()

	text := 0
	blank := 0

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			blank++
		} else {
			text++
		}
	}

	return text, blank, scanner.Err()
}

func (i *Importer) computeLOC(files []*model.File) (*gocloc.Result, error) {
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
