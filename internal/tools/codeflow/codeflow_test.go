package codeflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCodeFlow(t *testing.T) {
	cf := NewCodeFlow()
	assert.NotNil(t, cf)
	assert.NotNil(t, cf.records)
	assert.NotNil(t, cf.graphs)
	assert.NotNil(t, cf.metrics)
}

func TestCodeFlow_ListEngines(t *testing.T) {
	cf := NewCodeFlow()
	engines := cf.ListEngines()
	assert.ElementsMatch(t, []string{"execution_tracker", "graph_analyzer", "metrics_aggregator"}, engines)
}

func TestCodeFlow_RecordExecution(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	metrics := ExecutionMetrics{
		Duration:      100 * time.Millisecond,
		MemoryBytes:   1024,
		CPUMillicores: 500,
		CallCount:     0,
		ErrorCount:    0,
	}

	err := cf.RecordExecution(ctx, "tool_a", metrics)
	require.NoError(t, err)

	// Check metrics aggregation
	m, err := cf.GetMetrics(ctx, "tool_a")
	require.NoError(t, err)
	assert.Equal(t, int64(1), m.CallCount)
	assert.Equal(t, int64(0), m.ErrorCount)
	assert.Equal(t, 100*time.Millisecond, m.Duration)
}

func TestCodeFlow_RecordExecutionWithError(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	metrics := ExecutionMetrics{
		Duration:      50 * time.Millisecond,
		MemoryBytes:   512,
		CPUMillicores: 200,
		ErrorCount:    3,
	}

	err := cf.RecordExecution(ctx, "tool_b", metrics)
	require.NoError(t, err)

	// ErrorCount > 0 should set status to "error"
	r, err := cf.ListRecentExecutions(ctx, 10)
	require.NoError(t, err)
	require.Len(t, r, 1)
	assert.Equal(t, "error", r[0].Status)
}

func TestCodeFlow_RecordExecutionRollingAverage(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	// First record: 100ms
	err := cf.RecordExecution(ctx, "tool_a", ExecutionMetrics{Duration: 100 * time.Millisecond, MemoryBytes: 1000, CPUMillicores: 100})
	require.NoError(t, err)

	// Second record: 200ms → rolling avg: (100*1 + 200)/2 = 150ms
	err = cf.RecordExecution(ctx, "tool_a", ExecutionMetrics{Duration: 200 * time.Millisecond, MemoryBytes: 2000, CPUMillicores: 200})
	require.NoError(t, err)

	m, err := cf.GetMetrics(ctx, "tool_a")
	require.NoError(t, err)
	assert.Equal(t, int64(2), m.CallCount)
	assert.InDelta(t, float64(150*time.Millisecond), float64(m.Duration), float64(time.Millisecond))
	assert.Equal(t, int64(1500), m.MemoryBytes) // (1000+2000)/2
	assert.Equal(t, int64(150), m.CPUMillicores) // (100+200)/2
}

func TestCodeFlow_GetToolGraph_New(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	// Tool with no graph gets empty graph (not nil)
	graph, err := cf.GetToolGraph(ctx, "new_tool")
	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, "new_tool", graph.ToolID)
	assert.Empty(t, graph.Nodes)
	assert.Empty(t, graph.Edges)
}

func TestCodeFlow_SetToolGraph(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	graph := &ToolExecutionGraph{
		ToolID: "tool_a",
		Nodes: []GraphNode{
			{ID: "tool_a", Label: "Tool A", NodeType: NodeTypeTool},
			{ID: "dep_1", Label: "Dep 1", NodeType: NodeTypeDependency},
		},
		Edges: []GraphEdge{
			{Source: "tool_a", Target: "dep_1", EdgeType: "dependency", Weight: 1},
		},
	}

	err := cf.SetToolGraph(ctx, "tool_a", graph)
	require.NoError(t, err)

	loaded, err := cf.GetToolGraph(ctx, "tool_a")
	require.NoError(t, err)
	assert.Equal(t, "tool_a", loaded.ToolID)
	assert.Len(t, loaded.Nodes, 2)
	assert.Len(t, loaded.Edges, 1)
}

func TestCodeFlow_GetMetrics_NewTool(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	metrics, err := cf.GetMetrics(ctx, "nonexistent")
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.CallCount)
}

func TestCodeFlow_ListRecentExecutions(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		cf.RecordExecution(ctx, "tool_a", ExecutionMetrics{Duration: time.Duration(i) * time.Millisecond})
	}

	records, err := cf.ListRecentExecutions(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

func TestCodeFlow_ListRecentExecutions_LimitExceeds(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	cf.RecordExecution(ctx, "tool_a", ExecutionMetrics{})

	records, err := cf.ListRecentExecutions(ctx, 100) // exceeds available
	require.NoError(t, err)
	assert.Len(t, records, 1)
}

func TestCodeFlow_ListRecentExecutions_ZeroLimit(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	cf.RecordExecution(ctx, "tool_a", ExecutionMetrics{})

	records, err := cf.ListRecentExecutions(ctx, 0)
	require.NoError(t, err)
	assert.Len(t, records, 1) // returns all
}

func TestCodeFlow_GetRecords(t *testing.T) {
	cf := NewCodeFlow()
	ctx := context.Background()

	cf.RecordExecution(ctx, "tool_a", ExecutionMetrics{})
	cf.RecordExecution(ctx, "tool_b", ExecutionMetrics{})
	cf.RecordExecution(ctx, "tool_a", ExecutionMetrics{})

	records, err := cf.GetRecords(ctx, "tool_a")
	require.NoError(t, err)
	assert.Len(t, records, 2)

	records, err = cf.GetRecords(ctx, "tool_c")
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestCodeFlow_MaxRecordsLimit(t *testing.T) {
	cf := NewCodeFlow()
	cf.maxRecords = 3
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		cf.RecordExecution(ctx, "tool_a", ExecutionMetrics{MemoryBytes: int64(i)})
	}

	records, err := cf.ListRecentExecutions(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, records, 3) // capped at maxRecords
}

func TestExecutionMetrics(t *testing.T) {
	now := time.Now()
	m := ExecutionMetrics{
		Duration:       5 * time.Second,
		MemoryBytes:    65536,
		CPUMillicores:  1000,
		CallCount:      42,
		ErrorCount:     3,
		TotalCalls:     100,
		LastExecutedAt: now,
	}

	assert.Equal(t, 5*time.Second, m.Duration)
	assert.Equal(t, int64(65536), m.MemoryBytes)
	assert.Equal(t, int64(42), m.CallCount)
	assert.Equal(t, now, m.LastExecutedAt)
}

func TestGraphDataTypes(t *testing.T) {
	node := GraphNode{
		ID:       "node_1",
		Label:    "Node 1",
		NodeType: NodeTypeTool,
	}
	assert.Equal(t, "node_1", node.ID)
	assert.Equal(t, NodeTypeTool, node.NodeType)

	edge := GraphEdge{
		Source:   "node_1",
		Target:   "node_2",
		EdgeType: "calls",
		Weight:   5,
	}
	assert.Equal(t, "calls", edge.EdgeType)
	assert.Equal(t, 5, edge.Weight)
}

func TestExecutionRecord(t *testing.T) {
	now := time.Now()
	r := ExecutionRecord{
		ToolID:    "tool_a",
		UserID:    "user_1",
		Status:    "success",
		Timestamp: now,
	}
	assert.Equal(t, "success", r.Status)
	assert.Equal(t, "user_1", r.UserID)
}
