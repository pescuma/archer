package size

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hhatto/gocloc"
	"github.com/pkg/errors"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/utils"
)

type sizeImporter struct {
	storage archer.Storage
	filters []string
}

func NewSizeImporter(filters []string) archer.Importer {
	return &sizeImporter{
		filters: filters,
	}
}

func (s *sizeImporter) Import(projs *archer.Projects, storage archer.Storage) error {
	s.storage = storage

	projects, err := projs.FilterProjects(s.filters, archer.FilterExcludeExternal)
	if err != nil {
		return err
	}

	for i, proj := range projects {
		changed, err := s.importSize(proj)
		if err != nil {
			return err
		}

		fmt.Printf("[%v / %v] %v lines of code from '%v'\n", i, len(projects),
			utils.IIf(changed, "Imported", "Skipped"),
			proj)
	}

	return nil
}

func (s *sizeImporter) importSize(proj *archer.Project) (bool, error) {
	if len(proj.Dirs) == 0 {
		return false, nil
	}

	for t, dir := range proj.Dirs {
		size, err := s.computeCLOC(dir)
		if err != nil {
			return false, err
		}

		proj.AddSize(t, *size)
	}

	err := s.storage.WriteSize(proj)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *sizeImporter) computeCLOC(path string) (*archer.Size, error) {
	result := archer.Size{
		Other: map[string]int{},
	}

	_, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	}

	languages := gocloc.NewDefinedLanguages()
	options := gocloc.NewClocOptions()
	paths := []string{
		path,
	}

	processor := gocloc.NewProcessor(languages, options)
	loc, err := processor.Analyze(paths)
	if err != nil {
		return nil, errors.Wrapf(err, "error computing lones of code")
	}

	files := 0
	bytes := 0
	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}

			files += 1
			bytes += int(info.Size())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	result.Bytes = bytes
	result.Files = files
	result.Other["Code"] = int(loc.Total.Code)
	result.Other["Comments"] = int(loc.Total.Comments)
	result.Other["Blanks"] = int(loc.Total.Blanks)
	result.Lines = int(loc.Total.Code + loc.Total.Comments + loc.Total.Blanks)

	return &result, nil
}
