package mcp

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/repository"
)

type nopWC2 struct{ *bytes.Buffer }

func (nopWC2) Close() error { return nil }

type nopRC2 struct{ *strings.Reader }

func (nopRC2) Close() error { return nil }

func TestSendRequest_ContextCancel(t *testing.T) {
	stdin := nopWC2{&bytes.Buffer{}}
	stdout := strings.NewReader("")
	stderr := nopRC2{strings.NewReader("")}
	transport := NewMCPStdioTransport(stdin, io.NopCloser(stdout), stderr)

	req, _ := NewRequest("health", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := transport.SendRequest(ctx, req)
	assert.Error(t, err)
}

func TestPing_ErrorResponse(t *testing.T) {
	resp := JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      intPtr(1),
		Error:   &JSONRPCError{Code: -32603, Message: "internal error"},
	}
	respRaw, _ := json.Marshal(resp)

	stdin := nopWC2{&bytes.Buffer{}}
	stdout := strings.NewReader(string(respRaw) + "\n")
	stderr := nopRC2{strings.NewReader("")}
	transport := NewMCPStdioTransport(stdin, io.NopCloser(stdout), stderr)
	pinger := NewMCPStdioPinger(transport, "test-server")

	err := pinger.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
}

func TestTransport_ReadStderr(t *testing.T) {
	stdin := nopWC2{&bytes.Buffer{}}
	stdout := strings.NewReader("")
	stderr := nopRC2{strings.NewReader("error: connection refused\n")}
	transport := NewMCPStdioTransport(stdin, io.NopCloser(stdout), stderr)

	data, err := transport.ReadStderr()
	assert.NoError(t, err)
	assert.Contains(t, data, "connection refused")
}

func TestTransport_Close(t *testing.T) {
	stdin := nopWC2{&bytes.Buffer{}}
	stdout := strings.NewReader("")
	stderr := nopRC2{strings.NewReader("")}
	transport := NewMCPStdioTransport(stdin, io.NopCloser(stdout), stderr)

	err := transport.Close()
	assert.NoError(t, err)
	assert.True(t, transport.IsClosed())
}

func TestErrRestartPinger_Ping(t *testing.T) {
	e := &ErrRestartPinger{Err: assert.AnError}
	err := e.Ping(context.Background())
	assert.Error(t, err)
}

func TestErrRestartPinger_Close(t *testing.T) {
	e := &ErrRestartPinger{}
	err := e.Close()
	assert.NoError(t, err)
}

func TestErrRestartPinger_Restart(t *testing.T) {
	e := &ErrRestartPinger{}
	err := e.Restart(context.Background())
	assert.Error(t, err)
}

func TestExtractToolsGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/tools/list", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tools":[{"name":"tool1","description":"desc1"},{"name":"tool2","description":"desc2"}]}`))
	}))
	defer srv.Close()

	eng := &DiscoveryEngine{httpClient: srv.Client()}
	tools, err := eng.extractToolsGet(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Len(t, tools, 2)
	assert.Equal(t, "tool1", tools[0].Name)
	assert.Equal(t, "tool2", tools[1].Name)
}

func TestExtractToolsGet_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	eng := &DiscoveryEngine{httpClient: srv.Client()}
	_, err := eng.extractToolsGet(context.Background(), srv.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestFindToolByName_Found(t *testing.T) {
	db := setupTestRepo(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type)
		VALUES ('t1','alpha','desc','code','cat','1.0','unknown','package'),
		       ('t2','beta','desc2','code2','cat2','1.0','unknown','package')`)
	require.NoError(t, err)

	eng := &DiscoveryEngine{metaRepo: mustNewRepo(t, db)}
	tool, err := eng.findToolByName(context.Background(), "beta")
	require.NoError(t, err)
	assert.Equal(t, "beta", tool.Name)
	assert.Equal(t, "t2", tool.ID)
}

func TestFindToolByName_NotFound(t *testing.T) {
	db := setupTestRepo(t)
	defer db.Close()

	eng := &DiscoveryEngine{metaRepo: mustNewRepo(t, db)}
	_, err := eng.findToolByName(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrToolNotFound)
}

func TestPinger_Close(t *testing.T) {
	stdin := nopWC2{&bytes.Buffer{}}
	stdout := strings.NewReader("")
	stderr := nopRC2{strings.NewReader("")}
	transport := NewMCPStdioTransport(stdin, io.NopCloser(stdout), stderr)
	pinger := NewMCPStdioPinger(transport, "test-server")

	err := pinger.Close()
	assert.NoError(t, err)
	assert.True(t, transport.IsClosed())
}

func TestPing_Success(t *testing.T) {
	resp := JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      intPtr(1),
		Result:  json.RawMessage(`{}`),
	}
	respRaw, _ := json.Marshal(resp)

	stdin := nopWC2{&bytes.Buffer{}}
	stdout := strings.NewReader(string(respRaw) + "\n")
	stderr := nopRC2{strings.NewReader("")}
	transport := NewMCPStdioTransport(stdin, io.NopCloser(stdout), stderr)
	pinger := NewMCPStdioPinger(transport, "test-server")

	err := pinger.Ping(context.Background())
	assert.NoError(t, err)
}

func setupTestRepo(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS system_tools (
		id VARCHAR PRIMARY KEY,
		name VARCHAR NOT NULL DEFAULT '',
		description VARCHAR NOT NULL DEFAULT '',
		code VARCHAR NOT NULL DEFAULT '',
		category VARCHAR NOT NULL DEFAULT '',
		version VARCHAR NOT NULL DEFAULT '',
		health_status VARCHAR NOT NULL DEFAULT 'unknown',
		source_type VARCHAR NOT NULL DEFAULT ''
	)`)
	require.NoError(t, err)
	return db
}

func mustNewRepo(t *testing.T, db *sql.DB) *repository.MetadataRepository {
	t.Helper()
	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}
