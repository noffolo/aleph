# TDD Session Plan v8 — Remaining Coverage Gaps (Definitive)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Push internal/app and internal/sandbox to their realistic maximum coverage (app ~25%, sandbox ~65%) by adding tests for: (1) namespace_isolated_default.go stubs, (2) VerifyTool with in-memory DuckDB + mock SandboxManager.

**Architecture:** Two independent waves executed sequentially per subagent-driven-development. Wave A: trivial stub tests (namespace_isolated_default.go). Wave B: integration-style VerifyTool tests using real in-memory DuckDB for MetadataRepository + mock SandboxManager to exercise verification.go lines 87-162. Wave C: verification.

**Tech Stack:** Go, testing, DuckDB in-memory (github.com/marcboeker/go-duckdb), testify/mock.

**Reviews completed:** Metis, Oracle, Momus — feedback incorporated into this definitive version.

---

## Wave A: namespace_isolated_default.go Stub Tests

**Files:**
- Test: `internal/sandbox/namespace_isolated_default_test.go` (CREATE)

**Why:** Behind `//go:build !linux` — 7 functions with 0% coverage on macOS. Pure stubs returning nil, empty, or errors. Simple table-driven test.

**Task details:**
- 7 functions: `IsolateNamespace`, `UnshareNamespace`, `IsolateNetwork`, `ApplyNetworkLimits`, `ApplyCPULimits`, `ApplyMemoryLimits`, `ExecuteIsolated`
- 6 return `nil` or `""`, 1 returns error: `ExecuteIsolated` returns `fmt.Errorf("isolation not supported on this platform")`
- Full source known from `internal/sandbox/namespace_isolated_default.go`

### Test Structure

```go
// SPDX-License-Identifier: MIT
package sandbox

import (
	"context"
	"testing"
	"time"
)

func TestNamespaceIsolationStubs_NonLinux(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "IsolateNamespace",
			fn:   func() error { return IsolateNamespace(0) },
		},
		{
			name: "UnshareNamespace",
			fn:   func() error { return UnshareNamespace(0) },
		},
		{
			name: "IsolateNetwork",
			fn:   func() error { return IsolateNetwork(nil) },
		},
		{
			name: "ApplyNetworkLimits",
			fn:   func() error { return ApplyNetworkLimits("") },
		},
		{
			name: "ApplyCPULimits",
			fn:   func() error { return ApplyCPULimits(nil) },
		},
		{
			name: "ApplyMemoryLimits",
			fn:   func() error { return ApplyMemoryLimits(nil) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err != nil {
				t.Errorf("expected nil, got %v", err)
			}
		})
	}
}

func TestExecuteIsolated_NonLinux(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := ExecuteIsolated(ctx, nil)
	if err == nil {
		t.Fatal("expected error on non-Linux")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}
```

### Steps

- [ ] **Step 1: Write the failing tests** — Create `internal/sandbox/namespace_isolated_default_test.go` with the test code above.
- [ ] **Step 2: Run the tests**

Run: `go test -count=1 ./internal/sandbox/ -run TestNamespaceIsolationStubs_NonLinux -v`
Expected: PASS (functions exist and return nil)

- [ ] **Step 3: Run all sandbox tests to confirm no regression**

Run: `go test -count=1 ./internal/sandbox/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/sandbox/namespace_isolated_default_test.go
git commit -m "test(sandbox): add namespace isolation stub tests for non-Linux platforms"
```

---

## Wave B: VerifyTool with In-Memory DuckDB + Mock SandboxManager

**Files:**
- Modify: `internal/sandbox/verification_test.go` (append new tests)
- Test data: in-memory DuckDB with single-row system_tools

**Why:** Verification.go lines 87-162 (VerifyTool main path) currently at ~9.6% coverage because it requires real MetadataRepository + tool code. Oracle discovered we can use `sql.Open("duckdb", ":memory:")` with `github.com/marcboeker/go-duckdb` blank import to create a real metaRepo. Combined with `mocks.SandboxManager` from `internal/sandbox/mocks/`, we exercise the full VerifyTool path.

