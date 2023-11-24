package filters

type UsageType int

const (
	DontCare UsageType = iota
	Include
	Exclude // Exclude has preference over Include
)

func (u UsageType) Merge(other UsageType) UsageType {
	switch {
	case u == other:
		return u
	case u == Exclude || other == Exclude:
		return Exclude
	default: // One of them is Include, because they have 2 different values
		return Include
	}
}
