package codeflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolAnalyzer(t *testing.T) {
	cf := NewCodeFlow()
	ta := NewToolAnalyzer(cf)
	assert.NotNil(t, ta)
	assert.Equal(t, cf, ta.cf)
}

func TestAnalyzer_AnalyzeDependencies_EmptyGraph(t *testing.T) {
	cf := NewCodeFlow()
	ta := NewToolAnalyzer(cf)
	ctx := context.Background()

	deps, err := ta.AnalyzeDependencies(ctx, "tool_a")
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestAnalyzer_AnalyzeDependencies_WithEdges(t *testing.T) {
	cf := NewCodeFlow()
	ta := NewToolAnalyzer(cf)
	ctx := context.Background()

	graph := &ToolExecutionGraph{
		ToolID: "tool_a",
		Edges: []GraphEdge{
			{Source: "tool_a", Target: "dep_1", EdgeType: "dependency"},
			{Source: "tool_a", Target: "dep_2", EdgeType: "dataflow"},
			{Source: "dep_1", Target: "dep_3", EdgeType: "dependency"},
		},
	}
	cf.SetToolGraph(ctx, "tool_a", graph)

	deps, err := ta.AnalyzeDependencies(ctx, "tool_a")
	require.NoError(t, err)
	assert.Len(t, deps, 2)

	assert.Equal(t, "dep_1", deps[0].DependencyID)
	assert.True(t, deps[0].Required)
	assert.Equal(t, "dependency", deps[0].DependencyType)

	assert.Equal(t, "dep_2", deps[1].DependencyID)
	assert.False(t, deps[1].Required)
	assert.Equal(t, "dataflow", deps[1].DependencyType)
}

func TestAnalyzer_DetectAnomalies_EmptyMetrics(t *testing.T) {
	ta := NewToolAnalyzer(NewCodeFlow())

	anomalies, err := ta.DetectAnomalies([]ExecutionMetrics{})
	require.NoError(t, err)
	assert.Empty(t, anomalies)
}

func TestAnalyzer_DetectAnomalies_NormalMetrics(t *testing.T) {
	ta := NewToolAnalyzer(NewCodeFlow())

	metrics := []ExecutionMetrics{
		{Duration: 100 * time.Millisecond, MemoryBytes: 1024},
		{Duration: 110 * time.Millisecond, MemoryBytes: 1024},
		{Duration: 90 * time.Millisecond, MemoryBytes: 1024},
	}

	anomalies, err := ta.DetectAnomalies(metrics)
	require.NoError(t, err)
	assert.Empty(t, anomalies) // all within normal range
}

func TestAnalyzer_DetectAnomalies_DurationSpike(t *testing.T) {
	ta := NewToolAnalyzer(NewCodeFlow())

	metrics := []ExecutionMetrics{
		{Duration: 100 * time.Millisecond, MemoryBytes: 1024},
		{Duration: 100 * time.Millisecond, MemoryBytes: 1024},
		{Duration: 500 * time.Millisecond, MemoryBytes: 1024}, // > 2x average
	}

	anomalies, err := ta.DetectAnomalies(metrics)
	require.NoError(t, err)
	assert.NotEmpty(t, anomalies)

	foundDuration := false
	for _, a := range anomalies {
		if a.Type == "duration" {
			foundDuration = true
			break
		}
	}
	assert.True(t, foundDuration, "should detect duration anomaly")
}

func TestAnalyzer_DetectAnomalies_MemorySpike(t *testing.T) {
	ta := NewToolAnalyzer(NewCodeFlow())

	metrics := []ExecutionMetrics{
		{Duration: 100 * time.Millisecond, MemoryBytes: 1000},
		{Duration: 100 * time.Millisecond, MemoryBytes: 1000},
		{Duration: 100 * time.Millisecond, MemoryBytes: 10000}, // > 2x average
	}

	anomalies, err := ta.DetectAnomalies(metrics)
	require.NoError(t, err)
	assert.NotEmpty(t, anomalies)

	foundMem := false
	for _, a := range anomalies {
		if a.Type == "memory_spike" {
			foundMem = true
			break
		}
	}
	assert.True(t, foundMem, "should detect memory spike")
}

func TestAnalyzer_DetectAnomalies_ErrorRate(t *testing.T) {
	ta := NewToolAnalyzer(NewCodeFlow())

	metrics := []ExecutionMetrics{
		{Duration: 100 * time.Millisecond, MemoryBytes: 1000, ErrorCount: 3, TotalCalls: 10}, // 30% error rate > 10%
	}

	anomalies, err := ta.DetectAnomalies(metrics)
	require.NoError(t, err)
	assert.NotEmpty(t, anomalies)

	foundError := false
	for _, a := range anomalies {
		if a.Type == "error_rate" {
			foundError = true
			assert.Equal(t, "high", a.Severity) // 30% > 25% threshold → high
			break
		}
	}
	assert.True(t, foundError, "should detect error rate anomaly")
}

func TestAnomalyTypes(t *testing.T) {
	now := time.Now()
	a := Anomaly{
		ToolID:      "tool_a",
		Type:        "duration",
		Severity:    "high",
		Description: "Test anomaly",
		Value:       5.0,
		Threshold:   2.0,
		DetectedAt:  now,
	}

	assert.Equal(t, "duration", a.Type)
	assert.Equal(t, "high", a.Severity)
	assert.Equal(t, 5.0, a.Value)
}

func TestToolDependency(t *testing.T) {
	d := ToolDependency{
		ToolID:         "tool_a",
		DependencyID:   "dep_1",
		DependencyType: "data",
		Required:       true,
	}

	assert.Equal(t, "dep_1", d.DependencyID)
	assert.True(t, d.Required)
	assert.Equal(t, "data", d.DependencyType)
}

func TestNodeTypes(t *testing.T) {
	assert.Equal(t, NodeType("tool"), NodeTypeTool)
	assert.Equal(t, NodeType("data"), NodeTypeData)
	assert.Equal(t, NodeType("dependency"), NodeTypeDependency)
}
