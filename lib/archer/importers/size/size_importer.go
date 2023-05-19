package size

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

type sizeImporter struct {
	filters []string
}

func NewImporter(filters []string) archer.Importer {
	return &sizeImporter{
		filters: filters,
	}
}

func (s *sizeImporter) Import(projs *model.Projects, files *model.Files, storage archer.Storage) error {
	projects, err := projs.FilterProjects(s.filters, model.FilterExcludeExternal)
	if err != nil {
		return err
	}

	for i, proj := range projects {
		changed, err := s.importSize(files, proj)
		if err != nil {
			return err
		}

		fmt.Printf("[%v / %v] %v lines of code from '%v'\n", i, len(projects),
			utils.IIf(changed, "Imported", "Skipped"),
			proj)
	}

	err = storage.WriteProjects(projs, archer.ChangedSize)
	if err != nil {
		return err
	}

	err = storage.WriteFiles(files, archer.ChangedSize)
	if err != nil {
		return err
	}

	return nil
}

func (s *sizeImporter) importSize(files *model.Files, proj *model.Project) (bool, error) {
	if len(proj.Dirs) == 0 {
		return false, nil
	}

	proj.Sizes = map[string]*model.Size{}

	for _, dir := range proj.Dirs {
		err := s.computeCLOC(files, dir)
		if err != nil {
			return false, err
		}

		proj.AddSize(dir.Type.String(), dir.Size)
	}

	return true, nil
}

func (s *sizeImporter) computeCLOC(files *model.Files, dir *model.ProjectDirectory) error {
	languages := gocloc.NewDefinedLanguages()
	options := gocloc.NewClocOptions()

	paths := map[string]*model.File{}
	for _, f := range files.ListByProjectDirectory(dir) {
		paths[f.Path] = f
	}

	processor := gocloc.NewProcessor(languages, options)
	loc, err := processor.Analyze(lo.Keys(paths))
	if err != nil {
		return errors.Wrapf(err, "error computing lones of code")
	}

	dir.Size.Clear()

	for path, file := range paths {
		file.Size.Clear()

		file.Size.Files = 1
		dir.Size.Files += 1

		info, err := os.Stat(path)
		if err == nil {
			size := int(info.Size())
			file.Size.Bytes = size
			dir.Size.Bytes += size
		}

		floc, ok := loc.Files[path]
		if ok {
			file.Size.Lines = int(floc.Code + floc.Comments + floc.Blanks)
			file.Size.Other["Code"] = int(floc.Code)
			file.Size.Other["Comments"] = int(floc.Comments)
			file.Size.Other["Blanks"] = int(floc.Blanks)

			dir.Size.Lines += file.Size.Lines
			dir.Size.Other["Code"] += file.Size.Other["Code"]
			dir.Size.Other["Comments"] += file.Size.Other["Comments"]
			dir.Size.Other["Blanks"] += file.Size.Other["Blanks"]
		}
	}

	return nil
}