**Note from Momus:** The verifyToolCode flow at lines 87-162 includes: nil metaRepo guard, GetToolCode error (table doesn't exist + context manipulation), timeout handling, empty/expired code. Must NOT duplicate existing tests (TestVerifyTool_ExpiredContext, TestVerifyTool_GoNoImport already exist).

### Approach

1. Import DuckDB driver: `import _ "github.com/marcboeker/go-duckdb"` (already established pattern in registry tests)
2. Create in-memory DuckDB: `db, err := sql.Open("duckdb", "")` (empty string = in-memory)
3. Insert a row into system_tools: `INSERT INTO system_tools (id, code) VALUES ($1, $2)`
4. Create `NewMetadataRepository(db)` — this works because MetadataRepository takes `*sql.DB`
5. Use `mocks.SandboxManager{}` for the mock
6. Call `VerifyTool(ctx, metaRepo, "test-tool", "", mockSb)`

### Test Plan

**Test 1: VerifyTool_Success_WithValidCode**
- Insert system_tools row with id="python-3.11" and code="python3"
- Call VerifyTool with id="python-3.11"
- Expect: no error, validator returns Success
- Coverage: verification.go full validation path (check code exists, not expired, valid)

**Test 2: VerifyTool_MissingToolCode**
- Call VerifyTool with id="nonexistent-tool"
- Expect: GetToolCode returns error, function returns wrapped error
- Coverage: lines 87-95 (GetToolCode error path)

### Implementation Details

```go
package sandbox

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/aleph-v2/internal/repository"
	"github.com/aleph-v2/internal/sandbox/mocks"
)

func TestVerifyTool_Success_WithValidCode(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open in-memory duckdb: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS system_tools (id TEXT PRIMARY KEY, code TEXT)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	_, err = db.Exec(`INSERT INTO system_tools (id, code) VALUES ('python-3.11', 'python3')`)
	if err != nil {
		t.Fatalf("failed to insert test tool: %v", err)
	}

	metaRepo, err := repository.NewMetadataRepository(db)
	if err != nil {
		t.Fatalf("failed to create MetadataRepository: %v", err)
	}

	mockSb := &mocks.SandboxManager{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := VerifyTool(ctx, metaRepo, "python-3.11", "", mockSb)
	if err != nil {
		t.Fatalf("VerifyTool failed: %v", err)
	}
	if result.Status != "verified" {
		t.Errorf("expected status 'verified', got %q", result.Status)
	}
}

func TestVerifyTool_MissingToolCode(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open in-memory duckdb: %v", err)
	}
	defer db.Close()

	metaRepo, err := repository.NewMetadataRepository(db)
	if err != nil {
		t.Fatalf("failed to create MetadataRepository: %v", err)
	}

	mockSb := &mocks.SandboxManager{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = VerifyTool(ctx, metaRepo, "nonexistent", "", mockSb)
	if err == nil {
		t.Fatal("expected error for missing tool code, got nil")
	}
}
```

### Steps

- [ ] **Step 1: Verify existing tests compile and pass**

Run: `go test -count=1 ./internal/sandbox/...`
Expected: PASS

- [ ] **Step 2: Append the two new test functions** to `internal/sandbox/verification_test.go`

- [ ] **Step 3: Run the new tests**

Run: `go test -count=1 ./internal/sandbox/ -run 'TestVerifyTool_Success_WithValidCode|TestVerifyTool_MissingToolCode' -v`
Expected: PASS

- [ ] **Step 4: Run all sandbox tests to confirm no regression**

Run: `go test -count=1 ./internal/sandbox/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/sandbox/verification_test.go
git commit -m "test(sandbox): add VerifyTool integration tests with in-memory DuckDB"
```

---

## Wave C: Full Suite Verification

**Files:** None — verification only.

- [ ] **Step 1: Go build and vet**

Run: `go build ./...` && `go vet ./internal/sandbox/...`
Expected: exit 0

- [ ] **Step 2: Run all Go tests**

Run: `go test -count=1 ./internal/sandbox/...`
Expected: PASS

- [ ] **Step 3: Frontend build**

Run: `npx tsc --noEmit` (from frontend/)
Expected: PASS (only pre-existing test file errors)

- [ ] **Step 4: Frontend tests**

Run: `npx vitest run` (from frontend/)
Expected: PASS

- [ ] **Step 5: Generate coverage report**

Run: `go test -coverprofile=coverage.out ./internal/sandbox/... && go tool cover -func=coverage.out | tail -5`
Expected: sandbox coverage > 60%

- [ ] **Step 6: Final summary**

Record final coverage numbers for all packages.
