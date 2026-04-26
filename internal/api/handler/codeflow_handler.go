// Package handler provides HTTP handler implementations for the aleph-v2 API.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ff3300/aleph-v2/internal/tools/codeflow"
)

// CodeFlowHandler handles HTTP requests for codeflow operations.
type CodeFlowHandler struct {
	codeFlow *codeflow.CodeFlow
}

// NewCodeFlowHandler creates a new CodeFlowHandler.
func NewCodeFlowHandler(cf *codeflow.CodeFlow) *CodeFlowHandler {
	return &CodeFlowHandler{codeFlow: cf}
}

// HandleGetGraph handles GET /api/v1/codeflow/graph?tool_id=X.
// Returns JSON ToolExecutionGraph for the given tool.
func (h *CodeFlowHandler) HandleGetGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolID := r.URL.Query().Get("tool_id")
	if toolID == "" {
		writeError(w, "tool_id is required", http.StatusBadRequest)
		return
	}

	graph, err := h.codeFlow.GetToolGraph(r.Context(), toolID)
	if err != nil {
		writeError(w, "failed to get execution graph: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

// HandleGetMetrics handles GET /api/v1/codeflow/metrics?tool_id=X.
// Returns JSON ExecutionMetrics for the given tool.
func (h *CodeFlowHandler) HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolID := r.URL.Query().Get("tool_id")
	if toolID == "" {
		writeError(w, "tool_id is required", http.StatusBadRequest)
		return
	}

	metrics, err := h.codeFlow.GetMetrics(r.Context(), toolID)
	if err != nil {
		writeError(w, "failed to get metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// HandleListExecutions handles GET /api/v1/codeflow/executions?tool_id=X&limit=N.
// If tool_id is provided, returns records for that tool; otherwise returns recent executions.
// If limit is provided and valid, caps the result count; otherwise defaults to all.
func (h *CodeFlowHandler) HandleListExecutions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolID := r.URL.Query().Get("tool_id")
	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	var records []codeflow.ExecutionRecord
	var err error

	if toolID != "" {
		records, err = h.codeFlow.GetRecords(r.Context(), toolID)
	} else {
		records, err = h.codeFlow.ListRecentExecutions(r.Context(), limit)
	}

	if err != nil {
		writeError(w, "failed to list executions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if records == nil {
		records = []codeflow.ExecutionRecord{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// HandleListEngines handles GET /api/v1/codeflow/engines.
// Returns JSON array of available engine names.
func (h *CodeFlowHandler) HandleListEngines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	engines := h.codeFlow.ListEngines()
	if engines == nil {
		engines = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(engines)
}
