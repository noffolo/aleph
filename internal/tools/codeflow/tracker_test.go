package codeflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionTracker(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	assert.NotNil(t, et)
	assert.NotNil(t, et.errors)
}

func TestTracker_TrackExecution(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	metrics := ExecutionMetrics{
		Duration:      100 * time.Millisecond,
		MemoryBytes:   1024,
		CPUMillicores: 500,
	}

	err := et.TrackExecution(ctx, "tool_a", metrics)
	require.NoError(t, err)

	// Should have 0 consecutive errors
	assert.Equal(t, 0, et.GetConsecutiveErrors("tool_a"))
}

func TestTracker_TrackExecution_Error(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	metrics := ExecutionMetrics{Duration: time.Second, ErrorCount: 2}

	err := et.TrackExecution(ctx, "tool_a", metrics)
	require.NoError(t, err)

	assert.Equal(t, 1, et.GetConsecutiveErrors("tool_a"))

	// Second consecutive error
	err = et.TrackExecution(ctx, "tool_a", metrics)
	require.NoError(t, err)
	assert.Equal(t, 2, et.GetConsecutiveErrors("tool_a"))

	// Success resets counter
	err = et.TrackExecution(ctx, "tool_a", ExecutionMetrics{Duration: time.Second, ErrorCount: 0})
	require.NoError(t, err)
	assert.Equal(t, 0, et.GetConsecutiveErrors("tool_a"))
}

func TestTracker_TrackExecution_EmptyToolID(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	err := et.TrackExecution(ctx, "", ExecutionMetrics{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestTracker_GetConsecutiveErrors_NoTool(t *testing.T) {
	et := NewExecutionTracker(NewCodeFlow())
	assert.Equal(t, 0, et.GetConsecutiveErrors("nonexistent"))
}

func TestTracker_RecordDependency(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	err := et.RecordDependency(ctx, "tool_a", "dep_1", "dependency")
	require.NoError(t, err)

	graph, err := et.GetToolGraph(ctx, "tool_a")
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Len(t, graph.Nodes, 2) // tool_a + dep_1
	assert.Len(t, graph.Edges, 1)
	assert.Equal(t, "tool_a", graph.Edges[0].Source)
	assert.Equal(t, "dep_1", graph.Edges[0].Target)
}

func TestTracker_RecordDependency_DedupNodes(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	// Same dependency recorded twice
	err := et.RecordDependency(ctx, "tool_a", "dep_1", "dependency")
	require.NoError(t, err)

	err = et.RecordDependency(ctx, "tool_a", "dep_1", "dependency")
	require.NoError(t, err)

	graph, err := et.GetToolGraph(ctx, "tool_a")
	require.NoError(t, err)

	// Nodes should have 2 entries (tool_a + dep_1, not 3)
	assert.Len(t, graph.Nodes, 2)
	// Edges should have 2 (we don't dedup edges)
	assert.Len(t, graph.Edges, 2)
}

func TestTracker_GetToolGraph(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	graph, err := et.GetToolGraph(ctx, "new_tool")
	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, "new_tool", graph.ToolID)
}

func TestTracker_GetMetrics(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	metrics, err := et.GetMetrics(ctx, "no_such_tool")
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.CallCount)
}

func TestTracker_ListRecentExecutions(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	et.TrackExecution(ctx, "tool_a", ExecutionMetrics{Duration: time.Second})

	records, err := et.ListRecentExecutions(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "tool_a", records[0].ToolID)
}

func TestTracker_AggregateMetrics(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	et.TrackExecution(ctx, "tool_a", ExecutionMetrics{
		Duration: 100 * time.Millisecond, MemoryBytes: 1000, CPUMillicores: 100,
	})
	et.TrackExecution(ctx, "tool_a", ExecutionMetrics{
		Duration: 200 * time.Millisecond, MemoryBytes: 2000, CPUMillicores: 200,
	})
	et.TrackExecution(ctx, "tool_b", ExecutionMetrics{
		Duration: 50 * time.Millisecond, MemoryBytes: 500, CPUMillicores: 50,
	})

	agg, err := et.AggregateMetrics(ctx)
	require.NoError(t, err)
	assert.Len(t, agg, 2)

	// tool_a: 2 records, rolling avg
	ma, ok := agg["tool_a"]
	require.True(t, ok)
	assert.Equal(t, int64(2), ma.CallCount)
	assert.InDelta(t, float64(150*time.Millisecond), float64(ma.Duration), float64(time.Millisecond))
	assert.Equal(t, int64(1500), ma.MemoryBytes) // (1000+2000)/2
	assert.Equal(t, int64(150), ma.CPUMillicores) // (100+200)/2
}

func TestTracker_AggregateMetrics_Empty(t *testing.T) {
	cf := NewCodeFlow()
	et := NewExecutionTracker(cf)
	ctx := context.Background()

	agg, err := et.AggregateMetrics(ctx)
	require.NoError(t, err)
	assert.Empty(t, agg)
}
