package loc

import (
	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
)

type Computer struct {
	console consoles.Console
	storage storages.Storage
}

func NewComputer(console consoles.Console, storage storages.Storage) *Computer {
	return &Computer{
		console: console,
		storage: storage,
	}
}

func (c *Computer) Compute() error {
	projectsDB, err := c.storage.LoadProjects()
	if err != nil {
		return err
	}

	filesDB, err := c.storage.LoadFiles()
	if err != nil {
		return err
	}

	peopleDB, err := c.storage.LoadPeople()
	if err != nil {
		return err
	}

	c.console.Printf("Computing LOC for projects, dirs and areas...\n")

	for _, a := range peopleDB.ListProductAreas() {
		a.Size.Clear()
	}

	filesByDir := filesDB.GroupFilesByDirectory()

	for _, proj := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		proj.ClearSizes()

		for _, dir := range proj.Dirs {
			dir.Size.Clear()

			for _, file := range filesByDir[dir.ID] {
				if !file.Exists {
					continue
				}

				proj.SeenAt(file.FirstSeen, file.LastSeen)
				dir.SeenAt(file.FirstSeen, file.LastSeen)

				dir.Size.Add(file.Size)

				if file.ProductAreaID != nil {
					peopleDB.GetProductAreaByID(*file.ProductAreaID).Size.Add(file.Size)
				}
			}

			proj.AddSize(dir.Type.String(), dir.Size)
		}
	}

	c.console.Printf("Writing results...\n")

	err = c.storage.WriteProjects()
	if err != nil {
		return err
	}

	err = c.storage.WriteFiles()
	if err != nil {
		return err
	}

	err = c.storage.WritePeople()
	if err != nil {
		return err
	}

	return nil
}
