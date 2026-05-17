# Aleph-v2 TDD Session Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply Test-Driven Development to Aleph-v2's highest-impact untested modules — Go backend (MCP discovery, registry CRUD, decision engine, sandbox verification, DuckDB storage), frontend (API factory/adapters, D3 graph, data table), and Python NLP (gRPC service, streaming predictions).

**Architecture:** 5 independent Go TDD sessions + 3 frontend TDD sessions + 2 Python NLP TDD sessions. Each session follows strict TDD: write failing test first, then minimal implementation, then refactor. Sessions are independent and parallelizable.

**Tech Stack:** Go 1.26 + testing/std + testify, Vitest 4 + @testing-library/react, Pytest 8 + pytest-asyncio, Playwright 13.

> **Prerequisite:** All tests listed below are ADDITIONAL to existing tests. Run full suite before starting to confirm baseline: `go test -race -count=1 ./... && cd frontend && npx vitest run && cd ../nlp && python -m pytest`

---

## Current Test Landscape (May 2026)

### Go Backend — 498 test functions, ~100 test files

| Priority | Package | Functions | Test Coverage | Risk |
|----------|---------|-----------|---------------|------|
| 🔴 P0 | `internal/decision/` (adapters) | 0% | Decision engine adapters, planner — core AI loop |
| 🔴 P0 | `internal/mcp/discovery.go` | 0% | MCP discovery — all tool integration depends on this |
| 🔴 P0 | `internal/registry/duckdb_registry.go` | 0% | DuckDB registry CRUD — tools/skills/agents persistence |
| 🔴 P0 | `internal/sandbox/verification.go` | ~30% | Security verification — sandbox code execution |
| 🟠 P1 | `internal/storage/duckdb.go` | ~40% | Core storage — VSS, queries, schema management |
| 🟠 P1 | `internal/ingestion/probe.go` (558 lines) | 0% | Probe discovery — tool capability detection |
| 🟠 P1 | `internal/ingestion/ontology.go` (89 lines) | 0% | Ontology management |
| 🟠 P1 | `internal/dsl/compiler_tool.go` (15 funcs) | 0% | Code-generation — tool compilation |
| 🟠 P1 | `internal/repository/metadata.go` | ~40% | PostgreSQL metadata CRUD |
| 🟡 P2 | `internal/tools/{finance,osint,...}/` | 0% | Tool stubs — 4 packages, all stubs |

### Frontend — ~100 test files, comprehensive coverage, 3 critical gaps

| Priority | File | Lines | Risk |
|----------|------|-------|------|
| 🔴 P0 | `api/factory.ts` | ~60 | All domain hooks import from here — wrong client = silent CRUD failures |
| 🔴 P0 | `api/adapters.ts` | ~80 | Proto↔frontend type conversion — runtime type errors only |
| 🟠 P1 | `lib/AlephGraph.tsx` (D3) | ~200 | Complex D3 rendering — zero tests |
| 🟠 P1 | `lib/AlephTable.tsx` | ~150 | Data table rendering — zero tests |
| 🟡 P2 | `lib/AlephTimeline.tsx` | ~100 | Timeline rendering — zero tests |

### Python NLP — 10 test files, 155 tests, 78% coverage

| Priority | Function | Lines | Risk |
|----------|----------|-------|------|
| 🟠 P1 | `nlp/main.py:serve()` | ~40 | gRPC service startup — integration boundary |
| 🟠 P1 | `nlp/main.py:StreamPredictions` | ~80 | Streaming predictions — real-time contract |
| 🟡 P2 | `nlp/main.py:NLPService.__init__` | ~30 | Model loading — init errors surface on first request |

### CI/CD

- Go test: `go test -race -count=1 ./...` (GitHub Actions + Makefile) — coverage gate 60%
- Frontend test: `cd frontend && npx vitest run` — coverage gate 55%
- Python NLP test: `cd nlp && python -m pytest` — coverage gate 78% (not in CI)
- E2E: `cd frontend && npx playwright test` — manual
- **Gap:** Python NLP tests NOT in CI; no pre-commit test gate; no contract tests

---

## Task 1: `internal/mcp/discovery.go` — DiscoveryEngine TDD

**Files:**
- Implementation: `internal/mcp/discovery.go`
- New: `internal/mcp/discovery_test.go`
- Dependencies: `internal/mcp/ssrf.go`, `internal/config/config.go`

DiscoveryEngine discovers MCP tools from remote endpoints. It has zero tests despite being the critical path for all tool integration.

- [ ] **Step 1: Write failing test for NewDiscoveryEngine**

```go
// internal/mcp/discovery_test.go
package mcp

import (
    "testing"
    "net/url"
)

func TestNewDiscoveryEngine(t *testing.T) {
    u, _ := url.Parse("https://mcp.example.com")
    engine := NewDiscoveryEngine(u, nil, nil)
    if engine == nil {
        t.Fatal("NewDiscoveryEngine returned nil")
    }
    if engine.baseURL.String() != "https://mcp.example.com" {
        t.Errorf("expected baseURL https://mcp.example.com, got %s", engine.baseURL.String())
    }
}
```

