package model

type Metrics struct {
	GuiceDependencies    int
	CyclomaticComplexity int
	CognitiveComplexity  int
	ChangesIn6Months     int
	ChangesTotal         int
}

func NewMetrics() *Metrics {
	result := &Metrics{}
	result.Clear()
	return result
}

func (m *Metrics) Clear() {
	m.GuiceDependencies = -1
	m.CyclomaticComplexity = -1
	m.CognitiveComplexity = -1
	m.ChangesIn6Months = -1
	m.ChangesTotal = -1
}

func (m *Metrics) Add(other *Metrics) {
	m.GuiceDependencies = add(m.GuiceDependencies, other.GuiceDependencies)
	m.CyclomaticComplexity = add(m.CyclomaticComplexity, other.CyclomaticComplexity)
	m.CognitiveComplexity = add(m.CognitiveComplexity, other.CognitiveComplexity)
	m.ChangesIn6Months = add(m.ChangesIn6Months, other.ChangesIn6Months)
	m.ChangesTotal = add(m.ChangesTotal, other.ChangesTotal)
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
