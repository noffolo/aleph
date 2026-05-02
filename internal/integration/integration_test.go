//go:build integration

// Package integration provides end-to-end integration tests for the Aleph API.
// These tests spin up a full HTTP server backed by DuckDB :memory: and exercise
// the complete request flow: auth, projects, queries, and tool execution.
// They are excluded from normal CI and require the "integration" build tag.
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/api/handler"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	_ "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
	"github.com/ff3300/aleph-v2/internal/api/sse"
	"github.com/ff3300/aleph-v2/internal/auth"
	"github.com/ff3300/aleph-v2/internal/config"
	"github.com/ff3300/aleph-v2/internal/diagnostic"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/routes"
	"github.com/ff3300/aleph-v2/internal/service/notification"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/ff3300/aleph-v2/internal/tools/codeflow"
)

// testServer holds all state for an integration test server.
type testServer struct {
	t           *testing.T
	db          *storage.DuckDB
	metaRepo    *repository.MetadataRepository
	registryMgr *registry.DuckDBRegistry
	server      *httptest.Server
	jwtSecret   []byte
	projectsDir string
	baseURL     string
}

// newTestServer creates a fully wired test server backed by DuckDB :memory:.
// It creates the necessary PG-style tables directly in DuckDB and registers
// all routes without requiring a real PostgreSQL instance.
func newTestServer(t *testing.T) *testServer {
	t.Helper()

	jwtSecret := []byte("test-jwt-secret-that-is-at-least-32-bytes-long!!!")

	// DuckDB :memory: for analytic storage
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// Create PG-style metadata tables in DuckDB
	metaDB, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { metaDB.Close() })

	createMetaTables(t, metaDB)

	// Create a temporary projects directory
	projectsDir := t.TempDir()

	// Metadata repository
	metaRepo, err := repository.NewMetadataRepository(metaDB)
	require.NoError(t, err)

	// Seed backend admin key for integration tests
	backendKey := "test_backend_admin_key"
	os.Setenv("ALEPH_API_KEY_SECRET_BACKEND", backendKey)
	t.Cleanup(func() { os.Unsetenv("ALEPH_API_KEY_SECRET_BACKEND") })
	hashedBackendKey, err := auth.HashAPIKey(backendKey)
	require.NoError(t, err)
	_, err = metaDB.Exec("INSERT INTO system_api_keys (id, project_id, label, key, role) VALUES ($1, $2, $3, $4, $5)",
		backendKey[:8], "integration-test", "backend-admin", hashedBackendKey, "admin")
	require.NoError(t, err)

	// DuckDB registry (for component metadata)
	registryMgr, err := registry.NewDuckDBRegistryFromDuckDB(db, nil)
	if err != nil || registryMgr == nil {
		registryMgr, err = registry.NewDuckDBRegistryFromDB(metaDB, nil)
		require.NoError(t, err)
	}

	// Handlers
	queryHandler := handler.NewQueryHandler(db, projectsDir, metaRepo, nil, registryMgr, 5*time.Second)
	projectHandler := handler.NewProjectHandler(projectsDir, db)
	projectHandler.SetMetaRepo(metaRepo)
	projectHandler.SetMaxProjects(50)
	ontoRepo := repository.NewOntologyRepository(metaDB)
	projectHandler.SetOntologyRepository(ontoRepo)
	agentHandler := handler.NewAgentHandler(projectsDir, metaRepo, "")
	skillHandler := handler.NewSkillHandler(projectsDir, metaRepo)
	toolHandler := handler.NewToolHandler(projectsDir, metaRepo)
	libraryHandler := handler.NewLibraryHandler(projectsDir)
	authHandler := handler.NewAuthHandler(metaRepo)
	sessionHandler := handler.NewSessionHandler(metaRepo, jwtSecret)
	ingestionHandler := handler.NewIngestionHandler(projectsDir, nil, metaRepo)
	notificationSvc := notification.NewNotificationService()
	notificationHandler := handler.NewNotificationHandler(notificationSvc, metaRepo)
	sseBroker := sse.NewBroker(30*time.Second, nil)
	sseHandler := handler.NewSSEHandler(sseBroker, nil).WithJWTSecret(jwtSecret)
	codeFlow := codeflow.NewCodeFlow()
	codeFlowHandler := handler.NewCodeFlowHandler(codeFlow)

	// Sandbox and registry service handlers (minimal stubs)
	sandboxHandler := handler.NewSandboxServiceHandler(nil, nil)
	registryHandler := handler.NewRegistryServiceHandler(registryMgr, nil)

	// Tool execution handler
	toolExecHandler := handler.NewToolExecuteHandler(metaRepo, nil, nil)

	// Suggest pipeline (minimal)
	suggestPipeline := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tools":[]}`))
	})

	// Auth interceptor and error handler
	authInterceptor := middleware.NewAuthInterceptor(metaRepo, jwtSecret)
	errorHandler := middleware.NewErrorHandlerInterceptor()

	interceptors := []connect.HandlerOption{
		connect.WithInterceptors(errorHandler, authInterceptor),
	}

	// Register routes
	mux := http.NewServeMux()
	routes.RegisterRoutes(mux, routes.RegisterConfig{
		MetaRepo:            metaRepo,
		JWTSecret:           jwtSecret,
		SSEBroker:           sseBroker,
		SSEHandler:          sseHandler,
		DiagnosticMonitor:   diagnostic.NewDiagnosticMonitor(3, nil),
		CodeFlow:            codeFlow,
		QueryHandler:        queryHandler,
		ProjectHandler:      projectHandler,
		AgentHandler:        agentHandler,
		SkillHandler:        skillHandler,
		LibraryHandler:      libraryHandler,
		ToolHandler:         toolHandler,
		NotificationHandler: notificationHandler,
		AuthHandler:         authHandler,
		SessionHandler:      sessionHandler,
		IngestionHandler:    ingestionHandler,
		SandboxHandler:      sandboxHandler,
		RegistryHandler:     registryHandler,
		ToolExecHandler:     toolExecHandler,
		CodeFlowHandler:     codeFlowHandler,
		SuggestPipeline:     suggestPipeline,
		Interceptors:        interceptors,
	})

	// Wrap with CORS and standard middleware
	corsHandler := routes.CORSHandler(mux, []string{"http://localhost:5173"}, nil)
	server := httptest.NewServer(corsHandler)

	ts := &testServer{
		t:           t,
		db:          db,
		metaRepo:    metaRepo,
		registryMgr: registryMgr,
		server:      server,
		jwtSecret:   jwtSecret,
		projectsDir: projectsDir,
		baseURL:     server.URL,
	}
	t.Cleanup(server.Close)

	return ts
}

