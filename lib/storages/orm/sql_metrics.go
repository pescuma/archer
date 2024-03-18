package orm

import "github.com/pescuma/archer/lib/model"

type sqlMetrics struct {
	DependenciesGuice    *int
	Abstracts            *int
	ComplexityCyclomatic *int
	ComplexityCognitive  *int
	ComplexityFocus      *int
}

func newSqlMetrics(m *model.Metrics) *sqlMetrics {
	return &sqlMetrics{
		DependenciesGuice:    encodeMetric(m.GuiceDependencies),
		Abstracts:            encodeMetric(m.Abstracts),
		ComplexityCyclomatic: encodeMetric(m.CyclomaticComplexity),
		ComplexityCognitive:  encodeMetric(m.CognitiveComplexity),
		ComplexityFocus:      encodeMetric(m.FocusedComplexity),
	}
}

func (s *sqlMetrics) toModel() *model.Metrics {
	return &model.Metrics{
		GuiceDependencies:    decodeMetric(s.DependenciesGuice),
		Abstracts:            decodeMetric(s.Abstracts),
		CyclomaticComplexity: decodeMetric(s.ComplexityCyclomatic),
		CognitiveComplexity:  decodeMetric(s.ComplexityCognitive),
		FocusedComplexity:    decodeMetric(s.ComplexityFocus),
	}
}
