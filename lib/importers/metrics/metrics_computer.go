package metrics

import (
	"math"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
	"github.com/pescuma/archer/lib/utils"
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

	c.console.Printf("Computing metrics for projects, dirs and areas ...\n")

	for _, p := range projectsDB.ListProjects(model.FilterExcludeExternal) {
		p.Metrics.Clear()
		for _, d := range p.Dirs {
			d.Metrics.Clear()
		}
	}

	for _, a := range peopleDB.ListProductAreas() {
		a.Metrics.Clear()
	}

	filesByDir := filesDB.GroupByDirectory()

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

	return nil
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
