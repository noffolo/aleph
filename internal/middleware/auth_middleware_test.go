package middleware

import (
	"net/http"
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
