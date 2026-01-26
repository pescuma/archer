package model

type ProductArea struct {
	Name string
	ID   ID

	Size    *Size
	Changes *Changes
	Metrics *Metrics
	SeenAt  *SeenAt
	Data    map[string]string
}

func NewProductArea(name string, id ID) *ProductArea {
	return &ProductArea{
		Name:    name,
		ID:      id,
		Size:    NewSize(),
		Changes: NewChanges(),
		Metrics: NewMetrics(),
		SeenAt:  NewSeenAt(),
		Data:    map[string]string{},
	}
}
