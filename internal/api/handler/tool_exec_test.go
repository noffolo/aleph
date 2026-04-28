package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sort"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

func setupToolExecRepo(t *testing.T) *repository.MetadataRepository {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	queries := []string{
		`CREATE TABLE IF NOT EXISTS system_tools (
			id TEXT PRIMARY KEY, name TEXT, description TEXT, code TEXT,
			category TEXT, version TEXT, health_status TEXT, source_type TEXT,
			last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_api_keys (
			id TEXT PRIMARY KEY, project_id TEXT, label TEXT, key TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_agents (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, provider TEXT,
			model TEXT, api_key TEXT, system_prompt TEXT, skill_ids TEXT, base_url TEXT DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS system_skills (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, description TEXT, tool_ids TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS system_tasks (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, source_type TEXT,
			config_json TEXT, status TEXT, progress INTEGER,
			schedule TEXT DEFAULT '', is_predictive INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_chat_history (
			id TEXT PRIMARY KEY, project_id TEXT, agent_id TEXT, role TEXT,
			content TEXT, tool_call TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_notification_channels (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, type TEXT, config_json TEXT
		)`,
	}
	for _, q := range queries {
		_, err := db.Exec(q)
		require.NoError(t, err)
	}

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

func newToolExecMux(t *testing.T) (*ToolExecuteHandler, *http.ServeMux) {
	t.Helper()
	repo := setupToolExecRepo(t)
	h := NewToolExecuteHandler(repo, nil, nil)
	require.NotNil(t, h)

	// Trigger registry initialization (no broker → OSINT tools skipped)
	_ = h.Registry()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/tools/register", h.HandleRegister)
	mux.HandleFunc("GET /api/v1/tools/categories", h.HandleListCategories)
	mux.HandleFunc("GET /api/v1/tools/execute/{category}/{name}", h.HandleListToolsByCategory)
	mux.HandleFunc("POST /api/v1/tools/execute/{category}/{name}", h.HandleExecuteTool)
	mux.HandleFunc("POST /api/v1/tools/call", h.HandleCallTool)
	return h, mux
}

func TestToolExecuteHandler_HandleRegister(t *testing.T) {
	_, mux := newToolExecMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/register", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	err := json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	registered, ok := body["registered"].([]any)
	require.True(t, ok, "response should have 'registered' array")

	assert.GreaterOrEqual(t, len(registered), 3, "should register at least 3 finance tools")

	var registeredIDs []string
	for _, id := range registered {
		registeredIDs = append(registeredIDs, id.(string))
	}
	t.Logf("registered tools: %v", registeredIDs)
}

func TestToolExecuteHandler_HandleRegister_WrongMethod(t *testing.T) {
	_, mux := newToolExecMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/register", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Result().StatusCode)
}

func TestToolExecuteHandler_HandleListCategories(t *testing.T) {
	h, mux := newToolExecMux(t)
	_ = h // h is used implicitly via the mux, but the call h.Registry() already happened

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/categories", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var categories []string
	err := json.NewDecoder(resp.Body).Decode(&categories)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"finance", "human-ecosystems"}, categories)
}

func TestToolExecuteHandler_HandleExecuteTool_Finance(t *testing.T) {
	_, mux := newToolExecMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute/finance/finance_prophet_forecast", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestToolExecuteHandler_HandleExecuteTool_UnknownCategory(t *testing.T) {
	_, mux := newToolExecMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute/unknown/foo", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestToolExecuteHandler_HandleExecuteTool_UnknownName(t *testing.T) {
	_, mux := newToolExecMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute/finance/unknown_tool", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	// Registry returns an error for unknown tool → 500
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestToolExecuteHandler_HandleListToolsByCategory_Finance(t *testing.T) {
	_, mux := newToolExecMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/execute/finance/list", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var toolDefs []tools.ToolDefinition
	err := json.NewDecoder(resp.Body).Decode(&toolDefs)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(toolDefs), 3)
	sort.Slice(toolDefs, func(i, j int) bool { return toolDefs[i].Name < toolDefs[j].Name })
	assert.Equal(t, "finance_openbb_market_data", toolDefs[0].Name)
}

func TestToolExecuteHandler_HandleListToolsByCategory_Unknown(t *testing.T) {
	_, mux := newToolExecMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/execute/unknown/list", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}
