package manifest

import (
	"fmt"
	"strings"
)

type metricSuggester struct {
	cfg DomainConfig
}

func NewMetricSuggester(cfg DomainConfig) MetricSuggester {
	return &metricSuggester{cfg: cfg}
}

func (m *metricSuggester) Suggest(entities []Entity, _ []TableSchema) ([]MetricSuggestion, error) {
	if m.cfg.MeasureKeywords == nil {
		return nil, fmt.Errorf("metricSuggester: MeasureKeywords is nil")
	}

	var suggestions []MetricSuggestion

	for _, entity := range entities {
		var measures, categories, temporals []ColumnSchema
		for _, prop := range entity.Properties {
			switch prop.Class {
			case Measure:
				measures = append(measures, prop)
			case Category:
				categories = append(categories, prop)
			case Temporal:
				temporals = append(temporals, prop)
			}
		}

		dimNames := make([]string, len(categories))
		for i, cat := range categories {
			dimNames[i] = cat.Name
		}

		temporalKey := ""
		if len(temporals) > 0 {
			temporalKey = temporals[0].Name
		}

		for _, measure := range measures {
			suggestions = append(suggestions, MetricSuggestion{
				Name:        entity.Name + "_" + measure.Name,
				SourceTable: entity.Table,
				Dimensions:  dimNames,
				Measure:     measure.Name,
				TemporalKey: temporalKey,
				Aggregation: m.resolveAggregation(measure.Name),
			})
		}
	}

	return suggestions, nil
}

func (m *metricSuggester) resolveAggregation(columnName string) AggType {
	lower := strings.ToLower(columnName)
	for _, kw := range m.cfg.MeasureKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			if agg, ok := m.cfg.MetricMappings[kw]; ok {
				return agg
			}
		}
	}
	return Avg
}
