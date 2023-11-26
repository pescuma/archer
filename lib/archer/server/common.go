package server

type Filters struct {
	FilterFile     string `form:"file"`
	FilterProject  string `form:"proj"`
	FilterRepo     string `form:"repo"`
	FilterPerson   string `form:"person"`
	FilterPersonID string `form:"person.id"`
}

type ListParams struct {
	GridParams
	Filters
}

type StatsParams struct {
	Filters
}
