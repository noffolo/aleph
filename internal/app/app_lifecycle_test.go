package app

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/config"
	"github.com/ff3300/aleph-v2/internal/routes"
)

// ── test helpers ───────────────────────────────────────────────────────────

// freePort finds an available TCP port and returns it.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// postgresDSN returns the test Postgres DSN. No skipping — tests rely on it.
func postgresDSN() string {
	return "postgres://postgres:postgres@localhost:5432/aleph_test?sslmode=disable"
}

// newLifecycleConfig builds a Config suitable for lifecycle tests with a
// temp DuckDB file and a Postgres test database.
func newLifecycleConfig(t *testing.T, duckDBPath string) *config.Config {
	t.Helper()

	return &config.Config{
		Port:                 0,
		DataRoot:             filepath.Dir(duckDBPath),
		DuckDBPath:           duckDBPath,
		DuckDBSchema:         "main",
		PostgresDSN:          postgresDSN(),
		NLPAddr:              "http://localhost:8001",
		OllamaBaseURL:        "http://localhost:11434",
		OllamaPort:           "11434",
		JWTSecret:            []byte("test-jwt-secret-that-is-at-least-32-bytes-long!"),
		KeyEncryptionKey:     "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		EncryptionKey:        make([]byte, 32),
		BackupInterval:       "24h",
		BackupKeep:           7,
		RateLimitChat:        10,
		RateLimitHealth:      100,
		RateLimitDefault:     500,
		MaxProjects:          50,
		MaxAgentsPerProject:  20,
		CORSAllowedOrigins:   []string{"http://localhost:5173"},
		LLMTimeoutSeconds:    30,
		DevMode:              true,
	}
}

// waitForServer polls the base URL until the server responds 200 or times out.
func waitForServer(t *testing.T, baseURL string) {
	t.Helper()
	require.Eventually(t, func() bool {
		resp, err := http.Get(baseURL + "/livez")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "server did not become ready in time")
}

// ── TestNewAlephApp ────────────────────────────────────────────────────────

func TestNewAlephApp_WithSlowQueryThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	duckDBPath := filepath.Join(tmpDir, "test.duckdb")
	cfg := newLifecycleConfig(t, duckDBPath)
	cfg.SlowQueryThresholdMs = 1000

	app, err := NewAlephApp(cfg, embed.FS{})
	require.NoError(t, err)
	require.NotNil(t, app)

	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Logf("Close returned: %v", err)
		}
	})

	assert.NotNil(t, app.db)
}

func TestNewAlephApp_LogLevels(t *testing.T) {
	tmpDir := t.TempDir()

	logLevels := []string{"debug", "warn", "error", "info"}
	for _, level := range logLevels {
		t.Run("log_level_"+level, func(t *testing.T) {
			duckDBPath := filepath.Join(tmpDir, "test-"+level+".duckdb")
			cfg := newLifecycleConfig(t, duckDBPath)
			cfg.LogLevel = level

			app, err := NewAlephApp(cfg, embed.FS{})
			require.NoError(t, err)
			require.NotNil(t, app)

			t.Cleanup(func() {
				if cerr := app.Close(context.Background()); cerr != nil {
					t.Logf("Close returned: %v", cerr)
				}
			})
		})
	}
}

func TestNewAlephApp_WithSentryInit(t *testing.T) {
	tmpDir := t.TempDir()
	duckDBPath := filepath.Join(tmpDir, "test.duckdb")
	cfg := newLifecycleConfig(t, duckDBPath)

	t.Setenv("SENTRY_DSN", "https://fake@sentry.io/1")

	app, err := NewAlephApp(cfg, embed.FS{})
	require.NoError(t, err)
	require.NotNil(t, app)

	t.Cleanup(func() {
		if cerr := app.Close(context.Background()); cerr != nil {
			t.Logf("Close returned: %v", cerr)
		}
	})
}

