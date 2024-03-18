package server

import "github.com/pescuma/archer/lib/model"

type Filters struct {
	FilterFile     string   `form:"file"`
	FilterProject  string   `form:"proj"`
	FilterRepo     string   `form:"repo"`
	FilterPerson   string   `form:"person"`
	FilterPersonID model.ID `form:"person.id"`
}

type ListParams struct {
	GridParams
	Filters
}

type StatsParams struct {
	Filters
}
