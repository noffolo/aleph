package codeflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"log/slog"
)

// ExecutionTracker provides high-level tracking operations on top of CodeFlow.
type ExecutionTracker struct {
	cf     *CodeFlow
	mu     sync.RWMutex
	errors map[string]int // toolID -> consecutive error count
}

// NewExecutionTracker creates a new ExecutionTracker backed by the given CodeFlow.
func NewExecutionTracker(cf *CodeFlow) *ExecutionTracker {
	return &ExecutionTracker{
		cf:     cf,
		errors: make(map[string]int),
	}
}

// TrackExecution records a single execution with the given parameters.
func (et *ExecutionTracker) TrackExecution(ctx context.Context, toolID string, metrics ExecutionMetrics) error {
	if toolID == "" {
		return fmt.Errorf("toolID cannot be empty")
	}

	if err := et.cf.RecordExecution(ctx, toolID, metrics); err != nil {
		return fmt.Errorf("record execution: %w", err)
	}

	et.mu.Lock()
	if metrics.ErrorCount > 0 {
		et.errors[toolID]++
	} else {
		et.errors[toolID] = 0
	}
	et.mu.Unlock()

	slog.Debug("tracked execution",
		"tool_id", toolID,
		"duration", metrics.Duration,
		"errors", metrics.ErrorCount,
	)
	return nil
}

// GetConsecutiveErrors returns the consecutive error count for a tool.
func (et *ExecutionTracker) GetConsecutiveErrors(toolID string) int {
	et.mu.RLock()
	defer et.mu.RUnlock()
	return et.errors[toolID]
}

// RecordDependency records a dependency execution in the tool's graph.
func (et *ExecutionTracker) RecordDependency(ctx context.Context, toolID, depID string, depType string) error {
	graph, err := et.cf.GetToolGraph(ctx, toolID)
	if err != nil {
		return fmt.Errorf("get tool graph: %w", err)
	}

	// Add dependency node if not present
	found := false
	for _, n := range graph.Nodes {
		if n.ID == depID {
			found = true
			break
		}
	}
	if !found {
		graph.Nodes = append(graph.Nodes, GraphNode{
			ID:       depID,
			Label:    depID,
			NodeType: NodeTypeDependency,
		})
	}

	// Add edge
	graph.Edges = append(graph.Edges, GraphEdge{
		Source:   toolID,
		Target:   depID,
		EdgeType: depType,
		Weight:   1,
	})

	// Ensure tool node exists
	foundTool := false
	for _, n := range graph.Nodes {
		if n.ID == toolID {
			foundTool = true
			break
		}
	}
	if !foundTool {
		graph.Nodes = append(graph.Nodes, GraphNode{
			ID:       toolID,
			Label:    toolID,
			NodeType: NodeTypeTool,
		})
	}

	return et.cf.SetToolGraph(ctx, toolID, graph)
}

// GetToolGraph delegates to CodeFlow.GetToolGraph.
func (et *ExecutionTracker) GetToolGraph(ctx context.Context, toolID string) (*ToolExecutionGraph, error) {
	return et.cf.GetToolGraph(ctx, toolID)
}

// GetMetrics delegates to CodeFlow.GetMetrics.
func (et *ExecutionTracker) GetMetrics(ctx context.Context, toolID string) (*ExecutionMetrics, error) {
	return et.cf.GetMetrics(ctx, toolID)
}

// ListRecentExecutions delegates to CodeFlow.ListRecentExecutions.
func (et *ExecutionTracker) ListRecentExecutions(ctx context.Context, limit int) ([]ExecutionRecord, error) {
	return et.cf.ListRecentExecutions(ctx, limit)
}

// AggregateMetrics computes summary statistics across all recorded executions.
func (et *ExecutionTracker) AggregateMetrics(ctx context.Context) (map[string]*ExecutionMetrics, error) {
	records, err := et.cf.ListRecentExecutions(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("list executions: %w", err)
	}

	agg := make(map[string]*ExecutionMetrics)
	for _, r := range records {
		m, ok := agg[r.ToolID]
		if !ok {
			agg[r.ToolID] = &ExecutionMetrics{
				Duration:       r.Metrics.Duration,
				MemoryBytes:    r.Metrics.MemoryBytes,
				CPUMillicores:  r.Metrics.CPUMillicores,
				CallCount:      1,
				ErrorCount:     r.Metrics.ErrorCount,
				TotalCalls:     1,
				LastExecutedAt: r.Timestamp,
			}
			continue
		}
		n := m.CallCount + 1
		m.Duration = time.Duration(
			int64(m.Duration)*m.CallCount/int64(n) + int64(r.Metrics.Duration)/int64(n),
		)
		m.MemoryBytes = (m.MemoryBytes*m.CallCount + r.Metrics.MemoryBytes) / n
		m.CPUMillicores = (m.CPUMillicores*m.CallCount + r.Metrics.CPUMillicores) / n
		m.CallCount = n
		m.ErrorCount += r.Metrics.ErrorCount
		m.TotalCalls++
		if r.Timestamp.After(m.LastExecutedAt) {
			m.LastExecutedAt = r.Timestamp
		}
		agg[r.ToolID] = m
	}

	return agg, nil
}
