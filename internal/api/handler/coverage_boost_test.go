package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ff3300/aleph-v2/internal/auth"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── HandleCreateSession ─────────────────────────────────────────────────────

func TestSessionHandler_CreateSession_BootstrapUserPrefix(t *testing.T) {
	h := NewSessionHandler((*repository.MetadataRepository)(nil), testJWTSecret)

	body, _ := json.Marshal(createSessionRequest{APIKey: "user_bootstrapkey123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.HandleCreateSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp createSessionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "bootstrap", resp.ProjectID)

	cookies := rec.Result().Cookies()
	var jwtCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "aleph_jwt" {
			jwtCookie = c
			break
		}
	}
	require.NotNil(t, jwtCookie)
	assert.True(t, jwtCookie.HttpOnly)
	assert.Equal(t, 3600, jwtCookie.MaxAge)

	claims, err := auth.ValidateToken(jwtCookie.Value, testJWTSecret)
	require.NoError(t, err)
	assert.Equal(t, "bootstrap", claims.ProjectID)
}

func TestSessionHandler_CreateSession_BootstrapReadOnly(t *testing.T) {
	h := NewSessionHandler((*repository.MetadataRepository)(nil), testJWTSecret)

	body, _ := json.Marshal(createSessionRequest{APIKey: "ro_readonlykey1234"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.HandleCreateSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp createSessionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "bootstrap", resp.ProjectID)
}

func TestSessionHandler_CreateSession_EnvBootstrapKey(t *testing.T) {
	h := NewSessionHandler((*repository.MetadataRepository)(nil), testJWTSecret)
	t.Setenv("ALEPH_API_KEY_SECRET_BACKEND", "env-bootstrap-secret-key12")

	body, _ := json.Marshal(createSessionRequest{APIKey: "env-bootstrap-secret-key12"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.HandleCreateSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp createSessionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "bootstrap", resp.ProjectID)
}

func TestSessionHandler_CreateSession_ShortKeyPanics(t *testing.T) {
	h := NewSessionHandler((*repository.MetadataRepository)(nil), testJWTSecret)

	body, _ := json.Marshal(createSessionRequest{APIKey: "short"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	func() {
		defer func() { _ = recover() }()
		h.HandleCreateSession(rec, req)
	}()
}

// ── HandleDeleteSession ─────────────────────────────────────────────────────

func TestSessionHandler_DeleteSession_ValidJWT_WithStore(t *testing.T) {
	h := NewSessionHandler((*repository.MetadataRepository)(nil), testJWTSecret)
	store := middleware.NewTokenRevocationStore(10 * time.Minute)
	h.WithRevocationStore(store)

	token := makeValidJWT(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: "aleph_jwt", Value: token})
	rec := httptest.NewRecorder()

	h.HandleDeleteSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "aleph_jwt" {
			assert.Equal(t, "", c.Value)
			assert.Equal(t, -1, c.MaxAge)
		}
	}
}

func TestSessionHandler_DeleteSession_NoRevocationStore(t *testing.T) {
	h := NewSessionHandler((*repository.MetadataRepository)(nil), testJWTSecret)

	token := makeValidJWT(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: "aleph_jwt", Value: token})
	rec := httptest.NewRecorder()

	h.HandleDeleteSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ok")
}

// ── ServeHTTP ───────────────────────────────────────────────────────────────

func TestToolExecuteHandler_ServeHTTP_PUT(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tools/execute/cat/name", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestToolExecuteHandler_ServeHTTP_DELETE(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tools/execute/cat/name", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// ── HandleExecuteTool ───────────────────────────────────────────────────────

func TestToolExecuteHandler_ExecuteTool_MissingCategory(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute//some_name", nil)
	req.SetPathValue("category", "")
	req.SetPathValue("name", "some_name")
	rec := httptest.NewRecorder()
	h.HandleExecuteTool(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolExecuteHandler_ExecuteTool_MissingName(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute/finance/", nil)
	req.SetPathValue("category", "finance")
	req.SetPathValue("name", "")
	rec := httptest.NewRecorder()
	h.HandleExecuteTool(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolExecuteHandler_ExecuteTool_MissingBoth(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute//", nil)
	req.SetPathValue("category", "")
	req.SetPathValue("name", "")
	rec := httptest.NewRecorder()
	h.HandleExecuteTool(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToolExecuteHandler_ExecuteTool_EmptyBody(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute/finance/finance_sentiment_analysis", nil)
	req.SetPathValue("category", "finance")
	req.SetPathValue("name", "finance_sentiment_analysis")
	rec := httptest.NewRecorder()
	h.HandleExecuteTool(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── HandleListToolsByCategory ───────────────────────────────────────────────

func TestToolExecuteHandler_ListToolsByCategory_WrongMethod(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/execute/finance/list", nil)
	req.SetPathValue("category", "finance")
	rec := httptest.NewRecorder()
	h.HandleListToolsByCategory(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestToolExecuteHandler_ListToolsByCategory_Pagination(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/execute/finance?page=1&per_page=1", nil)
	req.SetPathValue("category", "finance")
	rec := httptest.NewRecorder()
	h.HandleListToolsByCategory(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var toolDefs []tools.ToolDefinition
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&toolDefs))
	assert.Len(t, toolDefs, 1)
}

func TestToolExecuteHandler_ListToolsByCategory_PageOverflow(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/execute/finance?page=999&per_page=2", nil)
	req.SetPathValue("category", "finance")
	rec := httptest.NewRecorder()
	h.HandleListToolsByCategory(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var toolDefs []tools.ToolDefinition
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&toolDefs))
	assert.Empty(t, toolDefs)
}

// ── HandleRegister ──────────────────────────────────────────────────────────

func TestToolExecuteHandler_HandleRegister_WrongMethodV2(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/register", nil)
	rec := httptest.NewRecorder()
	h.HandleRegister(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// ── Registry ────────────────────────────────────────────────────────────────

func TestToolExecuteHandler_Registry_LazyInitAndOverride(t *testing.T) {
	h := &ToolExecuteHandler{}
	assert.Nil(t, h.registry)

	reg := h.Registry()
	assert.NotNil(t, reg)

	reg2 := h.Registry()
	assert.Equal(t, reg, reg2)

	customReg := tools.NewToolRegistry()
	h.SetRegistry(customReg)
	assert.Equal(t, customReg, h.Registry())
}

// ── HandleListCategories ────────────────────────────────────────────────────

func TestToolExecuteHandler_ListCategories_MethodNotAllowedV2(t *testing.T) {
	h := setupToolExecHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/categories", nil)
	rec := httptest.NewRecorder()
	h.HandleListCategories(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// ── HandleCallTool ──────────────────────────────────────────────────────────

func TestToolExecuteHandler_HandleCallTool_SuccessV2(t *testing.T) {
	h := setupToolExecHandler(t)

	body := bytes.NewBufferString(`{"tool":"finance.finance_prophet_forecast","params":{"data":[1,2,3,4,5]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/call", body)
	rec := httptest.NewRecorder()
	h.HandleCallTool(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
