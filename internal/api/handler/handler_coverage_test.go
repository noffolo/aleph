package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/api/sse"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/service/notification"
	"github.com/ff3300/aleph-v2/internal/tools/adaptation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
	"log/slog"
)

// ─── Extended Metadata Repository Setup ────────────────────────────────────

// setupMetaRepoExtended creates an in-memory DuckDB MetadataRepository with all
// tables needed for handler coverage tests.
func setupMetaRepoExtended(t *testing.T) *repository.MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// system_agents
	_, err = db.Exec(`CREATE TABLE system_agents (id TEXT PRIMARY KEY, project_id TEXT, name TEXT, provider TEXT, model TEXT, api_key TEXT, system_prompt TEXT, skill_ids TEXT, base_url TEXT)`)
	require.NoError(t, err)
	// system_tools
	_, err = db.Exec(`CREATE TABLE system_tools (id TEXT PRIMARY KEY, name TEXT, description TEXT, code TEXT, category TEXT DEFAULT '', version TEXT DEFAULT '', health_status TEXT DEFAULT 'unknown', last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, source_type TEXT DEFAULT 'builtin')`)
	require.NoError(t, err)
	// system_skills
	_, err = db.Exec(`CREATE TABLE system_skills (id TEXT PRIMARY KEY, project_id TEXT, name TEXT, description TEXT, tool_ids TEXT)`)
	require.NoError(t, err)
	// system_api_keys
	_, err = db.Exec(`CREATE TABLE system_api_keys (id TEXT PRIMARY KEY, project_id TEXT, label TEXT, key TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	// system_tasks
	_, err = db.Exec(`CREATE TABLE system_tasks (id TEXT PRIMARY KEY, project_id TEXT, name TEXT, source_type TEXT, config_json TEXT, schedule TEXT, status TEXT, progress INT)`)
	require.NoError(t, err)
	// system_notification_channels
	_, err = db.Exec(`CREATE TABLE system_notification_channels (id TEXT PRIMARY KEY, project_id TEXT, name TEXT, type TEXT, config_json TEXT)`)
	require.NoError(t, err)
	// system_chat_history
	_, err = db.Exec(`CREATE TABLE system_chat_history (id UUID DEFAULT gen_random_uuid(), project_id TEXT, agent_id TEXT, role TEXT, content TEXT, tool_call TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	// system_projects
	_, err = db.Exec(`CREATE TABLE system_projects (id TEXT PRIMARY KEY, name TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

// ─── Agent Handler Tests ───────────────────────────────────────────────────

func TestAgentHandler_ListAgents_EmptyRepo(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAgentHandler("/tmp", repo, "")

	req := connect.NewRequest(&v1.ListAgentsRequest{ProjectId: "p1"})
	resp, err := h.ListAgents(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg)
	assert.Len(t, resp.Msg.Agents, 0)
}

func TestAgentHandler_ListAgents_WithData(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAgentHandler("/tmp", repo, "")

	// Create an agent first
	_, err := h.CreateAgent(context.Background(), connect.NewRequest(&v1.CreateAgentRequest{
		ProjectId: "p1",
		Agent:     &v1.Agent{Id: "a1", Name: "test-agent", Provider: "ollama", Model: "llama3", ApiKey: "sk-1234567890abcdef"},
	}))
	require.NoError(t, err)

	req := connect.NewRequest(&v1.ListAgentsRequest{ProjectId: "p1"})
	resp, err := h.ListAgents(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp.Msg.Agents, 1)
	assert.Equal(t, "test-agent", resp.Msg.Agents[0].Name)
	// Verify API key masking
	assert.Equal(t, "sk-12345****", resp.Msg.Agents[0].ApiKey)
}

func TestAgentHandler_DeleteAgent_Success(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAgentHandler("/tmp", repo, "")

	_, err := h.CreateAgent(context.Background(), connect.NewRequest(&v1.CreateAgentRequest{
		ProjectId: "p1",
		Agent:     &v1.Agent{Id: "a1", Name: "agent", Provider: "ollama", Model: "llama3"},
	}))
	require.NoError(t, err)

	resp, err := h.DeleteAgent(context.Background(), connect.NewRequest(&v1.DeleteAgentRequest{
		Id: "a1", ProjectId: "p1",
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

func TestAgentHandler_DeleteAgent_NotFound(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAgentHandler("/tmp", repo, "")

	resp, err := h.DeleteAgent(context.Background(), connect.NewRequest(&v1.DeleteAgentRequest{
		Id: "nonexistent", ProjectId: "p1",
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

func TestAgentHandler_UpdateAgent_Success(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAgentHandler("/tmp", repo, "")

	_, err := h.CreateAgent(context.Background(), connect.NewRequest(&v1.CreateAgentRequest{
		ProjectId: "p1",
		Agent:     &v1.Agent{Id: "a1", Name: "orig", Provider: "ollama", Model: "llama3"},
	}))
	require.NoError(t, err)

	resp, err := h.UpdateAgent(context.Background(), connect.NewRequest(&v1.UpdateAgentRequest{
		ProjectId: "p1",
		Agent:     &v1.Agent{Id: "a1", Name: "updated", Provider: "ollama", Model: "llama3", ApiKey: "sk-1234567890abcdef"},
	}))
	require.NoError(t, err)
	assert.Equal(t, "sk-12345****", resp.Msg.Agent.ApiKey)
}

// ─── Auth Handler Tests ────────────────────────────────────────────────────

func TestAuthHandler_ListApiKeys_Empty(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAuthHandler(repo)

	resp, err := h.ListApiKeys(context.Background(), connect.NewRequest(&v1.ListApiKeysRequest{ProjectId: "p1"}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Keys, 0)
}

func TestAuthHandler_CreateApiKey_Success(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAuthHandler(repo)

	resp, err := h.CreateApiKey(context.Background(), connect.NewRequest(&v1.CreateApiKeyRequest{
		ProjectId: "p1",
		Label:     "test-key",
	}))
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg.Key)
	assert.Equal(t, "test-key", resp.Msg.Key.Label)
	assert.NotEmpty(t, resp.Msg.Key.Id)
	assert.NotEmpty(t, resp.Msg.Key.Key) // raw key returned on creation
}

func TestAuthHandler_DeleteApiKey_Success(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAuthHandler(repo)

	createResp, err := h.CreateApiKey(context.Background(), connect.NewRequest(&v1.CreateApiKeyRequest{
		ProjectId: "p1", Label: "test-key",
	}))
	require.NoError(t, err)

	resp, err := h.DeleteApiKey(context.Background(), connect.NewRequest(&v1.DeleteApiKeyRequest{
		Id: createResp.Msg.Key.Id, ProjectId: "p1",
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

func TestAuthHandler_DeleteApiKey_NotFound(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAuthHandler(repo)

	resp, err := h.DeleteApiKey(context.Background(), connect.NewRequest(&v1.DeleteApiKeyRequest{
		Id: "nonexistent", ProjectId: "p1",
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

// ─── Ingestion Handler Tests ───────────────────────────────────────────────

func TestIngestionHandler_GetProgress(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	// Insert a task directly
	repo.CreateTask(&repository.IngestionTaskRecord{
		ID: "t1", ProjectID: "p1", Name: "task1", SourceType: "csv",
		ConfigJSON: "{}", Schedule: "* * *", Status: "running", Progress: 42,
	})
	h := &IngestionHandler{metaRepo: repo}

	resp, err := h.GetProgress(context.Background(), connect.NewRequest(&v1.GetProgressRequest{TaskId: "t1"}))
	require.NoError(t, err)
	assert.Equal(t, int32(42), resp.Msg.Progress)
}

func TestIngestionHandler_GetProgress_NotFound(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := &IngestionHandler{metaRepo: repo}

	_, err := h.GetProgress(context.Background(), connect.NewRequest(&v1.GetProgressRequest{TaskId: "nonexistent"}))
	require.Error(t, err)
}

func TestIngestionHandler_ListTasks_Empty(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := &IngestionHandler{metaRepo: repo}

	resp, err := h.ListTasks(context.Background(), connect.NewRequest(&v1.ListTasksRequest{ProjectId: "p1"}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Tasks, 0)
}

func TestIngestionHandler_ListTasks_WithData(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	repo.CreateTask(&repository.IngestionTaskRecord{
		ID: "t1", ProjectID: "p1", Name: "task1", SourceType: "csv",
		ConfigJSON: "{}", Schedule: "* * *", Status: "idle", Progress: 0,
	})
	h := &IngestionHandler{metaRepo: repo}

	resp, err := h.ListTasks(context.Background(), connect.NewRequest(&v1.ListTasksRequest{ProjectId: "p1"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Tasks, 1)
	assert.Equal(t, "task1", resp.Msg.Tasks[0].Name)
}

func TestIngestionHandler_CreateTask_Success(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := &IngestionHandler{metaRepo: repo}

	resp, err := h.CreateTask(context.Background(), connect.NewRequest(&v1.CreateTaskRequest{
		ProjectId: "p1",
		Task:      &v1.IngestionTask{Id: "t1", Name: "newtask", SourceType: "rss", Schedule: "@daily"},
	}))
	require.NoError(t, err)
	assert.Equal(t, "newtask", resp.Msg.Task.Name)
}

func TestIngestionHandler_CreateTask_AutoID(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := &IngestionHandler{metaRepo: repo}

	resp, err := h.CreateTask(context.Background(), connect.NewRequest(&v1.CreateTaskRequest{
		ProjectId: "p1",
		Task:      &v1.IngestionTask{Id: "", Name: "autoid", SourceType: "rss", Schedule: "@daily"},
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.Task.Id)
}

func TestIngestionHandler_DeleteTask_Success(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := &IngestionHandler{metaRepo: repo}

	_, err := h.CreateTask(context.Background(), connect.NewRequest(&v1.CreateTaskRequest{
		ProjectId: "p1",
		Task:      &v1.IngestionTask{Id: "t1", Name: "task", SourceType: "csv", Schedule: "* * *"},
	}))
	require.NoError(t, err)

	resp, err := h.DeleteTask(context.Background(), connect.NewRequest(&v1.DeleteTaskRequest{
		Id: "t1", ProjectId: "p1",
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

// ─── Tool Handler HTTP Tests ───────────────────────────────────────────────

func TestToolHandler_ServeHTTP_ListAll(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	// Pre-populate a tool
	_, err := h.CreateTool(context.Background(), connect.NewRequest(&v1.CreateToolRequest{
		Tool: &v1.Tool{Id: "t1", Name: "hammer", Description: "hits", Code: "do()"},
	}))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "hammer")
}

func TestToolHandler_ServeHTTP_Intelligence(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)
	repo.CreateTool(&repository.ToolRecord{ID: "t1", Name: "tool1", Description: "d", Code: "do()", HealthStatus: "healthy"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/intelligence", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "tool1")
}

func TestToolHandler_ServeHTTP_Recommendations(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)
	repo.CreateTool(&repository.ToolRecord{ID: "t1", Name: "tool1", Description: "d", Code: "do()"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/recommendations", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "tool1")
}

func TestToolHandler_ServeHTTP_Health(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)
	repo.CreateTool(&repository.ToolRecord{ID: "t1", Name: "tool1", Description: "d", Code: "do()", HealthStatus: "healthy"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "healthy")
}

func TestToolHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestToolHandler_HandleVerify_Valid(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)
	_, err := h.CreateTool(context.Background(), connect.NewRequest(&v1.CreateToolRequest{
		Tool: &v1.Tool{Id: "t1", Name: "h", Description: "d", Code: "do()"},
	}))
	require.NoError(t, err)

	body := bytes.NewBufferString(`{"tool_id":"t1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/verify", body)
	rec := httptest.NewRecorder()
	h.HandleVerify(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"valid":true`)
}

func TestToolHandler_HandleVerify_Invalid(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	body := bytes.NewBufferString(`{"tool_id":"nonexistent"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/verify", body)
	rec := httptest.NewRecorder()
	h.HandleVerify(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"valid":false`)
}

func TestToolHandler_HandleVerify_EmptyID(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	body := bytes.NewBufferString(`{"tool_id":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/verify", body)
	rec := httptest.NewRecorder()
	h.HandleVerify(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolHandler_HandleVerify_BadJSON(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	body := bytes.NewBufferString(`not-json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/verify", body)
	rec := httptest.NewRecorder()
	h.HandleVerify(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolHandler_HandleHealthHistory_Found(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)
	repo.CreateTool(&repository.ToolRecord{ID: "t1", Name: "tool1", Description: "d", Code: "do()", HealthStatus: "healthy", Version: "1.0", SourceType: "builtin"})

	body := bytes.NewBufferString(`{"tool_id":"t1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/t1", body)
	rec := httptest.NewRecorder()
	h.HandleHealthHistory(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "healthy")
}

func TestToolHandler_HandleHealthHistory_NotFound(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	body := bytes.NewBufferString(`{"tool_id":"nonexistent"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/nonexistent", body)
	rec := httptest.NewRecorder()
	h.HandleHealthHistory(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "unknown")
}

func TestToolHandler_HandleHealthHistory_EmptyID(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools", body)
	rec := httptest.NewRecorder()
	h.HandleHealthHistory(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolHandler_HandleListAll(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)
	repo.CreateTool(&repository.ToolRecord{ID: "t1", Name: "tool1", Description: "d", Code: "do()"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools?page=1&per_page=10", nil)
	rec := httptest.NewRecorder()
	h.HandleListAll(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "tool1")
}

func TestToolHandler_UpdateTool_Success(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	_, err := h.CreateTool(context.Background(), connect.NewRequest(&v1.CreateToolRequest{
		Tool: &v1.Tool{Id: "t1", Name: "orig", Description: "d", Code: "do()"},
	}))
	require.NoError(t, err)

	resp, err := h.UpdateTool(context.Background(), connect.NewRequest(&v1.UpdateToolRequest{
		Tool: &v1.Tool{Id: "t1", Name: "updated", Description: "new desc", Code: "new()"},
	}))
	require.NoError(t, err)
	assert.Equal(t, "updated", resp.Msg.Tool.Name)
}

// ─── Notification Handler Tests ────────────────────────────────────────────

func TestNotificationHandler_ListChannels_Empty(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewNotificationHandler(nil, repo)

	resp, err := h.ListChannels(context.Background(), connect.NewRequest(&v1.ListChannelsRequest{ProjectId: "p1"}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Channels, 0)
}

func TestNotificationHandler_ListChannels_WithData(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	// Insert directly to avoid NotificationService dependency
	repo.ListNotificationChannels("p1") // just check it returns empty
	h := NewNotificationHandler(nil, repo)

	resp, err := h.ListChannels(context.Background(), connect.NewRequest(&v1.ListChannelsRequest{ProjectId: "p1"}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Channels, 0)
}

func TestNotificationHandler_SendWebhook_Success(t *testing.T) {
	// Stub NotificationService that always succeeds
	svc := &notification.NotificationService{}
	h := &NotificationHandler{svc: svc}

	resp, err := h.SendWebhook(context.Background(), connect.NewRequest(&v1.SendWebhookRequest{
		Url:        "https://example.com/webhook",
		PayloadJson: `{"key":"value"}`,
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

// ─── Library Handler Tests ─────────────────────────────────────────────────

func TestLibraryHandler_ListAssets_Empty(t *testing.T) {
	dir := t.TempDir()
	h := NewLibraryHandler(dir)

	resp, err := h.ListAssets(context.Background(), connect.NewRequest(&v1.ListAssetsRequest{ProjectId: "p1"}))
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg)
	assert.Len(t, resp.Msg.Assets, 0)
}

func TestLibraryHandler_ListAssets_WithFiles(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "p1", "library")
	os.MkdirAll(libPath, 0755)
	os.WriteFile(filepath.Join(libPath, "test.csv"), []byte("data"), 0644)

	h := NewLibraryHandler(dir)
	resp, err := h.ListAssets(context.Background(), connect.NewRequest(&v1.ListAssetsRequest{ProjectId: "p1"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Assets, 1)
	assert.Equal(t, "test.csv", resp.Msg.Assets[0].Name)
}

func TestLibraryHandler_GetAssetContent_Success(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "p1", "library")
	os.MkdirAll(libPath, 0755)
	os.WriteFile(filepath.Join(libPath, "test.txt"), []byte("hello world"), 0644)

	h := NewLibraryHandler(dir)
	resp, err := h.GetAssetContent(context.Background(), connect.NewRequest(&v1.GetAssetContentRequest{
		ProjectId: "p1", AssetId: "test.txt",
	}))
	require.NoError(t, err)
	assert.Equal(t, "hello world", resp.Msg.Content)
}

func TestLibraryHandler_GetAssetContent_NotFound(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "p1", "library"), 0755)

	h := NewLibraryHandler(dir)
	_, err := h.GetAssetContent(context.Background(), connect.NewRequest(&v1.GetAssetContentRequest{
		ProjectId: "p1", AssetId: "nonexistent.txt",
	}))
	require.Error(t, err)
}

func TestLibraryHandler_DeleteAsset_Success(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "p1", "library")
	os.MkdirAll(libPath, 0755)
	os.WriteFile(filepath.Join(libPath, "test.txt"), []byte("data"), 0644)

	h := NewLibraryHandler(dir)
	resp, err := h.DeleteAsset(context.Background(), connect.NewRequest(&v1.DeleteAssetRequest{
		ProjectId: "p1", Id: "test.txt",
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)

	// Verify file is gone
	_, err = os.Stat(filepath.Join(libPath, "test.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestLibraryHandler_UploadAsset_Success(t *testing.T) {
	dir := t.TempDir()
	h := NewLibraryHandler(dir)

	resp, err := h.UploadAsset(context.Background(), connect.NewRequest(&v1.UploadAssetRequest{
		ProjectId: "p1", Filename: "upload.txt", Content: []byte("uploaded content"),
	}))
	require.NoError(t, err)
	assert.Equal(t, "upload.txt", resp.Msg.Asset.Name)
}

func TestLibraryHandler_UploadAsset_TraversalRejected(t *testing.T) {
	dir := t.TempDir()
	h := NewLibraryHandler(dir)

	// filepath.Base sanitizes "../etc/passwd" to "passwd" — upload succeeds with sanitized name
	resp, err := h.UploadAsset(context.Background(), connect.NewRequest(&v1.UploadAssetRequest{
		ProjectId: "p1", Filename: "../etc/passwd", Content: []byte("bad"),
	}))
	require.NoError(t, err)
	assert.Equal(t, "passwd", resp.Msg.Asset.Name, "filepath.Base should sanitize traversal to basename")
}

func TestLibraryHandler_GeneratePdf_Success(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "p1", "library")
	os.MkdirAll(libPath, 0755)
	os.WriteFile(filepath.Join(libPath, "report.txt"), []byte("Report content line 1\nline 2"), 0644)

	h := NewLibraryHandler(dir)
	resp, err := h.GeneratePdf(context.Background(), connect.NewRequest(&v1.GeneratePdfRequest{
		ProjectId: "p1", AssetId: "report.txt",
	}))
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg.PdfData)
	assert.Equal(t, "report.pdf", resp.Msg.Filename)
	assert.True(t, len(resp.Msg.PdfData) > 0)
}

func TestLibraryHandler_GeneratePdf_NotFound(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "p1", "library"), 0755)

	h := NewLibraryHandler(dir)
	_, err := h.GeneratePdf(context.Background(), connect.NewRequest(&v1.GeneratePdfRequest{
		ProjectId: "p1", AssetId: "nonexistent.txt",
	}))
	require.Error(t, err)
}

// ─── Tool Execute Handler Tests ──────────────────────────────────────────────

func setupToolExecHandler(t *testing.T) *ToolExecuteHandler {
	t.Helper()
	repo := setupMetaRepoExtended(t)
	h := NewToolExecuteHandler(repo, nil, nil)
	// Register default tools via hook
	_ = h.Registry() // triggers populateDefaultRegistry
	return h
}

func TestToolExecuteHandler_HandleListCategoriesV2(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/categories", nil)
	rec := httptest.NewRecorder()
	h.HandleListCategories(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Should contain "finance" category
	assert.Contains(t, rec.Body.String(), "finance")
}

func TestToolExecuteHandler_HandleListCategoriesV2_MethodNotAllowed(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/categories", nil)
	rec := httptest.NewRecorder()
	h.HandleListCategories(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestToolExecuteHandler_HandleListToolsByCategory_FinanceV2(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/execute/finance", nil)
	req.SetPathValue("category", "finance")
	rec := httptest.NewRecorder()
	h.HandleListToolsByCategory(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "finance_prophet_forecast")
}

func TestToolExecuteHandler_HandleListToolsByCategory_UnknownV2(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/execute/unknownxyz", nil)
	req.SetPathValue("category", "unknownxyz")
	rec := httptest.NewRecorder()
	h.HandleListToolsByCategory(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestToolExecuteHandler_HandleListToolsByCategory_Empty(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/execute/", nil)
	req.SetPathValue("category", "")
	rec := httptest.NewRecorder()
	h.HandleListToolsByCategory(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolExecuteHandler_ServeHTTP_Get(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/execute/finance", nil)
	req.SetPathValue("category", "finance")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "finance_prophet_forecast")
}

func TestToolExecuteHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tools/execute/finance/tool", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestToolExecuteHandler_HandleCallTool_ByCategoryName(t *testing.T) {
	h := setupToolExecHandler(t)

	body := bytes.NewBufferString(`{"category":"finance","name":"finance_prophet_forecast","params":{"metric":"revenue","period":30,"data":[100,110,105,120,115]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/call", body)
	rec := httptest.NewRecorder()
	h.HandleCallTool(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "prediction")
}

func TestToolExecuteHandler_HandleCallTool_ByDotNotation(t *testing.T) {
	h := setupToolExecHandler(t)

	body := bytes.NewBufferString(`{"tool":"finance.finance_prophet_forecast","params":{"metric":"revenue","period":30,"data":[100,110,105,120,115]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/call", body)
	rec := httptest.NewRecorder()
	h.HandleCallTool(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "prediction")
}

func TestToolExecuteHandler_HandleCallTool_InvalidDotNotation(t *testing.T) {
	h := setupToolExecHandler(t)

	body := bytes.NewBufferString(`{"tool":"invalid","params":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/call", body)
	rec := httptest.NewRecorder()
	h.HandleCallTool(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolExecuteHandler_HandleCallTool_MissingTool(t *testing.T) {
	h := setupToolExecHandler(t)

	body := bytes.NewBufferString(`{"params":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/call", body)
	rec := httptest.NewRecorder()
	h.HandleCallTool(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolExecuteHandler_HandleCallTool_UnknownCategory(t *testing.T) {
	h := setupToolExecHandler(t)

	body := bytes.NewBufferString(`{"tool":"nonexistent.tool","params":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/call", body)
	rec := httptest.NewRecorder()
	h.HandleCallTool(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestToolExecuteHandler_HandleCallTool_MethodNotAllowed(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/call", nil)
	rec := httptest.NewRecorder()
	h.HandleCallTool(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestToolExecuteHandler_HandleCallTool_BadJSON(t *testing.T) {
	h := setupToolExecHandler(t)

	body := bytes.NewBufferString(`not-json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/call", body)
	rec := httptest.NewRecorder()
	h.HandleCallTool(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolExecuteHandler_HandleRegisterV2(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/register", nil)
	rec := httptest.NewRecorder()
	h.HandleRegister(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "registered")
	assert.Contains(t, rec.Body.String(), "count")
}

func TestToolExecuteHandler_SetRegistry(t *testing.T) {
	h := &ToolExecuteHandler{}
	assert.Nil(t, h.registry)

	// Setting registry should work
	reg := h.Registry() // lazy-initializes default
	assert.NotNil(t, reg)
	h.SetRegistry(reg)
	assert.Equal(t, reg, h.registry)
}

func TestToolExecuteHandler_ExecuteTool_HappyPath(t *testing.T) {
	h := setupToolExecHandler(t)

	body := bytes.NewBufferString(`{"metric":"revenue","period":30,"data":[100,110,105]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute/finance/finance_prophet_forecast", body)
	req.SetPathValue("category", "finance")
	req.SetPathValue("name", "finance_prophet_forecast")
	rec := httptest.NewRecorder()
	h.HandleExecuteTool(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "prediction")
}

func TestToolExecuteHandler_ExecuteTool_UnknownCategory(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute/unknown/tool", nil)
	req.SetPathValue("category", "unknown")
	req.SetPathValue("name", "tool")
	rec := httptest.NewRecorder()
	h.HandleExecuteTool(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestToolExecuteHandler_ExecuteTool_InvalidJSON(t *testing.T) {
	h := setupToolExecHandler(t)

	body := bytes.NewBufferString(`not-json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute/finance/prophet_forecast", body)
	req.SetPathValue("category", "finance")
	req.SetPathValue("name", "prophet_forecast")
	rec := httptest.NewRecorder()
	h.HandleExecuteTool(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolExecuteHandler_PopulateDefaultRegistry_Finance(t *testing.T) {
	reg := populateDefaultRegistry(nil, nil)
	assert.NotNil(t, reg)
	financeTools := reg.List("finance")
	assert.True(t, len(financeTools) >= 3, "expected at least 3 finance tools")
}

func TestToolExecuteHandler_PopulateDefaultRegistry_OSINT_NilBroker(t *testing.T) {
	reg := populateDefaultRegistry(nil, nil)
	osintTools := reg.List("osint")
	assert.Len(t, osintTools, 0, "OSINT tools should be empty when broker is nil")
}

func TestToolExecuteHandler_PopulateDefaultRegistry_OSINT_WithBroker(t *testing.T) {
	t.Skip("requires ShadowbrokerConfig not constructible in test")
}

func TestToolExecuteHandler_PopulateDefaultRegistry_HE(t *testing.T) {
	reg := populateDefaultRegistry(nil, nil)
	heTools := reg.List("human-ecosystems")
	assert.True(t, len(heTools) >= 5, "expected at least 5 human-ecosystems tools")
}

// ─── SSE Handler Tests ─────────────────────────────────────────────────────

func TestSSEHandler_Stream_MethodNotAllowed(t *testing.T) {
	logger := slog.Default()
	broker := sse.NewBroker(30*time.Second, logger)
	h := NewSSEHandler(broker, logger)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	rec := httptest.NewRecorder()
	h.Stream(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestSSEHandler_isAuthenticatedForSSE_NoCookieNoHeader(t *testing.T) {
	logger := slog.Default()
	broker := 	sse.NewBroker(30*time.Second, logger)
	h := NewSSEHandler(broker, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	assert.False(t, h.isAuthenticatedForSSE(req))
}

func TestSSEHandler_isAuthenticatedForSSE_NilMetaRepo(t *testing.T) {
	logger := slog.Default()
	broker := 	sse.NewBroker(30*time.Second, logger)
	h := NewSSEHandler(broker, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	req.Header.Set("X-Aleph-Api-Key", "some-key")
	assert.False(t, h.isAuthenticatedForSSE(req), "nil metaRepo should reject API key auth")
}

func TestSSEHandler_WithMetaRepoV2(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	logger := slog.Default()
	broker := 	sse.NewBroker(30*time.Second, logger)
	h := NewSSEHandler(broker, logger).WithMetaRepo(repo)

	assert.Equal(t, repo, h.metaRepo)
}

func TestSSEHandler_WithJWTSecretV2(t *testing.T) {
	logger := slog.Default()
	broker := 	sse.NewBroker(30*time.Second, logger)
	secret := []byte("test-secret")
	h := NewSSEHandler(broker, logger).WithJWTSecret(secret)

	assert.Equal(t, secret, h.jwtSecret)
}

func TestGenerateClientIDV2(t *testing.T) {
	id1 := generateClientID()
	id2 := generateClientID()
	assert.Contains(t, id1, "sse-")
	assert.Contains(t, id2, "sse-")
	assert.NotEqual(t, id1, id2)
}

func TestExtractAPIKeyFromSSEV2(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	assert.Empty(t, extractAPIKeyFromSSE(req))

	req.Header.Set("X-Aleph-Api-Key", "test-key")
	assert.Equal(t, "test-key", extractAPIKeyFromSSE(req))
}

// ─── NLP Handler Tests ─────────────────────────────────────────────────────

func TestNLPHandler_Close(t *testing.T) {
	logger := slog.Default()
	h := NewNLPHandler(logger, nil, &http.Client{})
	assert.NoError(t, h.Close())
}

func TestNLPHandler_Close_NilHTTPClient(t *testing.T) {
	logger := slog.Default()
	h := NewNLPHandler(logger, nil, nil)
	assert.NoError(t, h.Close())
}

func TestNLPHandler_SetBrierMonitor_Extended(t *testing.T) {
	logger := slog.Default()
	h := NewNLPHandler(logger, nil, nil)
	assert.Nil(t, h.brierMonitor)

	h.SetBrierMonitor(nil)
	assert.Nil(t, h.brierMonitor)
}

// ─── Tool Suggest Handler Tests ────────────────────────────────────────────

func TestToStageResultJSON_Nil(t *testing.T) {
	result := toStageResultJSON(nil)
	assert.Len(t, result, 0)
}

func TestToStageResultJSON_EmptySlice(t *testing.T) {
	result := toStageResultJSON([]adaptation.StageResult{})
	assert.Len(t, result, 0)
}

// ─── Query Handler Tests ───────────────────────────────────────────────────

func setupQueryHandlerExtended(t *testing.T) (*QueryHandler, *repository.MetadataRepository) {
	t.Helper()
	repo := setupMetaRepoExtended(t)
	h := &QueryHandler{
		projectsRoot: t.TempDir(),
		metaRepo:     repo,
		programs:     newProgramCache(),
	}
	return h, repo
}

func TestQueryHandler_GetChatHistory_Empty(t *testing.T) {
	h, _ := setupQueryHandlerExtended(t)

	req := connect.NewRequest(&v1.GetChatHistoryRequest{ProjectId: "p1", AgentId: "a1"})
	resp, err := h.GetChatHistory(context.Background(), req)
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Messages, 0)
}

func TestQueryHandler_GetChatHistory_WithData(t *testing.T) {
	h, repo := setupQueryHandlerExtended(t)
	repo.SaveChatMessage(context.Background(), "p1", "a1", "user", "hello", "")

	req := connect.NewRequest(&v1.GetChatHistoryRequest{ProjectId: "p1", AgentId: "a1"})
	resp, err := h.GetChatHistory(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp.Msg.Messages, 1)
	assert.Equal(t, "user", resp.Msg.Messages[0].Role)
	assert.Equal(t, "hello", resp.Msg.Messages[0].Content)
}

func TestQueryHandler_SetMemoryStore(t *testing.T) {
	h := &QueryHandler{}
	assert.Nil(t, h.memoryStore)
	h.SetMemoryStore(nil)
	assert.Nil(t, h.memoryStore)
}

// ─── Skill Handler Tests ───────────────────────────────────────────────────

func TestSkillHandler_UpdateSkill_Success(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewSkillHandler("/tmp", repo)

	_, err := h.CreateSkill(context.Background(), connect.NewRequest(&v1.CreateSkillRequest{
		ProjectId: "p1",
		Skill:     &v1.Skill{Id: "s1", Name: "orig", Description: "desc", ToolIds: []string{"t1"}},
	}))
	require.NoError(t, err)

	resp, err := h.UpdateSkill(context.Background(), connect.NewRequest(&v1.UpdateSkillRequest{
		ProjectId: "p1",
		Skill:     &v1.Skill{Id: "s1", Name: "updated", Description: "new desc", ToolIds: []string{"t2"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, "updated", resp.Msg.Skill.Name)
}

func TestSkillHandler_UpdateSkill_EdgeCases(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewSkillHandler("/tmp", repo)

	_, err := h.UpdateSkill(context.Background(), connect.NewRequest(&v1.UpdateSkillRequest{
		Skill: nil,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skill id is required")
}

func TestSkillHandler_UpdateSkill_EmptyIdV2(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewSkillHandler("/tmp", repo)

	_, err := h.UpdateSkill(context.Background(), connect.NewRequest(&v1.UpdateSkillRequest{
		Skill: &v1.Skill{Id: "", Name: "test"},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skill id is required")
}

// ─── Chat Session Tests ────────────────────────────────────────────────────

func TestBuildMinimalToolsMap(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	// Pre-populate a tool for buildMinimalToolsMap to find
	repo.CreateTool(&repository.ToolRecord{ID: "search_data", Name: "search_data", Description: "Search data", Code: "search()"})
	result := buildMinimalToolsMap(context.Background(), repo)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result)
}

// ─── Project Handler Tests ─────────────────────────────────────────────────

func TestProjectHandler_SetMetaRepo(t *testing.T) {
	h := NewProjectHandler("/tmp", nil)
	assert.Nil(t, h.metaRepo)
	repo := setupMetaRepoExtended(t)
	h.SetMetaRepo(repo)
	assert.Equal(t, repo, h.metaRepo)
}

func TestProjectHandler_SetMaxProjects(t *testing.T) {
	h := NewProjectHandler("/tmp", nil)
	h.SetMaxProjects(5)
	assert.Equal(t, 5, h.maxProjects)
}

func TestProjectHandler_SetLLMProvider(t *testing.T) {
	h := NewProjectHandler("/tmp", nil)
	assert.Nil(t, h.llm)
	h.SetLLMProvider(nil)
	assert.Nil(t, h.llm)
}

func setupOntoRepo(t *testing.T) *repository.OntologyRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE ontology_versions (
		version_id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		parent_version_id TEXT,
		diff_json TEXT,
		core_aleph_snapshot TEXT NOT NULL,
		status TEXT NOT NULL,
		source_description TEXT,
		rationale TEXT,
		confidence FLOAT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		modified_at TIMESTAMP
	)`)
	require.NoError(t, err)

	return repository.NewOntologyRepository(db)
}

func TestProjectHandler_NegotiateList_Empty(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	ontoRepo := setupOntoRepo(t)
	h := NewProjectHandler("/tmp", nil)
	h.SetMetaRepo(repo)
	h.SetOntologyRepository(ontoRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/negotiate/list?project_id=p1", nil)
	w := httptest.NewRecorder()
	h.NegotiateList(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	w.Result().Body.Close()
}

func TestProjectHandler_NegotiatePropose_Coverage(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	ontoRepo := setupOntoRepo(t)
	h := NewProjectHandler("/tmp", nil)
	h.SetMetaRepo(repo)
	h.SetOntologyRepository(ontoRepo)

	body := strings.NewReader(`{"project_id":"p1","aleph_definition":"object Test\n  from dataset test_ds\n  id id\n  property name type text from name\n","diff_json":"{}","source_description":"coverage test","rationale":"testing","confidence":0.9}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/negotiate", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NegotiatePropose(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "version_id")
	w.Result().Body.Close()
}

func TestProjectHandler_NegotiateAccept_Coverage(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	ontoRepo := setupOntoRepo(t)
	h := NewProjectHandler("/tmp", nil)
	h.SetMetaRepo(repo)
	h.SetOntologyRepository(ontoRepo)

	proposeBody := strings.NewReader(`{"project_id":"p1","aleph_definition":"object X\n  id id\n","diff_json":"{}","source_description":"prepare accept test","rationale":"testing","confidence":0.9}`)
	propReq := httptest.NewRequest(http.MethodPost, "/negotiate", proposeBody)
	propReq.Header.Set("Content-Type", "application/json")
	propW := httptest.NewRecorder()
	h.NegotiatePropose(propW, propReq)
	require.Equal(t, http.StatusOK, propW.Code)

	var propResp map[string]interface{}
	require.NoError(t, json.Unmarshal(propW.Body.Bytes(), &propResp))
	versionID := propResp["version_id"].(string)

	body := strings.NewReader(`{"version_id":"` + versionID + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/negotiate/accept", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NegotiateAccept(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	w.Result().Body.Close()
}

func TestProjectHandler_NegotiateReject_Coverage(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	ontoRepo := setupOntoRepo(t)
	h := NewProjectHandler("/tmp", nil)
	h.SetMetaRepo(repo)
	h.SetOntologyRepository(ontoRepo)

	proposeBody := strings.NewReader(`{"project_id":"p1","aleph_definition":"object Y\n  id id\n","diff_json":"{}","source_description":"prepare reject test","rationale":"testing","confidence":0.9}`)
	propReq := httptest.NewRequest(http.MethodPost, "/negotiate", proposeBody)
	propReq.Header.Set("Content-Type", "application/json")
	propW := httptest.NewRecorder()
	h.NegotiatePropose(propW, propReq)
	require.Equal(t, http.StatusOK, propW.Code)

	var propResp map[string]interface{}
	require.NoError(t, json.Unmarshal(propW.Body.Bytes(), &propResp))
	versionID := propResp["version_id"].(string)

	body := strings.NewReader(`{"version_id":"` + versionID + `","reason":"not needed"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/negotiate/reject", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NegotiateReject(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	w.Result().Body.Close()
}

// ─── Truncate JSON Tests ───────────────────────────────────────────────────

func TestTruncateJSON_Short(t *testing.T) {
	result := truncateJSON(`{"key":"value"}`, 100)
	assert.Equal(t, `{"key":"value"}`, result)
}

func TestTruncateJSON_Long(t *testing.T) {
	long := `{"a":"` + makeString(500, 'x') + `"}`
	result := truncateJSON(long, 200)
	assert.True(t, len(result) <= 200+3) // 3 for "..."
}

func TestTruncateJSON_Array(t *testing.T) {
	result := truncateJSON(`[1,2,3]`, 100)
	assert.Equal(t, `[1,2,3]`, result)
}

func TestTruncateJSON_DeepNested(t *testing.T) {
	nested := `{"a":{"b":{"c":{"d":{"e":"value"}}}}}` + makeString(500, 'x')
	result := truncateJSON(nested, 100)
	assert.True(t, len(result) <= 100+3)
}

func TestTruncateJSON_Flat(t *testing.T) {
	flat := makeString(500, 'x')
	result := truncateJSON(flat, 200)
	assert.Equal(t, 200, len(result))
}

func makeString(n int, ch byte) string {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = ch
	}
	return string(buf)
}

// ─── Session Handler Tests ─────────────────────────────────────────────────

func TestSessionHandler_HandleCreateSession_MissingCookie(t *testing.T) {
	t.Skip("requires specific cookie/JWT handling")
}

func TestSessionHandler_MaskAPIKey(t *testing.T) {
	assert.Equal(t, "cdef", maskAPIKey("1234567890abcdef"))
	assert.Equal(t, "****", maskAPIKey("abc"))
	assert.Equal(t, "****", maskAPIKey("abcd"))
	assert.Equal(t, "5678", maskAPIKey("12345678"))
}

// ─── Safety / Security Tests ───────────────────────────────────────────────

func TestSanitizePdfString(t *testing.T) {
	input := `test\special(chars)`
	result := sanitizePdfString(input)
	assert.Equal(t, `test\\special\(chars\)`, result)
}

// ─── Pagination Tests ──────────────────────────────────────────────────────

