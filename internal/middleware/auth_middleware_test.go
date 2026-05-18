package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ff3300/aleph-v2/internal/auth"
	"github.com/ff3300/aleph-v2/internal/repository"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthInterceptor(t *testing.T) {
	i := NewAuthInterceptor(nil, nil)
	if i == nil {
		t.Error("expected non-nil interceptor")
	}
}

func TestAuthSkipSetEmpty(t *testing.T) {
	if len(authSkipSet) != 0 {
		t.Errorf("authSkipSet should be empty after W0-4 hardening, got %d entries", len(authSkipSet))
		for proc := range authSkipSet {
			t.Errorf("  unexpected entry: %s", proc)
		}
	}
}

func TestExtractApiKey(t *testing.T) {
	cases := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{"X-Aleph-Api-Key header", map[string]string{"X-Aleph-Api-Key": "aleph_123"}, "aleph_123"},
		{"Authorization Bearer", map[string]string{"Authorization": "Bearer aleph_456"}, "aleph_456"},
		{"No headers", map[string]string{}, ""},
		{"X-Aleph takes priority", map[string]string{"X-Aleph-Api-Key": "key1", "Authorization": "Bearer key2"}, "key1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			for k, v := range tc.headers {
				h.Set(k, v)
			}
			result := extractApiKey(h)
			if result != tc.expected {
				t.Errorf("got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestRoleFromEnvImpl(t *testing.T) {
	origBackend := os.Getenv("ALEPH_API_KEY_SECRET_BACKEND")
	defer os.Setenv("ALEPH_API_KEY_SECRET_BACKEND", origBackend)

	cases := []struct {
		name     string
		apiKey   string
		backend  string
		expected Role
	}{
		{"backend key match", "secret123", "secret123", RoleAdmin},
		{"backend key mismatch", "otherkey", "secret123", RoleUser},
		{"user_ prefix", "user_abc123", "", RoleUser},
		{"ro_ prefix", "ro_abc123", "", RoleReadOnly},
		{"no prefix, no backend", "plainkey", "", RoleUser},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("ALEPH_API_KEY_SECRET_BACKEND", tc.backend)
			got := roleFromEnvImpl(tc.apiKey)
			if got != tc.expected {
				t.Errorf("roleFromEnvImpl(%q) = %v, want %v", tc.apiKey, got, tc.expected)
			}
		})
	}
}

func TestRoleFromContext(t *testing.T) {
	ctx := projectIDToContext(context.Background(), "proj1", RoleAdmin)
	if r := RoleFromContext(ctx); r != RoleAdmin {
		t.Errorf("RoleFromContext = %v, want %v", r, RoleAdmin)
	}

	emptyCtx := context.Background()
	if r := RoleFromContext(emptyCtx); r != RoleUser {
		t.Errorf("RoleFromContext(empty) = %v, want %v", r, RoleUser)
	}
}

func TestRequireRole(t *testing.T) {
	ctx := projectIDToContext(context.Background(), "proj1", RoleReadOnly)

	if err := RequireRole(ctx, RoleReadOnly); err != nil {
		t.Errorf("RequireRole with ReadOnly allowed: %v", err)
	}
	if err := RequireRole(ctx, RoleAdmin); err == nil {
		t.Error("RequireRole with Admin allowed for ReadOnly user: expected error")
	}
	if err := RequireRole(ctx, RoleAdmin, RoleReadOnly); err != nil {
		t.Errorf("RequireRole with [Admin, ReadOnly]: %v", err)
	}
}

func TestIsAdmin(t *testing.T) {
	ctxAdmin := projectIDToContext(context.Background(), "proj1", RoleAdmin)
	ctxUser := projectIDToContext(context.Background(), "proj1", RoleUser)

	if !IsAdmin(ctxAdmin) {
		t.Error("IsAdmin should return true for admin role")
	}
	if IsAdmin(ctxUser) {
		t.Error("IsAdmin should return false for user role")
	}
}

func TestRBACEnforcement(t *testing.T) {
	cases := []struct {
		procedure   string
		role        Role
		shouldAllow bool
	}{
		{"/aleph.v1.AuthService/CreateApiKey", RoleAdmin, true},
		{"/aleph.v1.AuthService/CreateApiKey", RoleUser, false},
		{"/aleph.v1.AuthService/ListApiKeys", RoleAdmin, true},
		{"/aleph.v1.AuthService/ListApiKeys", RoleReadOnly, false},
		{"/aleph.v1.AuthService/DeleteApiKey", RoleAdmin, true},
		{"/aleph.v1.AuthService/DeleteApiKey", RoleUser, false},
		{"/aleph.v1.ProjectService/CreateProject", RoleUser, true},
		{"/aleph.v1.ProjectService/CreateProject", RoleReadOnly, false},
		{"/aleph.v1.ProjectService/DeleteProject", RoleAdmin, true},
		{"/aleph.v1.ProjectService/DeleteProject", RoleUser, false},
		{"/aleph.v1.AgentService/CreateAgent", RoleUser, true},
		{"/aleph.v1.AgentService/CreateAgent", RoleReadOnly, false},
		{"/aleph.v1.AgentService/ListAgents", RoleReadOnly, true},
		{"/aleph.v1.QueryService/ExecuteQuery", RoleReadOnly, true},
		{"/aleph.v1.IngestionService/RunTask", RoleUser, true},
		{"/aleph.v1.IngestionService/RunTask", RoleReadOnly, false},
		{"/aleph.v1.IngestionService/ListTasks", RoleReadOnly, true},
		{"/aleph.registry.v1.RegistryService/RegisterComponent", RoleUser, true},
		{"/aleph.registry.v1.RegistryService/RegisterComponent", RoleReadOnly, false},
		{"/aleph.registry.v1.RegistryService/ListComponents", RoleReadOnly, true},
		{"/aleph.tool.v1.SandboxService/ExecuteTool", RoleUser, true},
		{"/aleph.tool.v1.SandboxService/ExecuteTool", RoleReadOnly, false},
		{"/aleph.nlp.v1.NLPService/AnalyzeSentiment", RoleUser, true},
		{"/aleph.nlp.v1.NLPService/AnalyzeSentiment", RoleReadOnly, false},
		{"/aleph.v1.LibraryService/DeleteAsset", RoleUser, true},
		{"/aleph.v1.LibraryService/ListAssets", RoleReadOnly, true},
		{"/aleph.v1.UnknownService/UnknownRPC", RoleUser, true},
	}

	for _, tc := range cases {
		t.Run(tc.procedure+"_"+string(tc.role), func(t *testing.T) {
			err := checkProcedureRBAC(tc.procedure, tc.role)
			if tc.shouldAllow && err != nil {
				t.Errorf("expected %s with role %s to be allowed, got: %v", tc.procedure, tc.role, err)
			}
			if !tc.shouldAllow && err == nil {
				t.Errorf("expected %s with role %s to be forbidden", tc.procedure, tc.role)
			}
		})
	}
}

