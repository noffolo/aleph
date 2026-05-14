package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ff3300/aleph-v2/internal/tools/codeflow"
)

func TestNewCodeFlowHandler(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	h := NewCodeFlowHandler(cf)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.codeFlow == nil {
		t.Fatal("expected non-nil codeflow")
	}
}

func TestHandleGetGraph_MethodNotAllowed(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodPost, "/graph?tool_id=t1", nil)
	rr := httptest.NewRecorder()
	h.HandleGetGraph(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleGetGraph_MissingToolID(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodGet, "/graph", nil)
	rr := httptest.NewRecorder()
	h.HandleGetGraph(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGetGraph_ToolNotFound(t *testing.T) {
	// GetToolGraph returns empty graph for unknown tools (not error)
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodGet, "/graph?tool_id=nonexistent", nil)
	rr := httptest.NewRecorder()
	h.HandleGetGraph(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 (returns empty graph), got %d", rr.Code)
	}
}

func TestHandleGetGraph_EmptyRecorded(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	h := NewCodeFlowHandler(cf)
	req := httptest.NewRequest(http.MethodGet, "/graph?tool_id=t1", nil)
	rr := httptest.NewRecorder()
	h.HandleGetGraph(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 (returns empty graph), got %d", rr.Code)
	}
}

func TestHandleGetMetrics_MethodNotAllowed(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodDelete, "/metrics?tool_id=t1", nil)
	rr := httptest.NewRecorder()
	h.HandleGetMetrics(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleGetMetrics_MissingToolID(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.HandleGetMetrics(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGetMetrics_ToolNotFound(t *testing.T) {
	// GetMetrics returns empty metrics for unknown tools (not error)
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodGet, "/metrics?tool_id=unknown", nil)
	rr := httptest.NewRecorder()
	h.HandleGetMetrics(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 (returns empty metrics), got %d", rr.Code)
	}
}

func TestHandleListExecutions_MethodNotAllowed(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodPost, "/executions", nil)
	rr := httptest.NewRecorder()
	h.HandleListExecutions(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleListExecutions_EmptyDefault(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodGet, "/executions", nil)
	rr := httptest.NewRecorder()
	h.HandleListExecutions(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var records []codeflow.ExecutionRecord
	if err := json.NewDecoder(rr.Body).Decode(&records); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestHandleListExecutions_WithToolID(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodGet, "/executions?tool_id=t1", nil)
	rr := httptest.NewRecorder()
	h.HandleListExecutions(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleListExecutions_WithPagination(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodGet, "/executions?tool_id=t1&page=1&per_page=10", nil)
	rr := httptest.NewRecorder()
	h.HandleListExecutions(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleListExecutions_EmptyResults(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodGet, "/executions?tool_id=unknown", nil)
	rr := httptest.NewRecorder()
	h.HandleListExecutions(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var records []codeflow.ExecutionRecord
	if err := json.NewDecoder(rr.Body).Decode(&records); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if records == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
}

func TestHandleListEngines_Success(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodGet, "/engines", nil)
	rr := httptest.NewRecorder()
	h.HandleListEngines(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var engines []string
	if err := json.NewDecoder(rr.Body).Decode(&engines); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(engines) != 3 {
		t.Errorf("expected 3 engines, got %d", len(engines))
	}
}

func TestHandleListEngines_MethodNotAllowed(t *testing.T) {
	h := NewCodeFlowHandler(codeflow.NewCodeFlow())
	req := httptest.NewRequest(http.MethodPost, "/engines", nil)
	rr := httptest.NewRecorder()
	h.HandleListEngines(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}
