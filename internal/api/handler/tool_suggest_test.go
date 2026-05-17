package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/tools/adaptation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

func setupDiscoveryEngine(t *testing.T) *mcp.DiscoveryEngine {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	metaRepo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	cfg := mcp.DiscoveryConfig{ServerURIs: []string{}}
	return mcp.NewDiscoveryEngine(slog.Default(), metaRepo, cfg)
}

func TestNewToolSuggestHandler(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, nil)
	assert.NotNil(t, h)
	assert.NotNil(t, h.pending)
}

func TestToolSuggestHandler_ServeHTTP_Routing(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, nil)

	tests := []struct {
		path   string
		method string
		want   int
	}{
		{"/api/v1/tools/suggest", "POST", http.StatusBadRequest},
		{"/api/v1/tools/suggest/approve", "POST", http.StatusBadRequest},
		{"/api/v1/tools/suggest", "GET", http.StatusMethodNotAllowed},
		{"/api/v1/tools/unknown", "POST", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader([]byte(`{}`)))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			assert.Equal(t, tt.want, w.Code)
		})
	}
}

func TestToolSuggestHandler_HandleSuggest_EmptyBody(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/tools/suggest", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp suggestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp.Error, "name is required")
}

func TestToolSuggestHandler_HandleSuggest_InvalidJSON(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/tools/suggest", bytes.NewReader([]byte(`{invalid`)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp suggestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp.Error, "invalid JSON")
}

func TestToolSuggestHandler_HandleSuggest_DiscoveryNoURLs(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, []string{})
	req := httptest.NewRequest("POST", "/api/v1/tools/suggest", bytes.NewReader([]byte(`{"name":"test-tool"}`)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp suggestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp.Error, "tool discovery failed")
}

func TestHandleSuggest_NoServersConfigured(t *testing.T) {
	h := NewToolSuggestHandler(nil, nil, nil)
	body, _ := json.Marshal(suggestRequestBody{Name: "tool_x"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/suggest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleSuggest(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestToolSuggestHandler_HandleApprove_EmptyBody(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/tools/suggest/approve", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp approveResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp.Error, "name and suggestion_id are required")
}

func TestToolSuggestHandler_HandleApprove_NotFound(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, nil)
	body := `{"name":"test","suggestion_id":"sug-999"}`
	req := httptest.NewRequest("POST", "/api/v1/tools/suggest/approve", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp approveResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp.Error, "suggestion not found")
}

func TestHandleApprove_NameMismatch(t *testing.T) {
	h := NewToolSuggestHandler(nil, nil, nil)
	h.pending["sug-2"] = &pendingSuggestion{
		ToolDef: mcp.ToolDefinition{Name: "real_name"},
	}
	body, _ := json.Marshal(approveRequestBody{Name: "wrong_name", SuggestionID: "sug-2"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/suggest/approve", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleApprove(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStorePending_NonNil(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, nil)
	toolDef := mcp.ToolDefinition{Name: "test-tool", Description: "test"}
	result := &adaptation.AdaptationResult{Version: "1.0.0"}

	suggestionID := h.storePending(context.Background(), toolDef, result)
	assert.Contains(t, suggestionID, "sug-")
}

func TestToolSuggestHandler_DiscoverMCPTool_NoURLs(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, nil)
	_, err := h.discoverMCPTool(context.Background(), "test", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no MCP servers configured")
}

func TestToolSuggestHandler_DiscoverMCPTool_MCPScheme(t *testing.T) {
	engine := setupDiscoveryEngine(t)
	h := NewToolSuggestHandler(engine, nil, []string{"mcp://myserver:3000/tools"})
	_, err := h.discoverMCPTool(context.Background(), "test", "")
	assert.Error(t, err)
}

func TestToStageResultJSON_WithData(t *testing.T) {
	stages := []adaptation.StageResult{
		{Name: "verify", Passed: true, Message: "ok"},
		{Name: "adapt", Passed: false, Message: "failed"},
	}
	result := toStageResultJSON(stages)
	require.Len(t, result, 2)
	assert.Equal(t, "verify", result[0].Name)
	assert.True(t, result[0].Passed)
	assert.Equal(t, "adapt", result[1].Name)
	assert.False(t, result[1].Passed)
}

func TestServeHTTP_ImplementsHandler(t *testing.T) {
	var h http.Handler = (*ToolSuggestHandler)(nil)
	assert.Nil(t, h)
}