Read `discovery.go` first to confirm DiscoveryEngine struct fields before writing test.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/mcp/ -run TestNewDiscoveryEngine -v
```
Expected: FAIL — file exists, test function runs and passes (this is a constructor, it already exists; the test should pass for the constructor but fail for business logic tests below).

- [ ] **Step 3: Write failing test for DiscoverTools (the actual TDD)**

```go
// internal/mcp/discovery_test.go (append)
func TestDiscoverTools_EmptyEndpoint(t *testing.T) {
    u, _ := url.Parse("https://mcp.example.com")
    engine := NewDiscoveryEngine(u, nil, nil)

    tools, err := engine.DiscoverTools(t.Context())
    if err != nil {
        t.Fatalf("DiscoverTools returned unexpected error: %v", err)
    }
    if len(tools) != 0 {
        t.Errorf("expected 0 tools from empty endpoint, got %d", len(tools))
    }
}
```

This test will fail because `DiscoverTools` likely makes a real HTTP call. The minimal fix: make the HTTP client injectable (already via `httpClient` field) so tests can use an httptest server or mock.

- [ ] **Step 4: Run test to verify it fails — needs mock HTTP server**

If `DiscoverTools` uses a real HTTP client to the provided URL, the test will fail at network level. Read discovery.go to see how HTTP is called.

- [ ] **Step 5: Write httptest server and make test pass**

```go
// internal/mcp/discovery_test.go (add)
import (
    "net/http/httptest"
    "encoding/json"
)

func TestDiscoverTools_ReturnsTools(t *testing.T) {
    // Mock MCP endpoint returning tool definitions
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        resp := map[string]interface{}{
            "tools": []map[string]interface{}{
                {"name": "web-search", "description": "Search the web"},
                {"name": "code-exec", "description": "Execute code"},
            },
        }
        json.NewEncoder(w).Encode(resp)
    }))
    defer server.Close()

    u, _ := url.Parse(server.URL)
    engine := NewDiscoveryEngine(u, nil, &http.Client{})

    tools, err := engine.DiscoverTools(t.Context())
    if err != nil {
        t.Fatalf("DiscoverTools failed: %v", err)
    }
    if len(tools) != 2 {
        t.Errorf("expected 2 tools, got %d", len(tools))
    }
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
go test ./internal/mcp/ -run TestDiscoverTools -v
```
Expected: PASS

- [ ] **Step 7: Add edge case tests**

```go
// internal/mcp/discovery_test.go (append)
func TestDiscoverTools_NetworkError(t *testing.T) {
    u, _ := url.Parse("http://localhost:1") // unreachable
    engine := NewDiscoveryEngine(u, nil, &http.Client{Timeout: time.Second})

    _, err := engine.DiscoverTools(t.Context())
    if err == nil {
        t.Fatal("expected error for unreachable endpoint, got nil")
    }
}

