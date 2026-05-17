package routes

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ff3300/aleph-v2/internal/api/handler"
	"github.com/stretchr/testify/assert"
)

//go:embed dist/*
var testFrontend embed.FS

func TestRegisterRoutes_WithHandlerStructs(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, RegisterConfig{
		QueryHandler:        &handler.QueryHandler{},
		ProjectHandler:      &handler.ProjectHandler{},
		AgentHandler:        &handler.AgentHandler{},
		SkillHandler:        &handler.SkillHandler{},
		LibraryHandler:      &handler.LibraryHandler{},
		ToolHandler:         &handler.ToolHandler{},
		NLPHandler:          &handler.NLPHandler{},
		NotificationHandler: &handler.NotificationHandler{},
		AuthHandler:         &handler.AuthHandler{},
		IngestionHandler:    &handler.IngestionHandler{},
		SandboxHandler:      &handler.SandboxServiceHandler{},
		RegistryHandler:     &handler.RegistryServiceHandler{},
		ToolExecHandler:     &handler.ToolExecuteHandler{},
		CodeFlowHandler:     &handler.CodeFlowHandler{},
		SessionHandler:      &handler.SessionHandler{},
		Frontend:            testFrontend,
		JWTSecret:           []byte("test"),
	})

	for _, path := range []string{"/readyz", "/livez", "/api/v1/healthz", "/metrics"} {
		req := httptest.NewRequest("GET", path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "path=%s", path)
	}
}

func TestRegisterRoutes_SPAFallback(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, RegisterConfig{
		QueryHandler:        &handler.QueryHandler{},
		ProjectHandler:      &handler.ProjectHandler{},
		AgentHandler:        &handler.AgentHandler{},
		SkillHandler:        &handler.SkillHandler{},
		LibraryHandler:      &handler.LibraryHandler{},
		ToolHandler:         &handler.ToolHandler{},
		NLPHandler:          &handler.NLPHandler{},
		NotificationHandler: &handler.NotificationHandler{},
		AuthHandler:         &handler.AuthHandler{},
		IngestionHandler:    &handler.IngestionHandler{},
		SandboxHandler:      &handler.SandboxServiceHandler{},
		RegistryHandler:     &handler.RegistryServiceHandler{},
		ToolExecHandler:     &handler.ToolExecuteHandler{},
		CodeFlowHandler:     &handler.CodeFlowHandler{},
		SessionHandler:      &handler.SessionHandler{},
		Frontend:            testFrontend,
		JWTSecret:           []byte("test"),
	})

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "test")
}

func TestRegisterRoutes_SwaggerEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, RegisterConfig{
		QueryHandler:        &handler.QueryHandler{},
		ProjectHandler:      &handler.ProjectHandler{},
		AgentHandler:        &handler.AgentHandler{},
		SkillHandler:        &handler.SkillHandler{},
		LibraryHandler:      &handler.LibraryHandler{},
		ToolHandler:         &handler.ToolHandler{},
		NLPHandler:          &handler.NLPHandler{},
		NotificationHandler: &handler.NotificationHandler{},
		AuthHandler:         &handler.AuthHandler{},
		IngestionHandler:    &handler.IngestionHandler{},
		SandboxHandler:      &handler.SandboxServiceHandler{},
		RegistryHandler:     &handler.RegistryServiceHandler{},
		ToolExecHandler:     &handler.ToolExecuteHandler{},
		CodeFlowHandler:     &handler.CodeFlowHandler{},
		SessionHandler:      &handler.SessionHandler{},
		Frontend:            testFrontend,
		JWTSecret:           []byte("test"),
	})

	req := httptest.NewRequest("GET", "/swagger.json", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.NotEqual(t, 0, rr.Code)
}

func TestRegisterRoutes_NestedSPAFallback(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, RegisterConfig{
		QueryHandler:        &handler.QueryHandler{},
		ProjectHandler:      &handler.ProjectHandler{},
		AgentHandler:        &handler.AgentHandler{},
		SkillHandler:        &handler.SkillHandler{},
		LibraryHandler:      &handler.LibraryHandler{},
		ToolHandler:         &handler.ToolHandler{},
		NLPHandler:          &handler.NLPHandler{},
		NotificationHandler: &handler.NotificationHandler{},
		AuthHandler:         &handler.AuthHandler{},
		IngestionHandler:    &handler.IngestionHandler{},
		SandboxHandler:      &handler.SandboxServiceHandler{},
		RegistryHandler:     &handler.RegistryServiceHandler{},
		ToolExecHandler:     &handler.ToolExecuteHandler{},
		CodeFlowHandler:     &handler.CodeFlowHandler{},
		SessionHandler:      &handler.SessionHandler{},
		Frontend:            testFrontend,
		JWTSecret:           []byte("test"),
	})

	req := httptest.NewRequest("GET", "/app/settings", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "test")
}

