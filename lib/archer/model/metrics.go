package model

type Metrics struct {
	GuiceDependencies int
}

func NewMetrics() *Metrics {
	return &Metrics{
		GuiceDependencies: -1,
	}
}

func (m *Metrics) Add(other *Metrics) {
	m.GuiceDependencies = add(m.GuiceDependencies, other.GuiceDependencies)
}

func (m *Metrics) Clear() {
	m.GuiceDependencies = -1
}

func add(a, b int) int {
	if b == -1 {
		return a
	}
	if a == -1 {
		return b
	}
	return a + b
}