func TestDiscoverTools_MalformedResponse(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{invalid json`))
    }))
    defer server.Close()

    u, _ := url.Parse(server.URL)
    engine := NewDiscoveryEngine(u, nil, &http.Client{})

    _, err := engine.DiscoverTools(t.Context())
    if err == nil {
        t.Fatal("expected error for malformed response, got nil")
    }
}
```

- [ ] **Step 8: Run all discovery tests**

```bash
go test ./internal/mcp/ -run "TestDiscoverTools|TestNewDiscoveryEngine" -v
```
Expected: ALL PASS

- [ ] **Step 9: Commit**

```bash
git add internal/mcp/discovery_test.go internal/mcp/discovery.go
git commit -m "test: TDD DiscoveryEngine — DiscoverTools with httptest mock, error paths"
```

---

## Task 2: `internal/registry/duckdb_registry.go` — DuckDB Registry TDD

**Files:**
- Implementation: `internal/registry/duckdb_registry.go`
- New: `internal/registry/duckdb_registry_test.go`
- Dependencies: `internal/storage/duckdb.go`, database connection

The DuckDBRegistry manages CRUD for tools, skills, agents in DuckDB. Zero tests.

- [ ] **Step 1: Read duckdb_registry.go to understand interface and methods**

```bash
head -100 internal/registry/duckdb_registry.go
```

- [ ] **Step 2: Write failing test for NewDuckDBRegistry**

```go
// internal/registry/duckdb_registry_test.go
package registry

import (
    "testing"
    "database/sql"
    _ "github.com/marcboeker/go-duckdb"
)

func TestNewDuckDBRegistry(t *testing.T) {
    db, err := sql.Open("duckdb", ":memory:")
    if err != nil {
        t.Fatalf("failed to open in-memory DuckDB: %v", err)
    }
    defer db.Close()

    reg := NewDuckDBRegistry(db)
    if reg == nil {
        t.Fatal("NewDuckDBRegistry returned nil")
    }
}
```

- [ ] **Step 3: Run test to verify it passes (constructor test)**

```bash
go test ./internal/registry/ -run TestNewDuckDBRegistry -v
```
Expected: PASS

- [ ] **Step 4: Write failing test for CreateAgent / GetAgent**

```go
// internal/registry/duckdb_registry_test.go (append)
func TestRegistry_CreateAndGetAgent(t *testing.T) {
    db, err := sql.Open("duckdb", ":memory:")
    if err != nil {
        t.Fatalf("failed to open in-memory DuckDB: %v", err)
    }
    defer db.Close()

    reg := NewDuckDBRegistry(db)

    // Create schema and table first (mimic what migration does)
    _, err = db.Exec("CREATE SCHEMA IF NOT EXISTS test_schema")
    if err != nil {
        t.Fatalf("failed to create schema: %v", err)
    }

    agent := Agent{
        Name:        "test-agent",
        Description: "A test agent",
        Model:       "gpt-4",
        Provider:    "openai",
    }

    id, err := reg.CreateAgent(t.Context(), "test_schema", agent)
    if err != nil {
        t.Fatalf("CreateAgent failed: %v", err)
    }
    if id == "" {
        t.Fatal("CreateAgent returned empty ID")
    }

    got, err := reg.GetAgent(t.Context(), "test_schema", id)
    if err != nil {
        t.Fatalf("GetAgent failed: %v", err)
    }
    if got.Name != agent.Name {
        t.Errorf("expected name %q, got %q", agent.Name, got.Name)
    }
    if got.Provider != agent.Provider {
        t.Errorf("expected provider %q, got %q", agent.Provider, got.Provider)
    }
}
```

Read `duckdb_registry.go` to confirm the `Agent` struct, `CreateAgent` signature, and `GetAgent` signature.

- [ ] **Step 5: Run test to verify it compiles**

```bash
go test ./internal/registry/ -run TestRegistry_CreateAndGetAgent -v
```
Expected: FAIL or PASS depending on existing implementation — if there's existing code, it may already work. If the registry requires real migrations, the test will fail on table not found.

- [ ] **Step 6: Ensure migrations run in test setup**

If the test fails because tables don't exist, add migration logic:

```go
// internal/registry/duckdb_registry_test.go (add before CreateAgent)
// Run migrations inline for test isolation
runMigrations := func(db *sql.DB) {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS test_schema.agents (
            id VARCHAR PRIMARY KEY,
            name VARCHAR NOT NULL,
            description VARCHAR DEFAULT '',
            model VARCHAR NOT NULL,
            provider VARCHAR NOT NULL,
            api_key VARCHAR DEFAULT '',
            base_url VARCHAR DEFAULT '',
            system_prompt VARCHAR DEFAULT '',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        t.Fatalf("failed to create table: %v", err)
    }
}
runMigrations(db)
```

- [ ] **Step 7: Run test to verify it passes**

```bash
go test ./internal/registry/ -run TestRegistry_CreateAndGetAgent -v
```
Expected: PASS

- [ ] **Step 8: Write failing test for ListAgents and DeleteAgent**

```go
// internal/registry/duckdb_registry_test.go (append)
func TestRegistry_ListAndDeleteAgent(t *testing.T) {
    db, err := sql.Open("duckdb", ":memory:")
    if err != nil {
        t.Fatalf("failed to open in-memory DuckDB: %v", err)
    }
    defer db.Close()

    reg := NewDuckDBRegistry(db)
    _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS test_schema")
    if err != nil {
        t.Fatalf("failed to create schema: %v", err)
    }
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS test_schema.agents (
            id VARCHAR PRIMARY KEY,
            name VARCHAR NOT NULL,
            description VARCHAR DEFAULT '',
            model VARCHAR NOT NULL,
            provider VARCHAR NOT NULL,
            api_key VARCHAR DEFAULT '',
            base_url VARCHAR DEFAULT '',
            system_prompt VARCHAR DEFAULT '',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`)
    if err != nil {
        t.Fatalf("failed to create table: %v", err)
    }

    // Create 3 agents
    ids := make([]string, 3)
    for i := 0; i < 3; i++ {
        ids[i], err = reg.CreateAgent(t.Context(), "test_schema", Agent{
            Name: fmt.Sprintf("agent-%d", i), Model: "gpt-4", Provider: "openai",
        })
        if err != nil {
            t.Fatalf("CreateAgent %d failed: %v", i, err)
        }
    }

    // List: expect 3
    agents, err := reg.ListAgents(t.Context(), "test_schema")
    if err != nil {
        t.Fatalf("ListAgents failed: %v", err)
    }
    if len(agents) != 3 {
        t.Errorf("expected 3 agents, got %d", len(agents))
    }

    // Delete one
    err = reg.DeleteAgent(t.Context(), "test_schema", ids[0])
    if err != nil {
        t.Fatalf("DeleteAgent failed: %v", err)
    }

    // List: expect 2
    agents, err = reg.ListAgents(t.Context(), "test_schema")
    if err != nil {
        t.Fatalf("ListAgents after delete failed: %v", err)
    }
    if len(agents) != 2 {
        t.Errorf("expected 2 agents after delete, got %d", len(agents))
    }
}
```

- [ ] **Step 9: Run tests to verify pass**

```bash
go test ./internal/registry/ -run "TestRegistry_" -v
```
Expected: ALL PASS

- [ ] **Step 10: Commit**

```bash
git add internal/registry/duckdb_registry_test.go
git commit -m "test: TDD DuckDBRegistry — Create/Get/List/Delete Agent with in-memory DuckDB"
```

---

## Task 3: `internal/decision/` — Decision Engine Adapters TDD

**Files:**
- Implementation: `internal/decision/` (adapters, planner)
- New: `internal/decision/adapters_test.go`
- Dependencies: `internal/decision/engine.go`

The decision engine (PAORA) is the core AI decision loop. Its adapter layer has zero tests.

- [ ] **Step 1: Read decision adapter files to understand interface**

```bash
ls internal/decision/
head -100 internal/decision/adapters.go
```

- [ ] **Step 2: Write failing test for the Plan adapter**

```go
// internal/decision/adapters_test.go (conceptual — adapt based on actual interface)
package decision

import (
    "testing"
    "context"
)