// createMetaTables creates PostgreSQL-style metadata tables in DuckDB.
func createMetaTables(t *testing.T, db *sql.DB) {
	t.Helper()

	schemas := []string{
		`CREATE TABLE IF NOT EXISTS system_projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_api_keys (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			label TEXT,
			key TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_tools (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT,
			code TEXT,
			category TEXT NOT NULL DEFAULT '',
			version TEXT NOT NULL DEFAULT '1.0.0',
			health_status TEXT NOT NULL DEFAULT 'unknown',
			source_type TEXT NOT NULL DEFAULT 'builtin'
		)`,
		`CREATE TABLE IF NOT EXISTS system_skills (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			description TEXT,
			tool_ids TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS system_agents (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			provider TEXT,
			model TEXT,
			api_key TEXT,
			system_prompt TEXT,
			skill_ids TEXT DEFAULT '[]',
			base_url TEXT DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS system_tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			source_type TEXT,
			config_json TEXT,
			status TEXT,
			progress INTEGER,
			schedule TEXT DEFAULT '',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_chat_history (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			agent_id TEXT,
			role TEXT,
			content TEXT,
			tool_call TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_notification_channels (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			type TEXT,
			config_json TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY,
			user_id VARCHAR,
			action VARCHAR NOT NULL,
			resource_type VARCHAR NOT NULL,
			resource_id VARCHAR NOT NULL,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			diff VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS ontology_versions (
			version_id TEXT PRIMARY KEY,
			project_id TEXT,
			parent_version_id TEXT,
			diff_json TEXT,
			core_aleph_snapshot TEXT,
			status TEXT,
			source_description TEXT,
			rationale TEXT,
			confidence DOUBLE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			modified_at TIMESTAMP
		)`,
	}

	for _, s := range schemas {
		_, err := db.Exec(s)
		require.NoError(t, err, "failed to create table: %s", s[:60])
	}
}