func TestNewAlephApp(t *testing.T) {
	tmpDir := t.TempDir()
	duckDBPath := filepath.Join(tmpDir, "test.duckdb")
	cfg := newLifecycleConfig(t, duckDBPath)

	app, err := NewAlephApp(cfg, embed.FS{})
	require.NoError(t, err)
	require.NotNil(t, app)

	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Logf("Close returned: %v", err)
		}
	})

	t.Run("core_subsystems_initialized", func(t *testing.T) {
		assert.NotNil(t, app.db, "DuckDB should be initialized")
		assert.NotNil(t, app.pg, "Postgres should be initialized")
		assert.NotNil(t, app.metaRepo, "MetadataRepository should be initialized")
		assert.NotNil(t, app.eng, "Ingestion Engine should be initialized")
		assert.NotNil(t, app.logger, "logger should be initialized")
		assert.NotNil(t, app.ctx, "context should be initialized")
		assert.NotNil(t, app.cancel, "cancel func should be set")
		assert.NotNil(t, app.cfg, "config should be stored")

		// Verify DuckDB is functional
		var val int
		err := app.db.DB().QueryRow("SELECT 1").Scan(&val)
		assert.NoError(t, err, "DuckDB should respond")
		assert.Equal(t, 1, val)

		// Verify Postgres is functional
		err = app.pg.DB().QueryRow("SELECT 1").Scan(&val)
		assert.NoError(t, err, "Postgres should respond")
		assert.Equal(t, 1, val)
	})

	t.Run("nlp_subsystems_initialized", func(t *testing.T) {
		assert.NotNil(t, app.nlpHandler, "NLPHandler should be initialized")
		assert.NotNil(t, app.brierMonitor, "BrierMonitor should be initialized")
	})

	t.Run("usage_tracker_initialized", func(t *testing.T) {
		assert.NotNil(t, app.usageTracker, "UsageTracker should be initialized")
	})

	t.Run("server_is_nil_before_Serve", func(t *testing.T) {
		assert.Nil(t, app.server, "server should be nil before Serve() is called")
	})
}

// ── TestAlephApp_Serve_GracefulShutdown ────────────────────────────────────

func TestAlephApp_Serve_GracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	duckDBPath := filepath.Join(tmpDir, "serve.duckdb")
	port := freePort(t)
	cfg := newLifecycleConfig(t, duckDBPath)

	app, err := NewAlephApp(cfg, embed.FS{})
	require.NoError(t, err)
	require.NotNil(t, app)

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Serve(port)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	waitForServer(t, baseURL)

	t.Run("livez_returns_200_with_alive_status", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/livez")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"status":"alive"`)
	})

	t.Run("healthz_returns_200_with_json_ok", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v1/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, _ := io.ReadAll(resp.Body)
		var result map[string]string
		err = json.Unmarshal(body, &result)
		require.NoError(t, err, "response should be valid JSON")
		assert.Equal(t, "ok", result["status"])
	})

	t.Run("readyz_returns_200", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/readyz")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"status":"ok"`)
	})

	t.Run("server_initialized_after_Serve", func(t *testing.T) {
		assert.NotNil(t, app.server, "server field should be set after Serve()")
	})

	t.Run("graceful_shutdown_stops_accepting_connections", func(t *testing.T) {
		closeErr := app.Close(context.Background())
		// Close may return errors from shutting down goroutines; that's OK.
		if closeErr != nil {
			t.Logf("Close returned: %v", closeErr)
		}

		// After Close(), the server should reject new connections.
		require.Eventually(t, func() bool {
			_, err := http.Get(baseURL + "/livez")
			return err != nil
		}, 3*time.Second, 100*time.Millisecond, "server should stop accepting connections after Close")

		// Verify Serve() returned with http.ErrServerClosed.
		select {
		case serveErr := <-errCh:
			assert.ErrorIs(t, serveErr, http.ErrServerClosed,
				"Serve should return http.ErrServerClosed after graceful shutdown")
		case <-time.After(3 * time.Second):
			t.Fatal("Serve did not return within 3s of Close")
		}
	})
}

// ── TestAlephApp_HealthProbes ──────────────────────────────────────────────