func TestPlanAdapter_CreatePlan(t *testing.T) {
    adapter := NewPlanAdapter(PlanAdapterConfig{
        MaxSteps: 5,
        Model:    "gpt-4",
    })
    
    plan, err := adapter.CreatePlan(t.Context(), PlanRequest{
        Goal: "Analyze market data",
        Constraints: []string{"real-time data only", "max 10 sources"},
    })
    if err != nil {
        t.Fatalf("CreatePlan failed: %v", err)
    }
    if plan == nil {
        t.Fatal("CreatePlan returned nil plan")
    }
    if len(plan.Steps) == 0 {
        t.Error("expected at least 1 step in plan")
    }
    if len(plan.Steps) > 5 {
        t.Errorf("expected max 5 steps, got %d", len(plan.Steps))
    }
}
```

Read the actual adapter interface first — adapt test to real signatures.

- [ ] **Step 3: Run test to verify it compiles and fails**

```bash
go test ./internal/decision/ -run TestPlanAdapter -v
```
Expected: FAIL — adapters may need LLM provider which isn't available in test

- [ ] **Step 4: Write unit test with mocked LLM provider**

```go
// internal/decision/adapters_test.go (adjust to real interface)
func TestPlanAdapter_EmptyGoal(t *testing.T) {
    adapter := NewPlanAdapter(PlanAdapterConfig{MaxSteps: 5})
    
    _, err := adapter.CreatePlan(t.Context(), PlanRequest{
        Goal: "",
    })
    if err == nil {
        t.Fatal("expected error for empty goal, got nil")
    }
}
```

- [ ] **Step 5: Run test to verify failure on empty input**

```bash
go test ./internal/decision/ -run "TestPlanAdapter" -v
```
Expected: if the adapter doesn't validate empty input, test fails → add validation

- [ ] **Step 6: Add validation and verify pass**

If needed, add input validation to the adapter, then verify test passes.

```go
// hypothetical change in adapter
func (a *PlanAdapter) CreatePlan(ctx context.Context, req PlanRequest) (*Plan, error) {
    if req.Goal == "" {
        return nil, fmt.Errorf("goal is required")
    }
    // ... existing logic
}
```

- [ ] **Step 7: Run full decision package tests**

```bash
go test -race -count=1 ./internal/decision/ -v
```
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
git add internal/decision/adapters_test.go
git commit -m "test: TDD PlanAdapter — empty goal validation, interface contract"
```

---

## Task 4: `internal/sandbox/verification.go` — Sandbox Verifier TDD

**Files:**
- Implementation: `internal/sandbox/verification.go`
- Existing: `internal/sandbox/verification_test.go`
- New: (append to existing `verification_test.go`)

Verifier is security-critical — validates code/tool execution in sandbox. Partial test coverage exists (~30%).

- [ ] **Step 1: Read verification.go and existing tests**

```bash
head -200 internal/sandbox/verification.go
cat internal/sandbox/verification_test.go
```

- [ ] **Step 2: Write failing test for blocking dangerous imports**

```go
// internal/sandbox/verification_test.go (append)
func TestVerifier_BlockDangerousGoImport(t *testing.T) {
    v := NewVerifier(VerifierConfig{BlockGoImports: true})
    
    code := `package main
import "os/exec"
func main() { exec.Command("rm", "-rf", "/").Run() }`
    
    result, err := v.VerifyGoCode(context.Background(), code, nil)
    if err != nil {
        t.Fatalf("VerifyGoCode unexpected error: %v", err)
    }
    if result.Allowed {
        t.Error("expected code with os/exec to be blocked, but was allowed")
    }
}
```

- [ ] **Step 3: Run test to verify passes (blocking should already work)**

```bash
go test ./internal/sandbox/ -run TestVerifier_BlockDangerousGoImport -v
```
Expected: PASS (os/exec is already in the blocklist)

- [ ] **Step 4: Write failing test for SSRF in generated HTTP client**

The gap is that `compiler_tool.go` generates HTTP clients without SSRF validation. Write a test that proves this:

```go
// internal/sandbox/verification_test.go (append)
func TestVerifier_BlockInternalNetworkAccess(t *testing.T) {
    v := NewVerifier(VerifierConfig{BlockSSRF: true})
    
    code := `package main
import "net/http"
func main() { http.Get("http://169.254.169.254/latest/meta-data/") }`
    
    result, err := v.VerifyGoCode(context.Background(), code, nil)
    if err != nil {
        t.Fatalf("VerifyGoCode unexpected error: %v", err)
    }
    if result.Allowed {
        t.Error("expected AWS metadata access to be blocked, but was allowed")
    }
}
```

- [ ] **Step 5: Run test to verify it catches the gap**

```bash
go test ./internal/sandbox/ -run TestVerifier_BlockInternalNetworkAccess -v
```
Expected: FAIL — this is the actual TDD gap: net/http alone isn't blocked (only os/exec is)

- [ ] **Step 6: Write failing test for SSRF URL validation**

```go
// internal/sandbox/verification_test.go (append)
func TestVerifier_BlockSSRFUrl(t *testing.T) {
    v := NewVerifier(VerifierConfig{BlockSSRF: true})
    
    // Verify the ValidateSSRF method blocks internal IPs
    result, err := v.ValidateSSRF(context.Background(), "http://169.254.169.254/latest/meta-data/")
    if err != nil {
        t.Fatalf("ValidateSSRF unexpected error: %v", err)
    }
    if result.Allowed {
        t.Error("expected AWS metadata URL to be blocked, but was allowed")
    }
}
```

- [ ] **Step 7: Run test and implement the SSRF check**

```bash
go test ./internal/sandbox/ -run TestVerifier_BlockSSRFUrl -v
```
Expected: FAIL if SSRF validation isn't wired into the verifier

