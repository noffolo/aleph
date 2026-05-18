// Package codeflow provides tool execution graph tracking, dependency visualization,
// and metrics aggregation for the aleph-v2 system.
// Category: "codeflow", SourceType: "package"
package codeflow

import (
	"context"
	"sync"
	"time"
)

// NodeType represents the type of a node in the execution graph.
type NodeType string

const (
	NodeTypeTool       NodeType = "tool"
	NodeTypeData       NodeType = "data"
	NodeTypeDependency NodeType = "dependency"
)

// ExecutionMetrics captures runtime metrics for a single tool execution.
type ExecutionMetrics struct {
	Duration       time.Duration `json:"duration"`
	MemoryBytes    int64         `json:"memory_bytes"`
	CPUMillicores  int64         `json:"cpu_millicores"`
	CallCount      int64         `json:"call_count"`
	ErrorCount     int64         `json:"error_count"`
	TotalCalls     int64         `json:"total_calls"`
	LastExecutedAt time.Time     `json:"last_executed_at"`
}

// ToolExecutionGraph represents the execution graph for a tool.
type ToolExecutionGraph struct {
	ToolID    string      `json:"tool_id"`
	Nodes     []GraphNode `json:"nodes"`
	Edges     []GraphEdge `json:"edges"`
	CreatedAt time.Time   `json:"created_at"`
}

// GraphNode represents a single node in the execution graph.
type GraphNode struct {
	ID       string           `json:"id"`
	Label    string           `json:"label"`
	NodeType NodeType         `json:"node_type"`
	Metrics  ExecutionMetrics `json:"metrics,omitempty"`
}

// GraphEdge represents a directed edge between two nodes.
type GraphEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	EdgeType string `json:"edge_type"` // "dependency", "dataflow", "calls"
	Weight   int    `json:"weight"`
}

// ExecutionRecord stores a single execution event.
type ExecutionRecord struct {
	ToolID    string           `json:"tool_id"`
	UserID    string           `json:"user_id,omitempty"`
	Metrics   ExecutionMetrics `json:"metrics"`
	Status    string           `json:"status"` // "success", "error", "timeout"
	Error     string           `json:"error,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}

// Anomaly describes a detected anomaly in tool execution.
type Anomaly struct {
	ToolID      string    `json:"tool_id"`
	Type        string    `json:"type"`     // "duration", "error_rate", "memory_spike"
	Severity    string    `json:"severity"` // "low", "medium", "high"
	Description string    `json:"description"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ToolDependency describes a dependency relationship between tools.
type ToolDependency struct {
	ToolID         string `json:"tool_id"`
	DependencyID   string `json:"dependency_id"`
	DependencyType string `json:"dependency_type"` // "import", "data", "service"
	Required       bool   `json:"required"`
}

// CodeFlow manages tool execution tracking and analysis.
type CodeFlow struct {
	mu         sync.RWMutex
	records    []ExecutionRecord
	graphs     map[string]*ToolExecutionGraph
	metrics    map[string]*ExecutionMetrics
	maxRecords int
}

// NewCodeFlow creates a new CodeFlow instance.
func NewCodeFlow() *CodeFlow {
	return &CodeFlow{
		records:    make([]ExecutionRecord, 0, 1000),
		graphs:     make(map[string]*ToolExecutionGraph),
		metrics:    make(map[string]*ExecutionMetrics),
		maxRecords: 10000,
	}
}

// ListEngines returns the available tracking engines.
func (cf *CodeFlow) ListEngines() []string {
	return []string{"execution_tracker", "graph_analyzer", "metrics_aggregator"}
}

// RecordExecution stores an execution record and updates metrics.
func (cf *CodeFlow) RecordExecution(ctx context.Context, toolID string, metrics ExecutionMetrics) error {
	record := ExecutionRecord{
		ToolID:    toolID,
		Metrics:   metrics,
		Status:    "success",
		Timestamp: time.Now(),
	}
	if metrics.ErrorCount > 0 {
		record.Status = "error"
	}

	cf.mu.Lock()
	defer cf.mu.Unlock()

	// Append record, trimming if over limit
	cf.records = append(cf.records, record)
	if len(cf.records) > cf.maxRecords {
		cf.records = cf.records[len(cf.records)-cf.maxRecords:]
	}

	// Update aggregated metrics
	existing, ok := cf.metrics[toolID]
	if !ok {
		cf.metrics[toolID] = &ExecutionMetrics{
			Duration:       metrics.Duration,
			MemoryBytes:    metrics.MemoryBytes,
			CPUMillicores:  metrics.CPUMillicores,
			CallCount:      1,
			ErrorCount:     metrics.ErrorCount,
			TotalCalls:     1,
			LastExecutedAt: time.Now(),
		}
	} else {
		// Rolling average for duration
		n := existing.CallCount + 1
		existing.Duration = time.Duration(
			int64(existing.Duration)*existing.CallCount/int64(n) +
				int64(metrics.Duration)/int64(n),
		)
		existing.MemoryBytes = (existing.MemoryBytes*existing.CallCount + metrics.MemoryBytes) / n
		existing.CPUMillicores = (existing.CPUMillicores*existing.CallCount + metrics.CPUMillicores) / n
		existing.CallCount = n
		existing.ErrorCount += metrics.ErrorCount
		existing.TotalCalls++
		existing.LastExecutedAt = time.Now()
	}

	return nil
}

// GetToolGraph returns the execution graph for a given tool.
func (cf *CodeFlow) GetToolGraph(ctx context.Context, toolID string) (*ToolExecutionGraph, error) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	graph, ok := cf.graphs[toolID]
	if !ok {
		return &ToolExecutionGraph{
			ToolID:    toolID,
			Nodes:     []GraphNode{},
			Edges:     []GraphEdge{},
			CreatedAt: time.Now(),
		}, nil
	}
	return graph, nil
}

// SetToolGraph stores an execution graph for a tool.
func (cf *CodeFlow) SetToolGraph(ctx context.Context, toolID string, graph *ToolExecutionGraph) error {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	cf.graphs[toolID] = graph
	return nil
}

// GetMetrics returns aggregated metrics for a given tool.
func (cf *CodeFlow) GetMetrics(ctx context.Context, toolID string) (*ExecutionMetrics, error) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	metrics, ok := cf.metrics[toolID]
	if !ok {
		return &ExecutionMetrics{}, nil
	}
	return metrics, nil
}

// ListRecentExecutions returns the most recent execution records.
func (cf *CodeFlow) ListRecentExecutions(ctx context.Context, limit int) ([]ExecutionRecord, error) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	if limit <= 0 || limit > len(cf.records) {
		limit = len(cf.records)
	}

	result := make([]ExecutionRecord, limit)
	copy(result, cf.records[len(cf.records)-limit:])
	return result, nil
}

// GetRecords returns all records for a given tool.
func (cf *CodeFlow) GetRecords(ctx context.Context, toolID string) ([]ExecutionRecord, error) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	var result []ExecutionRecord
	for _, r := range cf.records {
		if r.ToolID == toolID {
			result = append(result, r)
		}
	}
	return result, nil
}