func TestRegisterRoutes_ReadyzJSONContent(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, RegisterConfig{
		QueryHandler:        &handler.QueryHandler{},
		ProjectHandler:      &handler.ProjectHandler{},
		AgentHandler:        &handler.AgentHandler{},
		SkillHandler:        &handler.SkillHandler{},
		LibraryHandler:      &handler.LibraryHandler{},
		ToolHandler:         &handler.ToolHandler{},
		NLPHandler:          &handler.NLPHandler{},
		NotificationHandler: &handler.NotificationHandler{},
		AuthHandler:         &handler.AuthHandler{},
		IngestionHandler:    &handler.IngestionHandler{},
		SandboxHandler:      &handler.SandboxServiceHandler{},
		RegistryHandler:     &handler.RegistryServiceHandler{},
		ToolExecHandler:     &handler.ToolExecuteHandler{},
		CodeFlowHandler:     &handler.CodeFlowHandler{},
		SessionHandler:      &handler.SessionHandler{},
		Frontend:            testFrontend,
		JWTSecret:           []byte("test"),
	})

	SetDraining(false)
	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rr.Body.String())
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestRegisterRoutes_AllProbeEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, RegisterConfig{
		QueryHandler:        &handler.QueryHandler{},
		ProjectHandler:      &handler.ProjectHandler{},
		AgentHandler:        &handler.AgentHandler{},
		SkillHandler:        &handler.SkillHandler{},
		LibraryHandler:      &handler.LibraryHandler{},
		ToolHandler:         &handler.ToolHandler{},
		NLPHandler:          &handler.NLPHandler{},
		NotificationHandler: &handler.NotificationHandler{},
		AuthHandler:         &handler.AuthHandler{},
		IngestionHandler:    &handler.IngestionHandler{},
		SandboxHandler:      &handler.SandboxServiceHandler{},
		RegistryHandler:     &handler.RegistryServiceHandler{},
		ToolExecHandler:     &handler.ToolExecuteHandler{},
		CodeFlowHandler:     &handler.CodeFlowHandler{},
		SessionHandler:      &handler.SessionHandler{},
		Frontend:            testFrontend,
		JWTSecret:           []byte("test"),
	})

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/livez", http.StatusOK},
		{"GET", "/api/v1/healthz", http.StatusOK},
		{"GET", "/metrics", http.StatusOK},
		{"GET", "/swagger.json", 0}, // may be 404 if file missing
	}
	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if tc.want != 0 {
			assert.Equal(t, tc.want, rr.Code, "path=%s", tc.path)
		} else {
			assert.NotZero(t, rr.Code, "path=%s", tc.path)
		}
	}
}

func TestRegisterRoutes_DrainingReadyz(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, RegisterConfig{
		QueryHandler:        &handler.QueryHandler{},
		ProjectHandler:      &handler.ProjectHandler{},
		AgentHandler:        &handler.AgentHandler{},
		SkillHandler:        &handler.SkillHandler{},
		LibraryHandler:      &handler.LibraryHandler{},
		ToolHandler:         &handler.ToolHandler{},
		NLPHandler:          &handler.NLPHandler{},
		NotificationHandler: &handler.NotificationHandler{},
		AuthHandler:         &handler.AuthHandler{},
		IngestionHandler:    &handler.IngestionHandler{},
		SandboxHandler:      &handler.SandboxServiceHandler{},
		RegistryHandler:     &handler.RegistryServiceHandler{},
		ToolExecHandler:     &handler.ToolExecuteHandler{},
		CodeFlowHandler:     &handler.CodeFlowHandler{},
		SessionHandler:      &handler.SessionHandler{},
		Frontend:            testFrontend,
		JWTSecret:           []byte("test"),
	})

	SetDraining(true)
	defer SetDraining(false)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "draining")
}
