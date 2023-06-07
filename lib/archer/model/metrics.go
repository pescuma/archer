package model

type Metrics struct {
	GuiceDependencies    int
	Abstracts            int
	CyclomaticComplexity int
	CognitiveComplexity  int
	FocusedComplexity    int
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
	m.FocusedComplexity = -1
}

func (m *Metrics) Add(other *Metrics) {
	m.GuiceDependencies = add(m.GuiceDependencies, other.GuiceDependencies)
	m.CyclomaticComplexity = add(m.CyclomaticComplexity, other.CyclomaticComplexity)
	m.CognitiveComplexity = add(m.CognitiveComplexity, other.CognitiveComplexity)
	m.FocusedComplexity = add(m.FocusedComplexity, other.FocusedComplexity)
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