func TestAlephApp_HealthProbes(t *testing.T) {
	callCount := &atomic.Int32{}
	var (
		mockErr      error
		mockErrMutex sync.Mutex
	)

	mockChecker := func(ctx context.Context) error {
		callCount.Add(1)
		mockErrMutex.Lock()
		defer mockErrMutex.Unlock()
		return mockErr
	}

	setMockErr := func(err error) {
		mockErrMutex.Lock()
		mockErr = err
		callCount.Store(0)
		mockErrMutex.Unlock()
	}

	// Minimal HTTP handler that mirrors the health probe logic from
	// routes.RegisterRoutes, keeping the test focused without needing
	// all the Connect RPC handler dependencies.
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if mockChecker != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := mockChecker(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprintf(w, `{"status":"unhealthy","reason":"%s"}`, err.Error())
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	})
	mux.HandleFunc("/api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if mockChecker != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := mockChecker(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprintf(w, `{"status":"unhealthy","reason":"%s"}`, err.Error())
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Run("livez_healthy_when_checker_returns_nil", func(t *testing.T) {
		setMockErr(nil)

		resp, err := http.Get(srv.URL + "/livez")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"status":"alive"`)
		assert.Equal(t, int32(1), callCount.Load(), "health check func should be called exactly once")
	})

	t.Run("livez_unhealthy_when_checker_returns_error", func(t *testing.T) {
		setMockErr(errors.New("db connection failed"))

		resp, err := http.Get(srv.URL + "/livez")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"status":"unhealthy"`)
		assert.Contains(t, string(body), "db connection failed")
		assert.Equal(t, int32(1), callCount.Load())
	})

	t.Run("healthz_healthy_when_checker_returns_nil", func(t *testing.T) {
		setMockErr(nil)

		resp, err := http.Get(srv.URL + "/api/v1/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"status":"ok"`)
		assert.Equal(t, int32(1), callCount.Load())
	})

	t.Run("healthz_unhealthy_when_checker_returns_error", func(t *testing.T) {
		setMockErr(errors.New("database timeout"))

		resp, err := http.Get(srv.URL + "/api/v1/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "database timeout")
		assert.Equal(t, int32(1), callCount.Load())
	})

	t.Run("health_check_is_not_called_for_non_health_routes", func(t *testing.T) {
		setMockErr(nil)

		// Register a non-health handler to verify checker is isolated.
		mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		resp, err := http.Get(srv.URL + "/test")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, int32(0), callCount.Load(),
			"health checker should not be called for non-health routes")
	})
}

// ── TestRegisterConfig_WithDefaults ────────────────────────────────────────

func TestRegisterConfig_WithDefaults(t *testing.T) {
	t.Run("zero_value_has_nil_pointers", func(t *testing.T) {
		cfg := routes.RegisterConfig{}
		assert.Nil(t, cfg.MetaRepo)
		assert.Nil(t, cfg.JWTSecret)
		assert.Nil(t, cfg.SSEBroker)
		assert.Nil(t, cfg.SSEHandler)
		assert.Nil(t, cfg.DiagnosticMonitor)
		assert.Nil(t, cfg.CodeFlow)
		assert.Nil(t, cfg.QueryHandler)
		assert.Nil(t, cfg.ProjectHandler)
		assert.Nil(t, cfg.AgentHandler)
		assert.Nil(t, cfg.SkillHandler)
		assert.Nil(t, cfg.LibraryHandler)
		assert.Nil(t, cfg.ToolHandler)
		assert.Nil(t, cfg.NLPHandler)
		assert.Nil(t, cfg.NotificationHandler)
		assert.Nil(t, cfg.AuthHandler)
		assert.Nil(t, cfg.IngestionHandler)
		assert.Nil(t, cfg.SandboxHandler)
		assert.Nil(t, cfg.RegistryHandler)
		assert.Nil(t, cfg.ToolExecHandler)
		assert.Nil(t, cfg.CodeFlowHandler)
		assert.Nil(t, cfg.SuggestPipeline)
		assert.Nil(t, cfg.SessionHandler)
		assert.Nil(t, cfg.AuthRateLimiter)
		assert.Nil(t, cfg.Interceptors)
		assert.Nil(t, cfg.HealthCheckFunc)
		// embed.FS has a non-nil zero value (contains nil pointer fields internally).
		assert.NotNil(t, cfg.Frontend)
	})

	t.Run("fields_are_settable", func(t *testing.T) {
		hcFunc := func(ctx context.Context) error { return nil }

		cfg := routes.RegisterConfig{
			JWTSecret:       []byte("test-secret"),
			HealthCheckFunc: hcFunc,
			Interceptors:    nil,
		}

		assert.Equal(t, []byte("test-secret"), cfg.JWTSecret)
		assert.NotNil(t, cfg.HealthCheckFunc)

		err := cfg.HealthCheckFunc(context.Background())
		assert.NoError(t, err)
	})

	t.Run("health_check_func_errored_returns_error", func(t *testing.T) {
		wantErr := errors.New("db unavailable")
		cfg := routes.RegisterConfig{
			HealthCheckFunc: func(ctx context.Context) error {
				return wantErr
			},
		}
		err := cfg.HealthCheckFunc(context.Background())
		assert.ErrorIs(t, err, wantErr)
	})
}