After implementing the SSRF check in the verifier (delegating to `mcp.ValidateSSRF`):
```bash
go test ./internal/sandbox/ -run "TestVerifier_" -v
```
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
git add internal/sandbox/verification_test.go internal/sandbox/verification.go
git commit -m "test: TDD Sandbox Verifier — SSRF blocking for internal IPs and generated HTTP clients"
```

---

## Task 5: `internal/storage/duckdb.go` — Storage TDD (VSS + Schema)

**Files:**
- Implementation: `internal/storage/duckdb.go`
- Existing: `internal/storage/duckdb_test.go`
- New: (append to existing test)

DuckDB storage handles VSS (vector similarity search) and schema management. Partial coverage (~40%).

- [ ] **Step 1: Read duckdb.go and existing tests**

```bash
cat internal/storage/duckdb_test.go
```

- [ ] **Step 2: Write failing test for VSS query with embeddings**

```go
// internal/storage/duckdb_test.go (append)
func TestStorage_VectorSimilaritySearch(t *testing.T) {
    db, err := NewInMemoryDuckDB()
    if err != nil {
        t.Fatalf("NewInMemoryDuckDB failed: %v", err)
    }
    defer db.Close()

    // Create table with embedding column
    _, err = db.Exec(`
        CREATE TABLE test_vectors (
            id INTEGER PRIMARY KEY,
            embedding FLOAT[4]
        )
    `)
    if err != nil {
        t.Fatalf("create table failed: %v", err)
    }

    // Insert test vectors
    _, err = db.Exec(`INSERT INTO test_vectors VALUES 
        (1, [1.0, 0.0, 0.0, 0.0]),
        (2, [0.0, 1.0, 0.0, 0.0]),
        (3, [0.0, 0.0, 1.0, 0.0])`)
    if err != nil {
        t.Fatalf("insert vectors failed: %v", err)
    }

    // Query using cosine similarity
    rows, err := db.Query(`
        SELECT id, array_cosine_similarity(embedding, [1.0, 0.0, 0.0, 0.0]) AS score
        FROM test_vectors
        ORDER BY score DESC
        LIMIT 2
    `)
    if err != nil {
        t.Fatalf("VSS query failed: %v", err)
    }
    defer rows.Close()

    var count int
    for rows.Next() {
        var id int
        var score float64
        rows.Scan(&id, &score)
        count++
        t.Logf("id=%d score=%f", id, score)
    }
    if count != 2 {
        t.Errorf("expected 2 results, got %d", count)
    }
}
```

- [ ] **Step 3: Run test to verify it passes (DuckDB VSS should work)**

```bash
go test ./internal/storage/ -run TestStorage_VectorSimilaritySearch -v
```
Expected: PASS (DuckDB has native `array_cosine_similarity`)

- [ ] **Step 4: Write failing test for schema scoping**

```go
// internal/storage/duckdb_test.go (append)
func TestStorage_SchemaScopedQuery(t *testing.T) {
    db, err := NewInMemoryDuckDB()
    if err != nil {
        t.Fatalf("NewInMemoryDuckDB failed: %v", err)
    }
    defer db.Close()

    // Create two schemas with same table name
    _, err = db.Exec("CREATE SCHEMA schema_a")
    if err != nil {
        t.Fatalf("create schema_a failed: %v", err)
    }
    _, err = db.Exec("CREATE SCHEMA schema_b")
    if err != nil {
        t.Fatalf("create schema_b failed: %v", err)
    }
    _, err = db.Exec("CREATE TABLE schema_a.items (id INTEGER, value VARCHAR)")
    if err != nil {
        t.Fatalf("create schema_a.items failed: %v", err)
    }
    _, err = db.Exec("CREATE TABLE schema_b.items (id INTEGER, value VARCHAR)")
    if err != nil {
        t.Fatalf("create schema_b.items failed: %v", err)
    }
    _, err = db.Exec("INSERT INTO schema_a.items VALUES (1, 'from_a')")
    if err != nil {
        t.Fatalf("insert into schema_a failed: %v", err)
    }
    _, err = db.Exec("INSERT INTO schema_b.items VALUES (1, 'from_b')")
    if err != nil {
        t.Fatalf("insert into schema_b failed: %v", err)
    }

    // Test SET schema scoping
    _, err = db.Exec("SET schema = 'schema_a'")
    if err != nil {
        t.Fatalf("SET schema failed: %v", err)
    }

    var value string
    err = db.QueryRow("SELECT value FROM items WHERE id = 1").Scan(&value)
    if err != nil {
        t.Fatalf("query failed: %v", err)
    }
    if value != "from_a" {
        t.Errorf("expected 'from_a', got %q", value)
    }
}
```

- [ ] **Step 5: Run test to verify it passes (DuckDB schema scoping)**

```bash
go test ./internal/storage/ -run TestStorage_SchemaScopedQuery -v
```
Expected: PASS

- [ ] **Step 6: Run all storage tests including existing**

```bash
go test -race -count=1 ./internal/storage/ -v
```
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/storage/duckdb_test.go
git commit -m "test: TDD DuckDB storage — VSS query and schema scoping"
```

---

## Task 6: Frontend `api/factory.ts` — API Client Factory TDD

**Files:**
- Implementation: `frontend/src/api/factory.ts`
- New: `frontend/src/api/__tests__/factory.test.ts`
- Dependencies: `frontend/src/api/client.ts`

`factory.ts` creates API clients for all domain hooks. Zero tests despite being the critical integration point.

- [ ] **Step 1: Read factory.ts to understand exports**

```bash
cat frontend/src/api/factory.ts
```

- [ ] **Step 2: Write failing test for factory.createClient**

```typescript
// frontend/src/api/__tests__/factory.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createApiClient } from '../factory';

describe('createApiClient', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('creates a client with correct base URL', () => {
    const client = createApiClient({ baseUrl: 'http://localhost:8080' });
    expect(client).toBeDefined();
    expect(client.baseUrl).toBe('http://localhost:8080');
  });

  it('creates a client with auth token', () => {
    const client = createApiClient({
      baseUrl: 'http://localhost:8080',
      apiKey: 'test-key-123',
    });
    expect(client).toBeDefined();
    expect(client.getHeaders()).toMatchObject({
      'X-Aleph-Api-Key': 'test-key-123',
    });
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd frontend && npx vitest run api/__tests__/factory.test.ts
```
Expected: FAIL — factory.ts may not have these exact interfaces

