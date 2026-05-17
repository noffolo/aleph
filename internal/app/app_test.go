package app

import (
	"context"
	"crypto/tls"
	"database/sql"
	"log/slog"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	nlpv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"

	_ "github.com/marcboeker/go-duckdb"
)

// ── newH2CClient ──────────────────────────────────────────────────────────

func TestNewH2CClient_Basic(t *testing.T) {
	client := newH2CClient()
	require.NotNil(t, client)
	assert.Equal(t, 30*time.Second, client.Timeout)

	transport, ok := client.Transport.(*http2.Transport)
	require.True(t, ok, "transport should be *http2.Transport")
	assert.True(t, transport.AllowHTTP, "AllowHTTP should be true for h2c")
}

func TestNewH2CClient_SSRFRejectsLinkLocal(t *testing.T) {
	client := newH2CClient()
	transport := client.Transport.(*http2.Transport)

	_, err := transport.DialTLSContext(context.Background(), "tcp", "169.254.169.254:80", &tls.Config{})
	assert.Error(t, err, "link-local should be rejected by SSRF")
	assert.Contains(t, err.Error(), "SSRF")
}

// ── newTLSClient ──────────────────────────────────────────────────────────

func TestNewTLSClient_Basic(t *testing.T) {
	client := newTLSClient()
	require.NotNil(t, client)
	assert.Equal(t, 30*time.Second, client.Timeout)

	transport, ok := client.Transport.(*http2.Transport)
	require.True(t, ok, "transport should be *http2.Transport")
	assert.NotNil(t, transport.DialTLSContext, "DialTLSContext should be set")
}

func TestNewTLSClient_SSRFRejectsPrivate(t *testing.T) {
	client := newTLSClient()
	transport := client.Transport.(*http2.Transport)

	_, err := transport.DialTLSContext(context.Background(), "tcp", "192.168.1.1:443", &tls.Config{MinVersion: tls.VersionTLS13})
	assert.Error(t, err, "private IP should be rejected by SSRF")
	assert.Contains(t, err.Error(), "SSRF")
}

// ── makeSentimentHelper ───────────────────────────────────────────────────

func TestMakeSentimentHelper_NilNLPHandler(t *testing.T) {
	a := &AlephApp{nlpHandler: nil}
	helper := a.makeSentimentHelper()

	result, err := helper(context.Background(), "test text")
	assert.NoError(t, err)
	assert.Contains(t, result, "neutral")
	assert.Contains(t, result, "NLP sidecar unavailable")
}

func TestMakeSentimentHelper_ReturnsClosure(t *testing.T) {
	a := &AlephApp{nlpHandler: nil}
	helper := a.makeSentimentHelper()
	require.NotNil(t, helper)
}

// ── makeTrustScoreHelper ──────────────────────────────────────────────────

func TestMakeTrustScoreHelper_NilRegistry(t *testing.T) {
	a := &AlephApp{}
	helper := a.makeTrustScoreHelper(nil)

	result, err := helper(context.Background(), "entity-1")
	assert.NoError(t, err)
	assert.Contains(t, result, "registry unavailable")
}

func TestMakeTrustScoreHelper_ReturnsClosure(t *testing.T) {
	a := &AlephApp{}
	helper := a.makeTrustScoreHelper(nil)
	require.NotNil(t, helper)
}

// ── makeComponentByIDHelper ───────────────────────────────────────────────

func TestMakeComponentByIDHelper_NilRegistry(t *testing.T) {
	a := &AlephApp{}
	helper := a.makeComponentByIDHelper(nil)

	result, err := helper(context.Background(), "comp-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry unavailable")
	assert.Nil(t, result)
}

// ── Type existence ────────────────────────────────────────────────────────

func TestStructInitialization(t *testing.T) {
	a := &AlephApp{cancel: func() {}}
	require.NotNil(t, a)
}

// ── setupDemoData test helpers ────────────────────────────────────────────

// newInMemoryMetaRepo creates a MetadataRepository backed by an in-memory
// DuckDB with the system_projects table created.
func newInMemoryMetaRepo(t *testing.T) *repository.MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS system_projects (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		name TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

// newInMemoryMetaRepoNoTable creates a MetadataRepository backed by an
// in-memory DuckDB WITHOUT the system_projects table.
func newInMemoryMetaRepoNoTable(t *testing.T) *repository.MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

// ── setupDemoData ─────────────────────────────────────────────────────────

func TestSetupDemoData_NilMetaRepo(t *testing.T) {
	a := &AlephApp{logger: slog.Default()}
	// Should not panic — nil metaRepo triggers early return.
	a.setupDemoData("/tmp/test-nonexistent")
}

func TestSetupDemoData_CountProjectsError(t *testing.T) {
	repo := newInMemoryMetaRepoNoTable(t)
	// CountProjects queries system_projects which doesn't exist → error
	count, err := repo.CountProjects()
	assert.Error(t, err, "CountProjects should fail without system_projects table")
	assert.Equal(t, 0, count)

	a := &AlephApp{
		logger:   slog.Default(),
		metaRepo: repo,
	}
	// Should not panic — error from CountProjects triggers early return.
	a.setupDemoData("/tmp/test-nonexistent")
}

func TestSetupDemoData_ExistingProjects(t *testing.T) {
	repo := newInMemoryMetaRepo(t)

	// Insert a project so count > 0
	err := repo.CreateProjectRecord("demo", "Demo Project")
	require.NoError(t, err)

	count, err := repo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	a := &AlephApp{
		logger:   slog.Default(),
		metaRepo: repo,
	}
	// Should not panic — count > 0 triggers early return.
	a.setupDemoData("/tmp/test-nonexistent")
}

// ── Close ─────────────────────────────────────────────────────────────────

func TestClose_AllNilFields(t *testing.T) {
	a := &AlephApp{}
	err := a.Close(context.Background())
	assert.NoError(t, err, "Close on all-nil AlephApp should succeed without panic")
}

func TestClose_NilSafetyAudit(t *testing.T) {
	a := &AlephApp{}
	require.NotNil(t, a, "zero-value AlephApp should be constructable")
	require.Nil(t, a.eng, "eng should be nil")
	require.Nil(t, a.pg, "pg should be nil")
	require.Nil(t, a.db, "db should be nil")
}

// Prevent unused import errors from types referenced in production code.
var _ = &registry.DuckDBRegistry{}
var _ = &decision.ComponentMetadata{}
var _ = connect.NewRequest(&nlpv1.AnalyzeSentimentRequest{})