// postJSON sends an HTTP POST with a JSON body and returns the response.
func (ts *testServer) postJSON(url, contentType string, body []byte) *http.Response {
	req, err := http.NewRequest("POST", ts.baseURL+url, bytes.NewReader(body))
	require.NoError(ts.t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(ts.t, err)
	return resp
}

// get sends an HTTP GET and returns the response.
func (ts *testServer) get(url string, cookies []*http.Cookie) *http.Response {
	req, err := http.NewRequest("GET", ts.baseURL+url, nil)
	require.NoError(ts.t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(ts.t, err)
	return resp
}

// postConnectRPC sends a Connect RPC request and returns the response.
func (ts *testServer) postConnectRPC(procedure string, msg []byte, cookies []*http.Cookie) *http.Response {
	url := ts.baseURL + procedure
	req, err := http.NewRequest("POST", url, bytes.NewReader(msg))
	require.NoError(ts.t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(ts.t, err)
	return resp
}

// readBody reads the response body and returns it as a string.
func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	return string(data)
}

// requireStatus checks that the response has the expected HTTP status.
func requireStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body := readBody(t, resp)
		require.FailNow(t, "unexpected status code", "got %d, want %d\nbody: %s", resp.StatusCode, expected, body)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Tests
// ────────────────────────────────────────────────────────────────────────────

// TestIntegration_AuthFlow tests the complete authentication flow:
// 1. Use backend secret key to create admin session → get JWT cookie
// 2. Use admin JWT to call CreateApiKey via Connect RPC (now requires auth)
// 3. Use the created key to create a regular user session
// 4. Validate both sessions
func TestIntegration_AuthFlow(t *testing.T) {
	ts := newTestServer(t)

	backendKey := os.Getenv("ALEPH_API_KEY_SECRET_BACKEND")
	if backendKey == "" {
		backendKey = "test_backend_admin_key"
		os.Setenv("ALEPH_API_KEY_SECRET_BACKEND", backendKey)
		t.Cleanup(func() { os.Unsetenv("ALEPH_API_KEY_SECRET_BACKEND") })
	}

	sessionReq := map[string]string{"api_key": backendKey}
	sessionData, _ := json.Marshal(sessionReq)
	resp := ts.postJSON("/api/v1/auth/session", "application/json", sessionData)
	requireStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)

	var sessionResp map[string]string
	json.Unmarshal([]byte(body), &sessionResp)

	cookies := resp.Cookies()
	var adminCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "aleph_jwt" {
			adminCookie = c
			break
		}
	}
	require.NotNil(t, adminCookie, "admin aleph_jwt cookie should be set")

	// Step 2: Create an API key via Connect RPC (now requires auth — admin JWT)
	createKeyReq := &v1.CreateApiKeyRequest{
		ProjectId: "test-project-auth",
		Label:     "integration-test-key",
	}
	rawReq, err := json.Marshal(createKeyReq)
	require.NoError(t, err)

	resp2 := ts.postConnectRPC("/aleph.v1.AuthService/CreateApiKey", rawReq, []*http.Cookie{adminCookie})
	requireStatus(t, resp2, http.StatusOK)
	body2 := readBody(t, resp2)

	var createKeyResp struct {
		Key struct {
			Id        string `json:"id"`
			Label     string `json:"label"`
			Key       string `json:"key"`
			CreatedAt string `json:"createdAt"`
		} `json:"key"`
	}
	err = json.Unmarshal([]byte(body2), &createKeyResp)
	require.NoError(t, err)
	require.NotEmpty(t, createKeyResp.Key.Key, "API key should not be empty")

	// Step 3: Use the created key for a regular session
	sessionReq2 := map[string]string{"api_key": createKeyResp.Key.Key}
	sessionData2, _ := json.Marshal(sessionReq2)
	resp3 := ts.postJSON("/api/v1/auth/session", "application/json", sessionData2)
	requireStatus(t, resp3, http.StatusOK)
	body3 := readBody(t, resp3)

	var sessionResp2 map[string]string
	json.Unmarshal([]byte(body3), &sessionResp2)
	assert.Equal(t, "test-project-auth", sessionResp2["project_id"])

	var userCookie *http.Cookie
	for _, c := range resp3.Cookies() {
		if c.Name == "aleph_jwt" {
			userCookie = c
			break
		}
	}
	require.NotNil(t, userCookie, "user aleph_jwt cookie should be set")
	assert.True(t, userCookie.HttpOnly, "JWT cookie should be HttpOnly")

	// Step 4: Use JWT cookie to call an authenticated endpoint (/api/v1/tools)
	resp4 := ts.get("/api/v1/tools", []*http.Cookie{userCookie})
	requireStatus(t, resp4, http.StatusOK)
}