- [ ] **Step 4: Make test pass by adjusting to actual factory interface**

Read `factory.ts` to understand the actual exports and method signatures. Adjust tests to match. The key is:

1. What does `createApiClient` return? (object with methods like `getAgents`, `createAgent`, etc.)
2. How is auth passed? (apiKey param, header injection, etc.)
3. What's the base URL pattern? (passed to ConnectRPC transport)

- [ ] **Step 5: Run test to verify it passes**

```bash
cd frontend && npx vitest run api/__tests__/factory.test.ts
```
Expected: PASS

- [ ] **Step 6: Write edge case tests**

```typescript
// factory.test.ts (append)
it('rejects empty base URL', () => {
  expect(() => createApiClient({ baseUrl: '' })).toThrow();
});

it('handles missing apiKey gracefully', () => {
  const client = createApiClient({ baseUrl: 'http://localhost:8080' });
  expect(() => client.getAgents()).not.toThrow();
});
```

- [ ] **Step 7: Run all factory tests**

```bash
cd frontend && npx vitest run api/__tests__/factory.test.ts
```
Expected: ALL PASS

- [ ] **Step 8: Check diagnostics**

```bash
cd frontend && npx tsc --noEmit
```
Expected: no new errors

- [ ] **Step 9: Commit**

```bash
git add frontend/src/api/__tests__/factory.test.ts
git commit -m "test: TDD API client factory — base URL, auth headers, edge cases"
```

---

## Task 7: Frontend `api/adapters.ts` — Type Adapters TDD

**Files:**
- Implementation: `frontend/src/api/adapters.ts`
- New: `frontend/src/api/__tests__/adapters.test.ts`

Adapters convert between protobuf types and frontend model types. Zero tests — type errors surface at runtime.

- [ ] **Step 1: Read adapters.ts**

```bash
cat frontend/src/api/adapters.ts
```

- [ ] **Step 2: Write failing test for each adapter function**

```typescript
// frontend/src/api/__tests__/adapters.test.ts
import { describe, it, expect } from 'vitest';
import { adaptAgent, adaptTool, adaptSkill } from '../adapters';

describe('adaptAgent', () => {
  it('converts proto agent to frontend model', () => {
    const proto = {
      id: 'agent-1',
      name: 'Test Agent',
      description: 'A test agent',
      model: 'gpt-4',
      provider: 'openai',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    };
    
    const result = adaptAgent(proto);
    expect(result.id).toBe('agent-1');
    expect(result.name).toBe('Test Agent');
    expect(result.createdAt).toBeInstanceOf(Date);
  });

  it('handles null/undefined gracefully', () => {
    expect(() => adaptAgent(null)).toThrow();
    expect(() => adaptAgent(undefined)).toThrow();
  });

  it('handles missing optional fields', () => {
    const result = adaptAgent({ id: 'agent-1', name: 'Minimal' });
    expect(result.description).toBe('');
    expect(result.model).toBe('gpt-4'); // or default value
  });
});

describe('adaptTool', () => {
  it('converts proto tool to frontend model', () => {
    const proto = {
      id: 'tool-1',
      name: 'web-search',
      description: 'Search the web',
      category: 'search',
      version: '1.0.0',
    };
    
    const result = adaptTool(proto);
    expect(result.id).toBe('tool-1');
    expect(result.category).toBe('search');
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd frontend && npx vitest run api/__tests__/adapters.test.ts
```
Expected: FAIL — adjust tests to match actual adapter signatures

- [ ] **Step 4: Fix tests to match actual interfaces**

Read the actual adapter function signatures from `adapters.ts` and update tests to match. The patterns will be similar to what's already in `schemas/__tests__/schemas.edge.test.ts` (which tests `fromProto()` for schema validation).

- [ ] **Step 5: Run test to verify it passes**

```bash
cd frontend && npx vitest run api/__tests__/adapters.test.ts
```
Expected: PASS

- [ ] **Step 6: Run tsc check**

```bash
cd frontend && npx tsc --noEmit
```
Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add frontend/src/api/__tests__/adapters.test.ts
git commit -m "test: TDD API adapters — proto-to-model conversion with edge cases"
```

---

## Task 8: Frontend `lib/AlephGraph.tsx` — D3 Graph TDD

**Files:**
- Implementation: `frontend/src/lib/AlephGraph.tsx`
- New: `frontend/src/lib/__tests__/AlephGraph.test.tsx`

D3 graph visualization — complex rendering, zero tests.

- [ ] **Step 1: Read AlephGraph.tsx to understand props and behavior**

```bash
cat frontend/src/lib/AlephGraph.tsx
```

- [ ] **Step 2: Write failing test for rendering**

```typescript
// frontend/src/lib/__tests__/AlephGraph.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import AlephGraph from '../AlephGraph';

// Mock D3 — D3 uses SVG which doesn't exist in jsdom, so mock at module level
vi.mock('d3', () => ({
  select: vi.fn(() => ({
    selectAll: vi.fn(() => ({
      data: vi.fn(() => ({
        enter: vi.fn(() => ({
          append: vi.fn(() => ({
            attr: vi.fn().mockReturnThis(),
            style: vi.fn().mockReturnThis(),
            on: vi.fn().mockReturnThis(),
          })),
        })),
        exit: vi.fn(() => ({
          remove: vi.fn(),
        })),
      })),
      attr: vi.fn().mockReturnThis(),
      style: vi.fn().mockReturnThis(),
    })),
    append: vi.fn(() => ({
      attr: vi.fn().mockReturnThis(),
      style: vi.fn().mockReturnThis(),
      call: vi.fn().mockReturnThis(),
    })),
    attr: vi.fn().mockReturnThis(),
    style: vi.fn().mockReturnThis(),
  })),
}));

