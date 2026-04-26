package codeflow

import (
	"context"
	"math"
	"time"
)

// ToolAnalyzer performs dependency analysis and anomaly detection on tool executions.
type ToolAnalyzer struct {
	cf *CodeFlow
}

// NewToolAnalyzer creates a new ToolAnalyzer.
func NewToolAnalyzer(cf *CodeFlow) *ToolAnalyzer {
	return &ToolAnalyzer{cf: cf}
}

// AnalyzeDependencies analyzes the dependencies of a given tool based on its execution graph.
func (ta *ToolAnalyzer) AnalyzeDependencies(ctx context.Context, toolID string) ([]ToolDependency, error) {
	graph, err := ta.cf.GetToolGraph(ctx, toolID)
	if err != nil {
		return nil, err
	}

	if graph == nil {
		return []ToolDependency{}, nil
	}

	deps := make([]ToolDependency, 0)
	seen := make(map[string]bool)

	for _, edge := range graph.Edges {
		if edge.Source == toolID && !seen[edge.Target] {
			seen[edge.Target] = true
			deps = append(deps, ToolDependency{
				ToolID:         toolID,
				DependencyID:   edge.Target,
				DependencyType: edge.EdgeType,
				Required:       edge.EdgeType == "dependency",
			})
		}
	}

	return deps, nil
}

// DetectAnomalies detects anomalies in a set of execution metrics.
// Anomalies detected: duration > 2*average, error rate > 10%, memory spikes > 2*average.
func (ta *ToolAnalyzer) DetectAnomalies(metrics []ExecutionMetrics) ([]Anomaly, error) {
	if len(metrics) == 0 {
		return []Anomaly{}, nil
	}

	var anomalies []Anomaly
	now := time.Now()

	// Compute averages
	var avgDuration time.Duration
	var avgMemory int64
	var totalErrors int64
	var totalCalls int64

	for _, m := range metrics {
		avgDuration += m.Duration
		avgMemory += m.MemoryBytes
		totalErrors += m.ErrorCount
		totalCalls += m.TotalCalls
	}

	n := float64(len(metrics))
	meanDuration := time.Duration(int64(float64(avgDuration) / n))
	meanMemory := int64(float64(avgMemory) / n)

	// Compute standard deviation for duration
	var varianceDur float64
	var varianceMem float64
	for _, m := range metrics {
		durDiff := float64(m.Duration-meanDuration) / float64(time.Millisecond)
		varianceDur += durDiff * durDiff
		memDiff := float64(m.MemoryBytes - meanMemory)
		varianceMem += memDiff * memDiff
	}
	stdDevDur := time.Duration(int64(math.Sqrt(varianceDur/n))) * time.Millisecond
	stdDevMem := int64(math.Sqrt(varianceMem / n))

	// Check each metric entry for anomalies
	for _, m := range metrics {
		// Duration anomaly: > 2*average or > 3*stddev
		if meanDuration > 0 && m.Duration > 2*meanDuration {
			severity := "medium"
			if m.Duration > 3*meanDuration {
				severity = "high"
			}
			anomalies = append(anomalies, Anomaly{
				Type:        "duration",
				Severity:    severity,
				Description: "Execution duration exceeds 2x average",
				Value:       m.Duration.Seconds(),
				Threshold:   (2 * meanDuration).Seconds(),
				DetectedAt:  now,
			})
		}

		if stdDevDur > 0 && m.Duration > meanDuration+3*stdDevDur {
			anomalies = append(anomalies, Anomaly{
				Type:        "duration",
				Severity:    "high",
				Description: "Execution duration exceeds 3 standard deviations",
				Value:       m.Duration.Seconds(),
				Threshold:   (meanDuration + 3*stdDevDur).Seconds(),
				DetectedAt:  now,
			})
		}

		// Error rate anomaly: > 10%
		if totalCalls > 0 {
			errorRate := float64(totalErrors) / float64(totalCalls) * 100
			if errorRate > 10 {
				severity := "medium"
				if errorRate > 25 {
					severity = "high"
				}
				anomalies = append(anomalies, Anomaly{
					Type:        "error_rate",
					Severity:    severity,
					Description: "Error rate exceeds 10%",
					Value:       errorRate,
					Threshold:   10,
					DetectedAt:  now,
				})
			}
		}

		// Memory spike: > 2*average or > 3*stddev
		if meanMemory > 0 && m.MemoryBytes > 2*meanMemory {
			severity := "medium"
			if m.MemoryBytes > 3*meanMemory {
				severity = "high"
			}
			anomalies = append(anomalies, Anomaly{
				Type:        "memory_spike",
				Severity:    severity,
				Description: "Memory usage exceeds 2x average",
				Value:       float64(m.MemoryBytes),
				Threshold:   float64(2 * meanMemory),
				DetectedAt:  now,
			})
		}

		if stdDevMem > 0 && m.MemoryBytes > meanMemory+3*stdDevMem {
			anomalies = append(anomalies, Anomaly{
				Type:        "memory_spike",
				Severity:    "high",
				Description: "Memory usage exceeds 3 standard deviations",
				Value:       float64(m.MemoryBytes),
				Threshold:   float64(meanMemory + 3*stdDevMem),
				DetectedAt:  now,
			})
		}
	}

	return anomalies, nil
}
