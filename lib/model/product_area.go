package model

type ProductArea struct {
	Name string
	ID   UUID

	Size    *Size
	Changes *Changes
	Metrics *Metrics
	Data    map[string]string
}

func NewProductArea(name string, id *UUID) *ProductArea {
	var uuid UUID
	if id == nil {
		uuid = NewUUID("a")
	} else {
		uuid = *id
	}

	return &ProductArea{
		Name:    name,
		ID:      uuid,
		Size:    NewSize(),
		Changes: NewChanges(),
		Metrics: NewMetrics(),
		Data:    map[string]string{},
	}
}