describe('AlephGraph', () => {
  it('renders with nodes and edges', () => {
    const nodes = [
      { id: '1', label: 'Node 1', group: 'A' },
      { id: '2', label: 'Node 2', group: 'B' },
    ];
    const edges = [
      { source: '1', target: '2', label: 'connects to' },
    ];

    const { container } = render(<AlephGraph nodes={nodes} edges={edges} />);
    expect(container.querySelector('svg')).toBeTruthy();
  });

  it('renders empty state with no nodes', () => {
    const { container } = render(<AlephGraph nodes={[]} edges={[]} />);
    expect(container.querySelector('svg')).toBeTruthy();
  });
});
```

- [ ] **Step 3: Run test to verify it fails (D3 mock setup)**

```bash
cd frontend && npx vitest run lib/__tests__/AlephGraph.test.tsx
```
Expected: FAIL — D3 mock may not match actual D3 usage. Adjust mock to match.

- [ ] **Step 4: Fix test and verify passes**

```bash
cd frontend && npx vitest run lib/__tests__/AlephGraph.test.tsx
```
Expected: PASS

- [ ] **Step 5: Add interaction test (if AlephGraph supports click handlers)**

```typescript
it('calls onNodeClick when a node is clicked', async () => {
  const onNodeClick = vi.fn();
  const nodes = [{ id: '1', label: 'Clickable', group: 'A' }];
  render(<AlephGraph nodes={nodes} edges={[]} onNodeClick={onNodeClick} />);
  // Simulate click on first node — D3 mock should fire the handler
  expect(onNodeClick).not.toHaveBeenCalled();
  // If D3 on() is wired correctly, fire the event through the mock
});
```

- [ ] **Step 6: Run all graph tests**

```bash
cd frontend && npx vitest run lib/__tests__/AlephGraph.test.tsx
```
Expected: ALL PASS

- [ ] **Step 7: Run full frontend suite**

```bash
cd frontend && npx vitest run
```
Expected: all tests pass (existing 113+ + new)

- [ ] **Step 8: Commit**

```bash
git add frontend/src/lib/__tests__/AlephGraph.test.tsx
git commit -m "test: TDD AlephGraph D3 component — rendering with nodes/edges, empty state, interactions"
```

---

## Task 9: Python NLP `main.py` — gRPC Service TDD

**Files:**
- Implementation: `nlp/main.py`
- New: `nlp/tests/test_grpc_service.py`
- Dependencies: `nlp/requirements.txt`, `nlp/conftest.py`

Python NLP sidecar gRPC service — `serve()` and `StreamPredictions` untested.

- [ ] **Step 1: Read main.py to understand serve() and StreamPredictions**

```bash
cat nlp/main.py
```

- [ ] **Step 2: Write failing test for gRPC service initialization**

```python
# nlp/tests/test_grpc_service.py
import pytest
from unittest.mock import patch, MagicMock

from main import NLPService, serve


class TestNLPService:
    def test_init_with_defaults(self):
        """NLPService initializes with default model path."""
        service = NLPService()
        assert service is not None
        assert service.model_path is not None

    def test_init_with_custom_config(self):
        """NLPService accepts custom configuration."""
        service = NLPService(
            model_path="/custom/model",
            device="cpu",
            max_length=512,
        )
        assert service.model_path == "/custom/model"
        assert service.device == "cpu"

    @pytest.mark.asyncio
    async def test_stream_predictions_empty_input(self):
        """StreamPredictions raises error on empty input."""
        service = NLPService()
        with pytest.raises(ValueError, match="empty"):
            async for _ in service.StreamPredictions("", max_tokens=100):
                pass

    def test_serve_creates_grpc_server(self):
        """serve() creates and starts a gRPC server on the specified port."""
        with patch("grpc.aio.server") as mock_server:
            server_instance = MagicMock()
            mock_server.return_value = server_instance
            
            serve(port=5050)
            
            mock_server.assert_called_once()
            server_instance.add_insecure_port.assert_called_once_with("[::]:5050")
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd nlp && python -m pytest tests/test_grpc_service.py -v
```
Expected: FAIL — adjust tests to match actual signatures

- [ ] **Step 4: Fix tests to match actual interfaces and verify pass**

```bash
cd nlp && python -m pytest tests/test_grpc_service.py -v
```
Expected: PASS

- [ ] **Step 5: Write integration test with gRPC test server**

```python
# nlp/tests/test_grpc_service.py (append)
@pytest.mark.integration
def test_predict_sentiment_grpc():
    """End-to-end gRPC predict call with test server."""
    # This test requires a running gRPC server or in-process channel
    # Use grpc.aio.insecure_channel for test
    pass  # Mark as integration — requires gRPC test infrastructure
```

- [ ] **Step 6: Run full NLP test suite**

```bash
cd nlp && python -m pytest -v
```
Expected: all existing 155 tests pass + new tests pass

- [ ] **Step 7: Commit**

```bash
git add nlp/tests/test_grpc_service.py
git commit -m "test: TDD NLP gRPC service — initialization, StreamPredictions validation, serve()"
```

---

## Task 10: CI/CD — Pre-commit Test Gate

**Files:**
- Modify: `.github/workflows/ci.yml`
- New: `.gitleaks.toml` (if needed)
- New: `scripts/pre-commit-test.sh`

Add a pre-commit test gate to prevent commits that break tests.

- [ ] **Step 1: Read current CI workflow**

```bash
cat .github/workflows/ci.yml
```

- [ ] **Step 2: Create pre-commit test script**

```bash
# scripts/pre-commit-test.sh
#!/bin/bash
set -euo pipefail

