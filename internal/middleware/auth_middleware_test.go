package middleware

import (
	"context"
	"net/http"
	"os"
	"testing"
)

func TestNewAuthInterceptor(t *testing.T) {
	i := NewAuthInterceptor(nil)
	if i == nil {
		t.Error("expected non-nil interceptor")
	}
}

func TestSkipAuth(t *testing.T) {
	cases := []struct {
		procedure string
		expected  bool
	}{
		{"/aleph.v1.AuthService/ListApiKeys", true},
		{"/aleph.v1.AuthService/CreateApiKey", true},
		{"/aleph.v1.NotificationService/SendWebhook", false},
		{"/aleph.v1.QueryService/ExecuteQuery", false},
		{"/aleph.v1.ProjectService/ListProjects", false},
	}
	for _, tc := range cases {
		result := skipAuth(tc.procedure)
		if result != tc.expected {
			t.Errorf("skipAuth(%q) = %v, want %v", tc.procedure, result, tc.expected)
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