func TestRoleRank(t *testing.T) {
	if roleRank(RoleAdmin) <= roleRank(RoleUser) {
		t.Error("admin should outrank user")
	}
	if roleRank(RoleUser) <= roleRank(RoleReadOnly) {
		t.Error("user should outrank readonly")
	}
	if roleRank(RoleReadOnly) <= roleRank("unknown") {
		t.Error("readonly should outrank unknown")
	}
}

func TestTokenRevocationStore(t *testing.T) {
	store := NewTokenRevocationStore(1 * time.Hour)
	defer store.Stop()

	if store.IsRevoked("token-1") {
		t.Error("unregistered token should not be revoked")
	}

	store.Revoke("token-1")
	if !store.IsRevoked("token-1") {
		t.Error("revoked token should be revoked")
	}

	if store.IsRevoked("token-2") {
		t.Error("different token should not be affected")
	}
}

func TestValidateScopes(t *testing.T) {
	cases := []struct {
		required   string
		tokenScope string
		expected   bool
	}{
		{"", "read,write", true},
		{"read", "read,write", true},
		{"read,write", "read,write", true},
		{"admin", "read,write", false},
		{"read", "", false},
		{"", "", true},
	}
	for _, tc := range cases {
		result := ValidateScopes(tc.required, tc.tokenScope)
		if result != tc.expected {
			t.Errorf("ValidateScopes(%q, %q) = %v, want %v", tc.required, tc.tokenScope, result, tc.expected)
		}
	}
}

func TestRequireRoleHTTP(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	adminOnly := RequireRoleHTTP(RoleAdmin)
	readAny := RequireRoleHTTP(RoleAdmin, RoleUser, RoleReadOnly)

	cases := []struct {
		name       string
		role       Role
		middleware func(http.Handler) http.Handler
		wantStatus int
	}{
		{"admin on admin-only", RoleAdmin, adminOnly, http.StatusOK},
		{"user on admin-only", RoleUser, adminOnly, http.StatusForbidden},
		{"readonly on admin-only", RoleReadOnly, adminOnly, http.StatusForbidden},
		{"admin on readAny", RoleAdmin, readAny, http.StatusOK},
		{"user on readAny", RoleUser, readAny, http.StatusOK},
		{"readonly on readAny", RoleReadOnly, readAny, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := projectIDToContext(context.Background(), "proj1", tc.role)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			tc.middleware(handler).ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

func setupMetaRepoForAuth(t *testing.T) (*sql.DB, *repository.MetadataRepository) {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE system_api_keys (id TEXT PRIMARY KEY, project_id TEXT, label TEXT, key TEXT, role TEXT DEFAULT '', created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return db, repo
}

func TestValidateAPIKey(t *testing.T) {
	db, repo := setupMetaRepoForAuth(t)

	rawKey := "sk-test-api-key-1234567890"
	hashedKey, err := auth.HashAPIKey(rawKey)
	require.NoError(t, err)

	keyID := rawKey[:8]

	_, err = db.Exec(
		"INSERT INTO system_api_keys (id, project_id, label, key, role) VALUES ($1, $2, $3, $4, $5)",
		keyID, "project-1", "test-key", hashedKey, "admin",
	)
	require.NoError(t, err)

	t.Run("valid_key_returns_project_and_role", func(t *testing.T) {
		projectID, role, err := ValidateAPIKey(repo, rawKey)
		assert.NoError(t, err)
		assert.Equal(t, "project-1", projectID)
		assert.Equal(t, RoleAdmin, role)
	})

	t.Run("key_too_short", func(t *testing.T) {
		_, _, err := ValidateAPIKey(repo, "short")
		assert.ErrorIs(t, err, ErrInvalidAPIKey)
	})

	t.Run("key_not_found_in_db", func(t *testing.T) {
		_, _, err := ValidateAPIKey(repo, "sk-unknown-key-not-in-db")
		assert.ErrorIs(t, err, ErrInvalidAPIKey)
	})

	t.Run("wrong_key_same_prefix", func(t *testing.T) {
		_, _, err := ValidateAPIKey(repo, "sk-test-wrong-key-value")
		assert.ErrorIs(t, err, ErrInvalidAPIKey)
	})
}
