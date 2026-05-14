# Aleph-v2 Integration Testing Strategy

## Test Layers

```
┌─────────────────────────────────────────────┐
│ E2E Tests (Playwright)                      │
│ Full browser flows, real backend            │
├─────────────────────────────────────────────┤
│ Integration Tests (testcontainers + httptest)│
│ Real DuckDB/Postgres, mocked external deps   │
├─────────────────────────────────────────────┤
│ Unit Tests (in-memory DuckDB + httptest)    │
│ Isolated handlers, pure functions           │
└─────────────────────────────────────────────┘
```

## Unit Tests (Current State)

**Location**: `*_test.go` alongside source files in `internal/api/handler/`

**Pattern**:
- In-memory DuckDB (`sql.Open("duckdb", ":memory:")`) with table DDL in test helpers
- `httptest.NewRecorder` for HTTP handler testing
- `connect.NewRequest` / `connect.NewResponse` for ConnectRPC handlers
- Table-driven tests with `t.Run` subtests using `testify/assert` and `require`
- Test cleanup via `t.Cleanup` and `t.TempDir`

**What to mock**:
- External services (Ollama, OpenAI, Anthropic, NLP sidecar) — use fake clients
- Email/IMAP — not tested at unit level
- File system — use `t.TempDir()`

**What NOT to mock**:
- DuckDB — use in-memory instances
- HTTP handlers — test through `httptest`
- ConnectRPC middleware — test handler methods directly

## Integration Tests

**Purpose**: Verify real database interactions, multi-component flows, and service boundary crossings.

**Tools**:
- `testcontainers-go` for PostgreSQL: spin up ephemeral Postgres containers for metadata repository tests
- `httptest.Server` with real handlers: test full request/response pipelines
- In-memory DuckDB: keep DuckDB in-memory (no container needed — DuckDB runs embedded)
- `net/http/httptest` for mock upstream servers (Ollama, NLP sidecar)

**Setup/Teardown Pattern**:
```go
func TestIntegration_ProjectLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("integration test requires containers")
    }

    ctx := context.Background()
    
    // Start PostgreSQL container
    pgContainer, err := testcontainers.GenericContainer(ctx, ...)
    require.NoError(t, err)
    defer pgContainer.Terminate(ctx)
    
    // Start DuckDB (in-memory, embedded)
    duckDB, err := storage.NewDuckDB(":memory:")
    require.NoError(t, err)
    defer duckDB.Close()
    
    // Wire real components
    metaRepo, _ := repository.NewMetadataRepository(pgDB)
    app := app.NewAlephApp(cfg, metaRepo, duckDB)
    
    // Create mock Ollama server
    ollamaServer := httptest.NewServer(http.HandlerFunc(...))
    defer ollamaServer.Close()
    
    // Test the flow
    // ...
}
```

**Integration Test Scenarios** (high-priority):
1. **Project lifecycle**: create → list → delete with real DuckDB schema creation/drop
2. **Chat flow**: create agent → send message → receive response (mock LLM backend)
3. **Ingestion pipeline**: create task → run → check progress → verify data in DuckDB
4. **Notification webhook**: send webhook → verify httptest server received it
5. **SSE streaming**: subscribe client → publish event → client receives event
6. **Tool execution**: create tool → execute → verify sandbox result
7. **API key auth**: create key → authenticate → revoke → verify rejection

**CI Integration**:
- Run unit tests always (`go test -race -count=1 ./...`)
- Run integration tests on PRs to main (`go test -race -count=1 -tags=integration ./...`)
- Use `t.Short()` or build tags to separate unit/integration
- PostgreSQL container reuse via `testcontainers` reuse mode for speed

## E2E Tests

**Location**: `frontend/tests/e2e/`

**Tools**: Playwright (already in `frontend/package.json`)

**Target Flows**:
1. Health check + login → create project → verify it appears in list
2. Create agent → send chat message → verify response appears
3. Create tool → verify tool appears in tool list → delete tool

**Configuration**: `playwright.config.ts` with:
- `webServer` pointing to `vite dev` + Go backend
- `fullyParallel: false` (sharing state)
- Screenshots on failure

## Mock Approach Summary

| Dependency | Unit Test | Integration Test | E2E Test |
|-----------|-----------|-----------------|----------|
| DuckDB | In-memory | In-memory | Real file |
| PostgreSQL | In-memory DuckDB* | testcontainer | Real instance |
| Ollama/LLM | Fake client / httptest | httptest server | Real (if available) |
| NLP sidecar | Fake ConnectRPC client | httptest gRPC server | Docker compose |
| File system | t.TempDir() | t.TempDir() | Real tmp |

*Unit tests currently use DuckDB for all DB operations including metadata that would use PostgreSQL in production. This is a known simplification — the DuckDB SQL dialect is close enough for CRUD operations.

## Running Tests

```bash
# Unit tests (fast, no external dependencies)
go test -race -count=1 ./...

# Handler tests specifically
go test -race -count=1 -timeout 60s ./internal/api/handler/

# Integration tests (requires Docker for testcontainers)
go test -race -count=1 -tags=integration ./internal/...

# E2E tests
cd frontend && npx playwright test

# All frontend tests
cd frontend && npx vitest run
```

## File Naming Convention

| Type | Pattern | Example |
|------|---------|---------|
| Unit test | `<file>_test.go` | `agent_test.go` |
| Unit test (table-driven) | Same file with `t.Run` | `TestAgentHandler_CreateAgent` |
| Integration test | `<feature>_integration_test.go` | `ingestion_integration_test.go` |
| E2E test | `<flow>.spec.ts` | `health.spec.ts` |
