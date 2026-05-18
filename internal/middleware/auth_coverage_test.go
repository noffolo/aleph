package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ff3300/aleph-v2/internal/auth"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── ExtractAPIKey variants ───────────────────────────────────────────────────

func TestExtractAPIKeyFromHeader(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{"X-Aleph header", map[string]string{"X-Aleph-Api-Key": "key123"}, "key123"},
		{"Bearer auth", map[string]string{"Authorization": "Bearer key456"}, "key456"},
		{"no headers", map[string]string{}, ""},
		{"X-Aleph priority", map[string]string{"X-Aleph-Api-Key": "key1", "Authorization": "Bearer key2"}, "key1"},
		{"Bearer without prefix", map[string]string{"Authorization": "key789"}, "key789"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}
			got := ExtractAPIKeyFromHeader(r)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestExtractAPIKeyFromCookie(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		cookie   string
		expected string
	}{
		{"valid cookie", "aleph_session=session-token-abc", "session-token-abc"},
		{"no cookie", "", ""},
		{"wrong cookie name", "other_cookie=value", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.cookie != "" {
				// Parse name=value format and set as cookie header
				r.Header.Set("Cookie", tc.cookie)
			}
			got := ExtractAPIKeyFromCookie(r)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestExtractAPIKey(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{"X-Aleph", map[string]string{"X-Aleph-Api-Key": "key1"}, "key1"},
		{"Bearer", map[string]string{"Authorization": "Bearer key2"}, "key2"},
		{"empty", map[string]string{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			for k, v := range tc.headers {
				h.Set(k, v)
			}
			assert.Equal(t, tc.expected, ExtractAPIKey(h))
		})
	}
}

// ─── bootstrapRole ────────────────────────────────────────────────────────────

func TestBootstrapRole(t *testing.T) {
	origBackend := os.Getenv("ALEPH_API_KEY_SECRET_BACKEND")
	defer os.Setenv("ALEPH_API_KEY_SECRET_BACKEND", origBackend)

	cases := []struct {
		name     string
		apiKey   string
		backend  string
		expected Role
	}{
		{"exact backend match", "secret123", "secret123", RoleAdmin},
		{"backend mismatch", "other", "secret123", ""},
		{"user_ prefix", "user_abc", "", RoleUser},
		{"ro_ prefix", "ro_abc", "", RoleReadOnly},
		{"no match", "plainkey", "", ""},
		{"empty backend var", "user_abc", "secret123", RoleUser},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("ALEPH_API_KEY_SECRET_BACKEND", tc.backend)
			got := bootstrapRole(tc.apiKey)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// ─── ProjectIDFromContext / RoleFromContext ──────────────────────────────────

func TestProjectIDFromContext(t *testing.T) {
	t.Parallel()
	ctx := projectIDToContext(context.Background(), "proj-xyz", RoleUser)
	assert.Equal(t, "proj-xyz", ProjectIDFromContext(ctx))

	emptyCtx := context.Background()
	assert.Empty(t, ProjectIDFromContext(emptyCtx))

	// Wrong type in context
	wrongCtx := context.WithValue(context.Background(), authCtxProjectID, 123)
	assert.Empty(t, ProjectIDFromContext(wrongCtx))
}

// ─── RequireProjectRole ──────────────────────────────────────────────────────

func TestRequireProjectRole(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireProjectRole()(handler)

	cases := []struct {
		name          string
		authProjectID string
		urlProject    string
		wantStatus    int
	}{
		{"matching projects", "proj1", "proj1", http.StatusOK},
		{"no auth project - passes", "", "proj1", http.StatusOK},
		{"mismatched projects", "proj1", "proj2", http.StatusForbidden},
		{"no url project - passes", "proj1", "", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.authProjectID != "" {
				ctx = projectIDToContext(ctx, tc.authProjectID, RoleUser)
			}
			url := "/api/v1/data"
			if tc.urlProject != "" {
				url = "/api/v1/data?project_id=" + tc.urlProject
			}
			req := httptest.NewRequest(http.MethodGet, url, nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestRequireProjectRole_QueryParam(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireProjectRole()(handler)

	t.Run("project_id_from_query", func(t *testing.T) {
		ctx := projectIDToContext(context.Background(), "proj-a", RoleUser)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/data?project_id=proj-a", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("query_project_mismatch", func(t *testing.T) {
		ctx := projectIDToContext(context.Background(), "proj-a", RoleUser)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/data?project_id=proj-b", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// ─── RoleFromContext edge cases ──────────────────────────────────────────────

func TestRoleFromContext_WrongType(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), authCtxRole, 12345) // wrong type
	assert.Equal(t, RoleUser, RoleFromContext(ctx))                    // safe default
}

// ─── RequireRole edge cases ──────────────────────────────────────────────────

func TestRequireRole_NoAllowedRoles(t *testing.T) {
	t.Parallel()
	ctx := projectIDToContext(context.Background(), "proj1", RoleAdmin)
	err := RequireRole(ctx) // no roles listed
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestRequireRole_MultipleAllowed(t *testing.T) {
	t.Parallel()
	ctx := projectIDToContext(context.Background(), "proj1", RoleReadOnly)
	// ReadOnly should fail against Admin-only but pass against multi-role list
	assert.ErrorIs(t, RequireRole(ctx, RoleAdmin), ErrForbidden)
	assert.NoError(t, RequireRole(ctx, RoleAdmin, RoleUser, RoleReadOnly))
}

// ─── IsAdmin edge cases ──────────────────────────────────────────────────────

func TestIsAdmin_EdgeCases(t *testing.T) {
	t.Parallel()
	assert.False(t, IsAdmin(context.Background())) // empty context
	assert.False(t, IsAdmin(projectIDToContext(context.Background(), "p", RoleReadOnly)))
	assert.True(t, IsAdmin(projectIDToContext(context.Background(), "p", RoleAdmin)))
}

// ─── JWT from cookie ─────────────────────────────────────────────────────────

func TestJWTFromCookie(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		cookies  []string
		expected string
	}{
		{"aleph_jwt present", []string{"aleph_jwt=token123"}, "token123"},
		{"multiple cookies", []string{"session=abc; aleph_jwt=token456"}, "token456"},
		{"no jwt cookie", []string{"session=abc"}, ""},
		{"empty cookies", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			for _, c := range tc.cookies {
				h.Add("Cookie", c)
			}
			got := jwtFromCookie(h)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// ─── AuthMiddleware with HTTP handler ─────────────────────────────────────────

func TestAuthMiddleware_NoAuthReturns401(t *testing.T) {
	repo, err := repository.NewMetadataRepository(setupAuthDB(t))
	require.NoError(t, err)

	mw := AuthMiddleware(repo, []byte("secret"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_WithAPIKeyHeader(t *testing.T) {
	db, repo := setupMetaRepoForAuth(t)

	rawKey := "sk-test-api-key-1234567890"
	hashedKey, err := auth.HashAPIKey(rawKey)
	require.NoError(t, err)
	keyID := rawKey[:8]
	_, err = db.Exec("INSERT INTO system_api_keys (id, project_id, label, key, role) VALUES ($1, $2, $3, $4, $5)",
		keyID, "project-1", "test-key", hashedKey, "admin")
	require.NoError(t, err)

	mw := AuthMiddleware(repo, []byte("secret"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pid := ProjectIDFromContext(r.Context())
		role := RoleFromContext(r.Context())
		w.Header().Set("X-Auth-Project", pid)
		w.Header().Set("X-Auth-Role", string(role))
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("X-Aleph-Api-Key", rawKey)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "project-1", rec.Header().Get("X-Auth-Project"))
	assert.Equal(t, "admin", rec.Header().Get("X-Auth-Role"))
}

func TestAuthMiddleware_BootstrapKey(t *testing.T) {
	origBackend := os.Getenv("ALEPH_API_KEY_SECRET_BACKEND")
	defer os.Setenv("ALEPH_API_KEY_SECRET_BACKEND", origBackend)
	os.Setenv("ALEPH_API_KEY_SECRET_BACKEND", "bootstrap-secret")

	repo, err := repository.NewMetadataRepository(setupAuthDB(t))
	require.NoError(t, err)

	mw := AuthMiddleware(repo, []byte("secret"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := RoleFromContext(r.Context())
		w.Header().Set("X-Auth-Role", string(role))
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("X-Aleph-Api-Key", "bootstrap-secret")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "admin", rec.Header().Get("X-Auth-Role"))
}

// ─── authenticateHTTP ────────────────────────────────────────────────────────

func TestAuthenticateHTTP_HeaderValidation(t *testing.T) {
	db, repo := setupMetaRepoForAuth(t)

	rawKey := "sk-http-auth-test-key"
	hashedKey, err := auth.HashAPIKey(rawKey)
	require.NoError(t, err)
	keyID := rawKey[:8]
	_, err = db.Exec("INSERT INTO system_api_keys (id, project_id, label, key, role) VALUES ($1, $2, $3, $4, $5)",
		keyID, "http-proj", "http-key", hashedKey, "user")
	require.NoError(t, err)

	t.Run("valid_api_key_header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Aleph-Api-Key", rawKey)
		pid, role, err := authenticateHTTP(req, repo, nil)
		assert.NoError(t, err)
		assert.Equal(t, "http-proj", pid)
		assert.Equal(t, RoleUser, role)
	})

	t.Run("valid_api_key_cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Cookie", "aleph_session="+rawKey)
		pid, role, err := authenticateHTTP(req, repo, nil)
		assert.NoError(t, err)
		assert.Equal(t, "http-proj", pid)
		assert.Equal(t, RoleUser, role)
	})

	t.Run("no_credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		_, _, err := authenticateHTTP(req, repo, nil)
		assert.ErrorIs(t, err, ErrNoAPIKey)
	})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func setupAuthDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE system_api_keys (id TEXT PRIMARY KEY, project_id TEXT, label TEXT, key TEXT, role TEXT DEFAULT '', created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	return db
}