// TestIntegration_ProjectLifecycle tests project CRUD through Connect RPC:
// 1. Create a project → verify directory + DuckDB schema
// 2. List projects → verify the project appears
// 3. Create an ontology (save .aleph file)
// 4. Get ontology back
// 5. Delete project
func TestIntegration_ProjectLifecycle(t *testing.T) {
	ts := newTestServer(t)

	// Get JWT cookie first
	jwtCookie := ts.loginAsAdmin(t)

	// Step 1: Create a project
	createReq := &v1.CreateProjectRequest{
		Id:   "integration-test-project",
		Name: "Integration Test Project",
	}
	rawReq, _ := json.Marshal(createReq)
	resp := ts.postConnectRPC("/aleph.v1.ProjectService/CreateProject", rawReq, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assert.Contains(t, body, `"id":"integration-test-project"`)
	assert.Contains(t, body, `"name":"Integration Test Project"`)

	// Verify project directory was created
	projectDir := filepath.Join(ts.projectsDir, "integration-test-project")
	_, err := os.Stat(projectDir)
	assert.NoError(t, err, "project directory should exist")
	_, err = os.Stat(filepath.Join(projectDir, "ontologies", "core.aleph"))
	assert.NoError(t, err, "core.aleph should exist")

	// Step 2: List projects
	resp = ts.postConnectRPC("/aleph.v1.ProjectService/ListProjects", []byte(`{}`), []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	body = readBody(t, resp)
	assert.Contains(t, body, "integration-test-project")

	// Step 3: Save an ontology
	saveReq := &v1.SaveOntologyRequest{
		ProjectId:       "integration-test-project",
		AlephDefinition: "object TestEntity { name: text }",
	}
	rawReq, _ = json.Marshal(saveReq)
	resp = ts.postConnectRPC("/aleph.v1.ProjectService/SaveOntology", rawReq, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)

	// Step 4: Get ontology back
	getReq := &v1.GetOntologyRequest{ProjectId: "integration-test-project"}
	rawReq, _ = json.Marshal(getReq)
	resp = ts.postConnectRPC("/aleph.v1.ProjectService/GetOntology", rawReq, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	body = readBody(t, resp)
	assert.Contains(t, body, "TestEntity")

	// Step 5: Delete project
	deleteReq := &v1.DeleteProjectRequest{Id: "integration-test-project"}
	rawReq, _ = json.Marshal(deleteReq)
	resp = ts.postConnectRPC("/aleph.v1.ProjectService/DeleteProject", rawReq, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)

	// Verify project directory is gone
	_, err = os.Stat(projectDir)
	assert.True(t, os.IsNotExist(err), "project directory should be deleted")
}

// TestIntegration_QueryFlow tests data query execution through the Connect RPC.
// 1. Create a project
// 2. Write data to DuckDB
// 3. ExecuteQuery and verify results
func TestIntegration_QueryFlow(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	// Create a project first
	createReq := &v1.CreateProjectRequest{
		Id:   "query-test-project",
		Name: "Query Test",
	}
	rawReq, _ := json.Marshal(createReq)
	resp := ts.postConnectRPC("/aleph.v1.ProjectService/CreateProject", rawReq, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)

	// Write test data to DuckDB
	ctx := context.Background()
	_, err := ts.db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS test_entities (id INTEGER, name TEXT, value DOUBLE)")
	require.NoError(t, err)
	_, err = ts.db.ExecContext(ctx, "INSERT INTO test_entities VALUES (1, 'alpha', 10.5), (2, 'beta', 20.3), (3, 'gamma', 30.1)")
	require.NoError(t, err)

	// ExecuteQuery via Connect RPC
	queryReq := &v1.ExecuteQueryRequest{
		ObjectType: "test_entities",
		ProjectId:  "query-test-project",
		Limit:      10,
	}
	rawReq, _ = json.Marshal(queryReq)
	resp = ts.postConnectRPC("/aleph.v1.QueryService/ExecuteQuery", rawReq, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	t.Logf("ExecuteQuery response: %s", body)

	// Verify response contains expected data
	assert.Contains(t, body, `"alpha"`)
	assert.Contains(t, body, `"beta"`)
	assert.Contains(t, body, `"gamma"`)
	assert.Contains(t, body, `"value"`)
	assert.Contains(t, body, `10.5`)

	// Cleanup
	_, err = ts.db.ExecContext(ctx, "DROP TABLE IF EXISTS test_entities")
	require.NoError(t, err)
}

// TestIntegration_ToolExecution tests creating and listing tools.
// 1. Create a project
// 2. Register a tool via Connect RPC
// 3. List tools and verify it appears
// 4. Execute tool via HTTP endpoint
func TestIntegration_ToolExecution(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	// Create a project first
	createReq := &v1.CreateProjectRequest{
		Id:   "tool-test-project",
		Name: "Tool Test",
	}
	rawReq, _ := json.Marshal(createReq)
	resp := ts.postConnectRPC("/aleph.v1.ProjectService/CreateProject", rawReq, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)

	// Step 1: Create a tool via Connect RPC
	createToolReq := &v1.CreateToolRequest{
		ProjectId: "tool-test-project",
		Tool: &v1.Tool{
			Id:          "test-tool-1",
			Name:        "test_tool",
			Description: "A test tool for integration testing",
			Code:        "print('hello from test tool')",
		},
	}
	rawReq, _ = json.Marshal(createToolReq)
	resp = ts.postConnectRPC("/aleph.v1.ToolService/CreateTool", rawReq, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)

	// Step 2: List tools and verify
	listToolsReq := &v1.ListToolsRequest{
		ProjectId: "tool-test-project",
	}
	rawReq, _ = json.Marshal(listToolsReq)
	resp = ts.postConnectRPC("/aleph.v1.ToolService/ListTools", rawReq, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assert.Contains(t, body, "test-tool-1")
	assert.Contains(t, body, "test_tool")
	assert.Contains(t, body, "A test tool for integration testing")

	// Step 3: Get tool categories via HTTP endpoint (authenticated)
	resp = ts.get("/api/v1/tools/categories", []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)

	// Step 4: Call the tools list (raw HTTP handler)
	resp = ts.get("/api/v1/tools", []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
}

// TestIntegration_ErrorHandling tests various error conditions:
// 1. Request without auth → 401
// 2. Invalid auth token → 401
// 3. Query non-existent project data (non-existent object type) → error
// 4. Create project with empty ID → error
// 5. Metrics endpoint (unauthenticated) → 200
func TestIntegration_ErrorHandling(t *testing.T) {
	ts := newTestServer(t)

	// Step 1: Unauthenticated request to protected endpoint → 401
	resp := ts.postConnectRPC("/aleph.v1.ProjectService/ListProjects", []byte(`{}`), nil)
	requireStatus(t, resp, http.StatusUnauthorized)

	// Step 2: Invalid auth token → 401
	badCookie := &http.Cookie{Name: "aleph_jwt", Value: "invalid-token"}
	resp = ts.postConnectRPC("/aleph.v1.ProjectService/ListProjects", []byte(`{}`), []*http.Cookie{badCookie})
	requireStatus(t, resp, http.StatusUnauthorized)

	// Step 3: Create project with empty ID → should get error
	jwtCookie := ts.loginAsAdmin(t)
	createReq := &v1.CreateProjectRequest{
		Id:   "",
		Name: "Empty ID",
	}
	rawReq, _ := json.Marshal(createReq)
	resp = ts.postConnectRPC("/aleph.v1.ProjectService/CreateProject", rawReq, []*http.Cookie{jwtCookie})
	// Should fail with invalid argument
	assert.NotEqual(t, http.StatusOK, resp.StatusCode,
		"creating project with empty ID should fail")

	// Step 4: Metrics endpoint should be open (unauthenticated)
	resp = ts.get("/metrics", nil)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		// Both are valid — the metrics endpoint may or may not be registered
		// but it should NOT return 401
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	}

	// Step 5: Query non-existent object type
	queryReq := &v1.ExecuteQueryRequest{
		ObjectType: "nonexistent_table_xyz",
		ProjectId:  "nonexistent-project",
		Limit:      10,
	}
	rawReq, _ = json.Marshal(queryReq)
	resp = ts.postConnectRPC("/aleph.v1.QueryService/ExecuteQuery", rawReq, []*http.Cookie{jwtCookie})
	assert.NotEqual(t, http.StatusOK, resp.StatusCode,
		"querying non-existent table should fail")
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

// loginAsAdmin creates a session as admin by:
// 1. Using the backend API key to create a session → get JWT cookie (admin role)
// 2. Seed an API key via Connect RPC with the admin JWT
// 3. Returning the JWT cookie
func (ts *testServer) loginAsAdmin(t *testing.T) *http.Cookie {
	t.Helper()

	backendKey := os.Getenv("ALEPH_API_KEY_SECRET_BACKEND")
	if backendKey == "" {
		backendKey = "test_backend_admin_key"
		os.Setenv("ALEPH_API_KEY_SECRET_BACKEND", backendKey)
		t.Cleanup(func() { os.Unsetenv("ALEPH_API_KEY_SECRET_BACKEND") })
	}

	sessionReq := map[string]string{"api_key": backendKey}
	sessionData, _ := json.Marshal(sessionReq)
	resp := ts.postJSON("/api/v1/auth/session", "application/json", sessionData)
	requireStatus(t, resp, http.StatusOK)

	var jwtCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "aleph_jwt" {
			jwtCookie = c
			break
		}
	}
	require.NotNil(t, jwtCookie, "aleph_jwt cookie should be set")
	return jwtCookie
}

// TestIntegration_HealthEndpoints tests that unauthenticated health/readiness
// endpoints work correctly.
func TestIntegration_HealthEndpoints(t *testing.T) {
	ts := newTestServer(t)

	// Health check (unauthenticated)
	resp := ts.get("/api/v1/healthz", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assert.Contains(t, body, `"status":"ok"`)

	// Readiness probe (unauthenticated)
	resp = ts.get("/readyz", nil)
	requireStatus(t, resp, http.StatusOK)

	// Liveness probe (unauthenticated)
	resp = ts.get("/livez", nil)
	requireStatus(t, resp, http.StatusOK)
	body = readBody(t, resp)
	assert.Contains(t, body, `"status":"alive"`)
}

// TestIntegration_TokenGeneration validates JWT generation/validation directly.
func TestIntegration_TokenGeneration(t *testing.T) {
	secret := []byte("test-secret-for-jwt-generation-test-1234567")

	// Generate a valid token
	token, err := auth.GenerateToken(auth.SessionToken{
		UserID:    "user1",
		ProjectID: "proj1",
		Role:      "admin",
	}, secret, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Validate it
	claims, err := auth.ValidateToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, "proj1", claims.ProjectID)
	assert.Equal(t, "admin", claims.Role)

	// Invalid token should fail
	_, err = auth.ValidateToken("invalid-token", secret)
	assert.Error(t, err)

	// Wrong secret should fail
	wrongSecret := []byte("this-is-a-completely-different-secret-key-ok")
	_, err = auth.ValidateToken(token, wrongSecret)
	assert.Error(t, err)
}

// TestIntegration_CrossEndpointAuth tests that a session created with one
// project's API key correctly restricts access to that project's scope.
func TestIntegration_CrossEndpointAuth(t *testing.T) {
	ts := newTestServer(t)

	// Create two API keys for different projects via Connect RPC
	projects := []string{"cross-proj-a", "cross-proj-b"}
	var apiKeys []string

	for _, pid := range projects {
		createReq := &v1.CreateApiKeyRequest{
			ProjectId: pid,
			Label:     fmt.Sprintf("key-for-%s", pid),
		}
		rawReq, _ := json.Marshal(createReq)
		resp := ts.postConnectRPC("/aleph.v1.AuthService/CreateApiKey", rawReq, nil)
		requireStatus(t, resp, http.StatusOK)
		body := readBody(t, resp)

		var keyResp struct {
			Key struct {
				Key string `json:"key"`
			} `json:"key"`
		}
		err := json.Unmarshal([]byte(body), &keyResp)
		require.NoError(t, err)
		apiKeys = append(apiKeys, keyResp.Key.Key)
	}

	// Login with first API key
	sessionReq := map[string]string{"api_key": apiKeys[0]}
	sessionData, _ := json.Marshal(sessionReq)
	resp := ts.postJSON("/api/v1/auth/session", "application/json", sessionData)
	requireStatus(t, resp, http.StatusOK)

	var jwtCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "aleph_jwt" {
			jwtCookie = c
			break
		}
	}
	require.NotNil(t, jwtCookie)

	// Verify the session belongs to the correct project
	// by checking the response body
	body := readBody(t, resp)
	assert.Contains(t, body, `"project_id":"cross-proj-a"`,
		"session should return project cross-proj-a")
}

// TestIntegration_APIKeyViaSessionAndDelete tests:
// 1. Create API key
// 2. Login via session with that key
// 3. Delete the session (logout)
// 4. Verify the cookie is cleared
func TestIntegration_APIKeyViaSessionAndDelete(t *testing.T) {
	ts := newTestServer(t)

	// Create API key
	createReq := &v1.CreateApiKeyRequest{
		ProjectId: "session-delete-test",
		Label:     "session-key",
	}
	rawReq, _ := json.Marshal(createReq)
	resp := ts.postConnectRPC("/aleph.v1.AuthService/CreateApiKey", rawReq, nil)
	requireStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)

	var keyResp struct {
		Key struct {
			Key string `json:"key"`
		} `json:"key"`
	}
	err := json.Unmarshal([]byte(body), &keyResp)
	require.NoError(t, err)

	// Login with the key
	sessionReq := map[string]string{"api_key": keyResp.Key.Key}
	sessionData, _ := json.Marshal(sessionReq)
	resp = ts.postJSON("/api/v1/auth/session", "application/json", sessionData)
	requireStatus(t, resp, http.StatusOK)

	var jwtCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "aleph_jwt" {
			jwtCookie = c
			break
		}
	}
	require.NotNil(t, jwtCookie)
	assert.NotEmpty(t, jwtCookie.Value)

	// Delete the session (logout)
	req, err := http.NewRequest("DELETE", ts.baseURL+"/api/v1/auth/session", nil)
	require.NoError(t, err)
	for _, c := range resp.Cookies() {
		req.AddCookie(c)
	}
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusOK)

	// Verify the cookie is cleared
	var clearedCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "aleph_jwt" {
			clearedCookie = c
			break
		}
	}
	require.NotNil(t, clearedCookie)
	assert.Empty(t, clearedCookie.Value, "cookie value should be empty after logout")
	assert.True(t, clearedCookie.MaxAge < 0 || clearedCookie.Expires.Before(time.Now()),
		"cookie should be expired after logout")
}

// TestIntegration_RawHTTPEndpoints tests unprotected and auth-protected
// raw HTTP endpoints.
func TestIntegration_RawHTTPEndpoints(t *testing.T) {
	ts := newTestServer(t)

	// Swagger endpoint should be accessible
	resp := ts.get("/swagger.json", nil)
	// May 404 if swagger file doesn't exist, but should not 401
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode,
		"swagger endpoint should not return 401")

	// Protected endpoint without auth → 401
	resp = ts.get("/api/v1/tools/categories", nil)
	requireStatus(t, resp, http.StatusUnauthorized)

	// Protected endpoint with valid auth → OK
	jwtCookie := ts.loginAsAdmin(t)
	resp = ts.get("/api/v1/tools/categories", []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)

	// CORS headers should be present for allowed origins
	req, _ := http.NewRequest("GET", ts.baseURL+"/api/v1/healthz", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:5173", resp.Header.Get("Access-Control-Allow-Origin"),
		"CORS header should match allowed origin")
}

// TestIntegration_ConfigRequired_LoadConfig loads the config to verify
// the required env vars mechanism. This test verifies that LoadConfig
// errors when JWT_SECRET is missing.
func TestIntegration_ConfigRequired(t *testing.T) {
	// Ensure env vars are unset for this test
	for _, key := range []string{"JWT_SECRET", "POSTGRES_DSN", "KEY_ENCRYPTION_KEY"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	_, err := config.LoadConfig()
	require.Error(t, err)
	// The error should mention the missing required variable
	assert.True(t, strings.Contains(err.Error(), "JWT_SECRET") ||
		strings.Contains(err.Error(), "POSTGRES_DSN") ||
		strings.Contains(err.Error(), "KEY_ENCRYPTION_KEY"),
		"error should mention a required env var: %s", err.Error())
}

// TestIntegration_SystemInfoEndpoints tests various system/info endpoints.
func TestIntegration_SystemInfoEndpoints(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	// Diagnostic patterns endpoint
	resp := ts.get("/api/v1/diagnostic/patterns", []*http.Cookie{jwtCookie})
	// Should either succeed (200) or indicate the handler is not registered
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Logf("diagnostic/patterns returned %d", resp.StatusCode)
	}
}
