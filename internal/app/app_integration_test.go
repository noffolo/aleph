//go:build integration

package app

import (
	"context"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/config"
)

func requirePostgres(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("ALEPH_TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://localhost:5432/aleph_test?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("PostgreSQL driver open: %v (set ALEPH_TEST_POSTGRES_DSN)", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL not reachable at %s: %v (set ALEPH_TEST_POSTGRES_DSN)", dsn, err)
	}
	return dsn
}

func newIntegrationConfig(t *testing.T, duckDBPath, postgresDSN string) *config.Config {
	t.Helper()
	keyHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	encKey, err := hex.DecodeString(keyHex)
	require.NoError(t, err)
	require.Len(t, encKey, 32)

	return &config.Config{
		Port:                0,
		DataRoot:            filepath.Dir(duckDBPath),
		DuckDBPath:          duckDBPath,
		DuckDBSchema:        "main",
		PostgresDSN:         postgresDSN,
		NLPAddr:             "http://localhost:8001",
		OllamaBaseURL:       "http://localhost:11434",
		OllamaPort:          "11434",
		JWTSecret:           []byte("test-jwt-secret-that-is-at-least-32-bytes!!"),
		KeyEncryptionKey:    keyHex,
		EncryptionKey:       encKey,
		BackupInterval:      "24h",
		BackupKeep:          7,
		RateLimitChat:       10,
		RateLimitHealth:     100,
		RateLimitDefault:    500,
		MaxProjects:         50,
		MaxAgentsPerProject: 20,
		CORSAllowedOrigins:  []string{"http://localhost:5173"},
		LLMTimeoutSeconds:   30,
		DevMode:             true,
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

// TestNewAlephApp_Integration creates a full AlephApp with real DuckDB file
// and real PostgreSQL, then verifies core subsystems.
func TestNewAlephApp_Integration(t *testing.T) {
	postgresDSN := requirePostgres(t)

	tmpDir := t.TempDir()
	duckDBPath := filepath.Join(tmpDir, "test.duckdb")
	cfg := newIntegrationConfig(t, duckDBPath, postgresDSN)

	app, err := NewAlephApp(cfg, embed.FS{})
	require.NoError(t, err)
	require.NotNil(t, app)
	defer func() { _ = app.Close(context.Background()) }()

	assert.NotNil(t, app.db, "DuckDB should be initialized")
	assert.NotNil(t, app.pg, "Postgres should be initialized")
	assert.NotNil(t, app.metaRepo, "MetadataRepository should be initialized")
	assert.NotNil(t, app.eng, "Ingestion Engine should be initialized")

	var val int
	err = app.db.DB().QueryRow("SELECT 1").Scan(&val)
	assert.NoError(t, err, "DuckDB should respond")
	assert.Equal(t, 1, val)

	err = app.pg.DB().QueryRow("SELECT 1").Scan(&val)
	assert.NoError(t, err, "Postgres should respond")
	assert.Equal(t, 1, val)

	_, statErr := os.Stat(duckDBPath)
	assert.NoError(t, statErr, "DuckDB file should exist on disk")

	count, err := app.metaRepo.CountProjects()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	if err := app.Close(context.Background()); err != nil {
		t.Logf("Close returned errors: %v", err)
	}
}

// TestServe_Integration starts the full Aleph HTTP server on a real port and
// verifies health-check endpoints.
func TestServe_Integration(t *testing.T) {
	postgresDSN := requirePostgres(t)

	tmpDir := t.TempDir()
	duckDBPath := filepath.Join(tmpDir, "serve.duckdb")
	port := freePort(t)

	cfg := newIntegrationConfig(t, duckDBPath, postgresDSN)
	cfg.Port = port

	app, err := NewAlephApp(cfg, embed.FS{})
	require.NoError(t, err)
	require.NotNil(t, app)
	defer func() { _ = app.Close(context.Background()) }()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Serve(port)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	require.Eventually(t, func() bool {
		resp, e := http.Get(baseURL + "/livez")
		if e != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "server should start and respond to /livez")

	t.Run("livez", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/livez")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"status":"alive"`)
	})

	t.Run("readyz", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/readyz")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("healthz", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v1/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"status":"ok"`)
	})

	closeErr := app.Close(context.Background())
	select {
	case serveErr := <-errCh:
		assert.ErrorIs(t, serveErr, http.ErrServerClosed)
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return within 5s of Close")
	}
	if closeErr != nil {
		t.Logf("Close: %v", closeErr)
	}
}