// ── TestSetupDemoData_FullPath ─────────────────────────────────────────────

func TestSetupDemoData_FullPath(t *testing.T) {
	tmpDir := t.TempDir()
	duckDBPath := filepath.Join(tmpDir, "demo.duckdb")
	cfg := newLifecycleConfig(t, duckDBPath)

	app, err := NewAlephApp(cfg, embed.FS{})
	require.NoError(t, err)
	require.NotNil(t, app)

	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Logf("Close returned: %v", err)
		}
	})

	projectsRoot := filepath.Join(tmpDir, "projects")

	// Clean up any stale data from prior test runs that share this Postgres DB.
	_, err = app.pg.DB().Exec("DELETE FROM system_agents")
	require.NoError(t, err)
	_, err = app.pg.DB().Exec("DELETE FROM system_projects")
	require.NoError(t, err)

	// Create the system_projects table in Postgres (NewAlephApp doesn't
	// auto-create it; migrations handle that in production).
	_, err = app.pg.DB().Exec(`CREATE TABLE IF NOT EXISTS system_projects (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		name TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	// Create the system_agents table for agent creation.
	_, err = app.pg.DB().Exec(`CREATE TABLE IF NOT EXISTS system_agents (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		api_key VARCHAR(255) DEFAULT '',
		system_prompt TEXT DEFAULT '',
		skill_ids TEXT DEFAULT '[]',
		base_url TEXT DEFAULT ''
	)`)
	require.NoError(t, err)

	// Verify zero projects before setup.
	count, err := app.metaRepo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should start with zero projects")

	// Run setupDemoData synchronously.
	app.setupDemoData(projectsRoot)

	// Verify project was created.
	count, err = app.metaRepo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should have created one demo project")

	// Verify sample CSV was written.
	csvPath := filepath.Join(projectsRoot, "demo", "raw", "sample.csv")
	csvData, err := os.ReadFile(csvPath)
	require.NoError(t, err)
	assert.Contains(t, string(csvData), "Widget Alpha", "CSV should contain sample data")

	// Verify ontology file was written.
	ontPath := filepath.Join(projectsRoot, "demo", "ontologies", "core.aleph")
	ontData, err := os.ReadFile(ontPath)
	require.NoError(t, err)
	assert.Contains(t, string(ontData), "object Sales", "ontology should be created")

	// Verify raw/agents/skills/ontologies directories exist.
	for _, sub := range []string{"raw", "agents", "skills", "ontologies"} {
		info, err := os.Stat(filepath.Join(projectsRoot, "demo", sub))
		require.NoError(t, err, "subdirectory %s should exist", sub)
		assert.True(t, info.IsDir(), "%s should be a directory", sub)
	}

	// Verify agent was created.
	agents, err := app.metaRepo.ListAgents("demo")
	require.NoError(t, err)
	assert.Len(t, agents, 1, "should have one demo agent")
	assert.Equal(t, "Analista Demo", agents[0].Name)
}
