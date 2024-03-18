package orm

import (
	"math"

	"github.com/pescuma/archer/lib/model"
)

type sqlMetricsAggregate struct {
	DependenciesGuiceTotal    *int
	DependenciesGuiceAvg      *float32
	ComplexityCyclomaticTotal *int
	ComplexityCyclomaticAvg   *float32
	ComplexityCognitiveTotal  *int
	ComplexityCognitiveAvg    *float32
	ComplexityFocusTotal      *int
	ComplexityFocusAvg        *float32
}

func newSqlMetricsAggregate(m *model.Metrics, s *model.Size) *sqlMetricsAggregate {
	return &sqlMetricsAggregate{
		DependenciesGuiceTotal:    encodeMetric(m.GuiceDependencies),
		DependenciesGuiceAvg:      encodeMetricAggregate(m.GuiceDependencies, s.Files),
		ComplexityCyclomaticTotal: encodeMetric(m.CyclomaticComplexity),
		ComplexityCyclomaticAvg:   encodeMetricAggregate(m.CyclomaticComplexity, s.Files),
		ComplexityCognitiveTotal:  encodeMetric(m.CognitiveComplexity),
		ComplexityCognitiveAvg:    encodeMetricAggregate(m.CognitiveComplexity, s.Files),
		ComplexityFocusTotal:      encodeMetric(m.FocusedComplexity),
		ComplexityFocusAvg:        encodeMetricAggregate(m.FocusedComplexity, s.Files),
	}
}

func (s *sqlMetricsAggregate) ToModel() *model.Metrics {
	return &model.Metrics{
		GuiceDependencies:    decodeMetric(s.DependenciesGuiceTotal),
		CyclomaticComplexity: decodeMetric(s.ComplexityCyclomaticTotal),
		CognitiveComplexity:  decodeMetric(s.ComplexityCognitiveTotal),
		FocusedComplexity:    decodeMetric(s.ComplexityFocusTotal),
	}
}

func encodeMetricAggregate(v int, t int) *float32 {
	if v == -1 {
		return nil
	}
	if t == 0 {
		return nil
	}
	a := float32(math.Round(float64(v)*10/float64(t)) / 10)
	return &a
}