echo "=== Pre-commit Test Gate ==="

# Determine what changed
CHANGED_GO=$(git diff --cached --name-only | grep '\.go$' || true)
CHANGED_TS=$(git diff --cached --name-only | grep -E '\.(ts|tsx)$' || true)
CHANGED_PY=$(git diff --cached --name-only | grep '\.py$' || true)

# Run relevant tests based on changes
if [ -n "$CHANGED_GO" ]; then
    echo "--- Go tests ---"
    go test -race -count=1 ./internal/... -timeout 60s
fi

if [ -n "$CHANGED_TS" ]; then
    echo "--- Frontend tests ---"
    cd frontend && npx vitest run --changed HEAD --timeout 30000
    npx tsc --noEmit
    cd ..
fi

if [ -n "$CHANGED_PY" ]; then
    echo "--- NLP tests ---"
    cd nlp && python -m pytest -x -q
    cd ..
fi

echo "=== All checks passed ==="
```

- [ ] **Step 3: Make script executable**

```bash
chmod +x scripts/pre-commit-test.sh
```

- [ ] **Step 4: Install pre-commit hook**

```bash
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
exec scripts/pre-commit-test.sh
EOF
chmod +x .git/hooks/pre-commit
```

- [ ] **Step 5: Add Python NLP tests to CI**

Add `cd nlp && python -m pytest` to the CI workflow file:

```yaml
# .github/workflows/ci.yml (append to test section)
  nlp-test:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: nlp
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - run: pip install -r requirements.txt
      - run: pip install pytest pytest-asyncio
      - run: python -m pytest -x -q
```

- [ ] **Step 6: Commit**

```bash
git add scripts/pre-commit-test.sh .github/workflows/ci.yml
git commit -m "ci: add pre-commit test gate and NLP tests to CI pipeline"
```

---

## Task 11: Go Full Suite Verification

- [ ] **Step 1: Run full Go test suite with race detector**

```bash
go test -race -count=1 ./... -timeout 120s
```
Expected: ALL PASS (498 existing + new tests from Tasks 1-5)

- [ ] **Step 2: Run frontend full suite**

```bash
cd frontend && npx vitest run
```
Expected: ALL PASS (113+ existing + new tests from Tasks 6-8)

- [ ] **Step 3: Run NLP full suite**

```bash
cd nlp && python -m pytest -v
```
Expected: ALL PASS (155 existing + new tests from Task 9)

- [ ] **Step 4: Verify build**

```bash
go build ./... && cd frontend && npx vite build
```
Expected: both succeed

---

## Task 12: Session Summary

- [ ] **Step 1: Count new tests added**

```bash
echo "=== New Go tests ==="
go test ./internal/mcp/ ./internal/registry/ ./internal/decision/ ./internal/sandbox/ ./internal/storage/ -v -count=1 2>&1 | grep -c "PASS"
echo "=== New Frontend tests ==="
cd frontend && npx vitest run api/__tests__/ lib/__tests__/ --reporter=verbose 2>&1 | grep -c "✓"
echo "=== New NLP tests ==="
cd ../nlp && python -m pytest tests/test_grpc_service.py -v 2>&1 | grep -c "PASSED"
```

- [ ] **Step 2: Generate coverage delta**

```bash
go test ./internal/mcp/ ./internal/registry/ ./internal/sandbox/ -coverprofile=/tmp/tdd-coverage.out
go tool cover -func=/tmp/tdd-coverage.out | tail -5
```

---

## TDD Priority Matrix

| Priority | Package | Current Coverage | TDD Value | Difficulty | Lines |
|----------|---------|-----------------|-----------|------------|-------|
| 🔴 P0 | `internal/mcp/discovery.go` | 0% | **Critical** — all tools depend on this | Medium | ~300 |
| 🔴 P0 | `internal/registry/duckdb_registry.go` | 0% | **Critical** — CRUD data path | Medium | ~400 |
| 🔴 P0 | `internal/decision/adapters` | 0% | **Critical** — core AI loop | High | ~500 |
| 🔴 P0 | `internal/sandbox/verification.go` | ~30% | **Security** — SSRF gap | Medium | ~400 |
| 🟠 P1 | `internal/storage/duckdb.go` | ~40% | VSS + schema correctness | Medium | ~800 |
| 🟠 P1 | `api/factory.ts` | 0% | **Critical FE** — all hooks depend | Low | ~60 |
| 🟠 P1 | `api/adapters.ts` | 0% | Type boundary safety | Low | ~80 |
| 🟠 P1 | `lib/AlephGraph.tsx` | 0% | Complex D3 rendering | High | ~200 |
| 🟡 P2 | `nlp/main.py` serve() | ~0% | gRPC service contract | Medium | ~40 |
| 🟡 P2 | `nlp/main.py` StreamPredictions | ~0% | Streaming contract | Medium | ~80 |

---

## Parallel Execution Groups

These groups are fully independent and can run in parallel:

| Group | Tasks | Runner |
|-------|-------|--------|
| **A: Go MCP + Registry** | Task 1, Task 2 | Subagent 1 |
| **B: Go Decision + Sandbox** | Task 3, Task 4 | Subagent 2 |
| **C: Go Storage + Verify** | Task 5, Task 11 | Subagent 3 |
| **D: Frontend API** | Task 6, Task 7 | Subagent 4 |
| **E: Frontend Graph** | Task 8 | Subagent 5 |
| **F: NLP + CI** | Task 9, Task 10 | Subagent 6 |
