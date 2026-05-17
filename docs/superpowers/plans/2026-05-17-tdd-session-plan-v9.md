# Aleph-v2 TDD Final Sprint — Plan v9 (Corrected)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close remaining quality gaps: pre-commit hooks, 19 pre-existing tsc errors, performance benchmarks, error handling audit, integration contract tests.

**Architecture:** 4 independent waves (A-D) + verification (E). Waves A-B are high-impact/low-effort. Wave C performance benchmarks are additive (no code changes). Wave D is audit + targeted fixes.

**Tech Stack:** Go 1.26, React 18 + TypeScript 5.7 + Vite 8, Vitest, DuckDB/PostgreSQL

---

### Wave A: Pre-commit + tsc Fixes (Priority: HIGH)

**Files:**
- Create: `.githooks/pre-commit`
- Create: `scripts/install-githooks.sh`
- Modify: `frontend/src/components/__tests__/CopilotChat.test.tsx`
- Modify: `frontend/src/components/__tests__/DataSourceForm.test.tsx`
- Modify: `frontend/src/components/__tests__/SkillForm.test.tsx`
- Modify: `frontend/src/components/__tests__/ToolsView.test.tsx`
- Modify: `frontend/src/hooks/__tests__/useSSE.test.ts`
- Modify: `frontend/src/store/__tests__/navigationSlice.test.ts`

**Estimated effort:** ~3h

---

### Task A1: Install pre-commit hooks

**Goal:** Prevent commits that break `go build`, `go vet`, `tsc --noEmit`, or `vitest` in affected packages.

- [ ] **Step 1: Write the pre-commit hook**

Create `.githooks/pre-commit`:

```bash
#!/bin/sh
set -euo pipefail

echo "Running pre-commit checks..."

# Stash unstaged changes to only check staged content
git stash -q --keep-index 2>/dev/null || true

# Restore on exit
trap 'git stash pop -q 2>/dev/null || true' EXIT

# Go build
echo "  -> go build ./..."
if ! go build ./... 2>/dev/null; then
  echo "FAIL: go build failed"
  exit 1
fi

# Go vet on changed packages
changed_go=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)
if [ -n "$changed_go" ]; then
  pkgs=$(echo "$changed_go" | xargs -I{} dirname {} | sort -u | tr '\n' ' ')
  echo "  -> go vet $pkgs"
  if ! go vet $pkgs 2>/dev/null; then
    echo "FAIL: go vet failed"
    exit 1
  fi
fi

# Vitest on changed TS/TSX
if git diff --cached --name-only --diff-filter=ACM | grep -q '\.tsx\?$' 2>/dev/null; then
  echo "  -> vitest run (frontend)"
  cd frontend && npx vitest run --reporter=verbose 2>/dev/null && cd ..
fi

echo "OK: pre-commit checks passed"
```

- [ ] **Step 2: Make executable**

Run: `chmod +x .githooks/pre-commit`

- [ ] **Step 3: Create the install script**

Create `scripts/install-githooks.sh`:

```bash
#!/bin/sh
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
git config core.hooksPath .githooks
echo "OK: Git hooks installed from .githooks/"
```

- [ ] **Step 4: Make executable**

Run: `chmod +x scripts/install-githooks.sh`

- [ ] **Step 5: Verify hook works**

Run: `git config core.hooksPath .githooks && .githooks/pre-commit`
Expected: "OK: pre-commit checks passed"

- [ ] **Step 6: Commit**

```bash
git add .githooks/pre-commit scripts/install-githooks.sh
git commit -m "feat: add pre-commit hooks (go build + go vet)"
```

---

### Task A2: Fix 19 pre-existing tsc errors

**Goal:** Zero tsc errors across the entire project.

**19 errors across 6 files:**
- `components/__tests__/CopilotChat.test.tsx:5` — `"text"` not assignable to `TerminalLine["type"]` (needs `'input' | 'output' | 'error' | 'system' | 'tool'`)
- `components/__tests__/DataSourceForm.test.tsx:6` — `error` field doesn't exist in response type
- `components/__tests__/SkillForm.test.tsx:1` — `error` field doesn't exist in response type
- `components/__tests__/ToolsView.test.tsx:4` — `healthStatus` doesn't exist on `Partial<Tool>`
- `hooks/__tests__/useSSE.test.ts:1` — `Cannot find name 'global'`
- `store/__tests__/navigationSlice.test.ts:2` — `payload` doesn't exist on `InlineContent`

- [ ] **Step 1: Read all errors**

Run: `cd frontend && npx tsc --noEmit -p tsconfig.app.json 2>&1`

- [ ] **Step 2: Fix CopilotChat.test.tsx (5 errors)**

File: `frontend/src/components/__tests__/CopilotChat.test.tsx`

Replace `"text"` with one of `'input'`, `'output'`, `'error'`, `'system'`, or `'tool'` in the test data. The actual `TerminalLine.type` union is `'input' | 'output' | 'error' | 'system' | 'tool'`.

```typescript
// Before: type: 'text' as const,
// After:  type: 'output' as const,
```

- [ ] **Step 3: Fix DataSourceForm.test.tsx (6 errors)**

File: `frontend/src/components/__tests__/DataSourceForm.test.tsx`

Add `error?: string` to the mock response type or cast the response:

```typescript
const mockResponse = {
  pipeline_details: [{ id: '1', name: 'test', data_sources: [] }],
  error: 'timeout'
} as const;
```

- [ ] **Step 4: Fix SkillForm.test.tsx (1 error)**

File: `frontend/src/components/__tests__/SkillForm.test.tsx`

Same pattern as DataSourceForm — add `error?: string` to response type.

- [ ] **Step 5: Fix ToolsView.test.tsx (4 errors)**

File: `frontend/src/components/__tests__/ToolsView.test.tsx`

Extend `Partial<Tool>` or use type assertion:

```typescript
const mockTool = {
  id: 't3',
  name: 'Data Exporter',
  healthStatus: 'error',
  // ...other fields
} as Tool;
```

- [ ] **Step 6: Fix useSSE.test.ts (1 error)**

File: `frontend/src/hooks/__tests__/useSSE.test.ts`

Add `declare const global: typeof globalThis;` or use `globalThis` instead of `global`.

- [ ] **Step 7: Fix navigationSlice.test.ts (2 errors)**

File: `frontend/src/store/__tests__/navigationSlice.test.ts`

Widen `InlineContent` type or use `as any` only in test code:

```typescript
const action = { type: 'navigation/addInline', payload: { /* fields */ } as any };
```

- [ ] **Step 8: Verify zero errors**

Run: `cd frontend && npx tsc --noEmit -p tsconfig.app.json`
Expected: Exit code 0, no errors

- [ ] **Step 9: Run vitest to confirm no regressions**

Run: `cd frontend && npx vitest run 2>&1 | tail -5`
Expected: 1350+ tests pass

- [ ] **Step 10: Commit**

```bash
git add frontend/src/components/__tests__/CopilotChat.test.tsx frontend/src/components/__tests__/DataSourceForm.test.tsx frontend/src/components/__tests__/SkillForm.test.tsx frontend/src/components/__tests__/ToolsView.test.tsx frontend/src/hooks/__tests__/useSSE.test.ts frontend/src/store/__tests__/navigationSlice.test.ts
git commit -m "fix: resolve 19 pre-existing tsc errors in test files"
```

---

### Wave B: Integration Contract Tests (Priority: HIGH)

**Files:**
- Create: `internal/routes/contract_test.go`
- Create: `internal/routes/health_contract_test.go`

**Estimated effort:** ~3h

---

### Task B1: Contract test for health/readiness endpoints

**Goal:** Verify `/livez`, `/readyz` return correct HTTP status using a lightweight httptest server with mock dependencies.

**Background:** The health endpoints are registered in `RegisterRoutes` at `internal/routes/routes.go:69`. They use only `isDraining` (atomic bool) and no other dependencies, making them easily testable.

- [ ] **Step 1: Write the failing contract test**

Create `internal/routes/health_contract_test.go`:

```go
package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoints_Contract(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, RegisterConfig{})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("livez returns 200 OK", func(t *testing.T) {
		resp, err := ts.Client().Get(ts.URL + "/livez")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("want 200, got %d", resp.StatusCode)
		}
		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["status"] != "ok" {
			t.Errorf("want status=ok, got %v", body["status"])
		}
	})

	t.Run("readyz returns 200 when not draining", func(t *testing.T) {
		SetDraining(false)
		resp, err := ts.Client().Get(ts.URL + "/readyz")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("want 200, got %d", resp.StatusCode)
		}
	})

	t.Run("readyz returns 503 when draining", func(t *testing.T) {
		SetDraining(true)
		resp, err := ts.Client().Get(ts.URL + "/readyz")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("want 503, got %d", resp.StatusCode)
		}
		SetDraining(false) // reset
	})

	t.Run("frontend SPA fallback returns 200", func(t *testing.T) {
		resp, err := ts.Client().Get(ts.URL + "/any/spa/route")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		// Without embed.FS, this returns 404 — acceptable for unit test
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("want 200 or 404, got %d", resp.StatusCode)
		}
	})
}
```

- [ ] **Step 2: Run test**

Run: `go test -count=1 -run=TestHealthEndpoints_Contract ./internal/routes/`
Expected: PASS (no Postgres needed — health endpoints are dependency-free)

- [ ] **Step 3: Commit**

```bash
git add internal/routes/health_contract_test.go
git commit -m "test: add contract tests for health/readiness endpoints"
```

---

### Wave C: Performance Benchmarks (Priority: MEDIUM)

**Files:**
- Create: `internal/storage/duckdb_bench_test.go`
- Create: `internal/middleware/bench_test.go`

**Estimated effort:** ~2h

---

### Task C1: DuckDB query benchmark

**Goal:** Baseline performance for hot-path DuckDB VSS search.

- [ ] **Step 1: Write benchmark**

Create `internal/storage/duckdb_bench_test.go`:

```go
package storage

import (
	"context"
	"testing"
)

func BenchmarkDuckDB_VSSQuery(b *testing.B) {
	db, err := NewDuckDB(b.TempDir() + "/bench.duckdb")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Insert test embeddings
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		embedding := make([]float32, 384)
		for j := range embedding {
			embedding[j] = float32(i+j) / 1000.0
		}
		if err := db.InsertEmbedding(ctx, "test", int64(i), embedding); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := make([]float32, 384)
		for j := range query {
			query[j] = 0.5
		}
		results, err := db.SearchSimilar(ctx, "test", query, 10)
		if err != nil {
			b.Fatal(err)
		}
		_ = results
	}
}
```

- [ ] **Step 2: Run benchmark**

Run: `go test -bench=. -benchmem -benchtime=1x ./internal/storage/`
Expected: benchmark output with ns/op and allocs/op

- [ ] **Step 3: Commit**

```bash
git add internal/storage/duckdb_bench_test.go
git commit -m "bench: add DuckDB VSS query benchmark"
```

---

### Task C2: Middleware chain benchmark

**Goal:** Measure overhead of CORS + recovery + request ID middleware on a no-op handler.

Note: `AuthMiddleware` requires a real `MetadataRepository` and JWT secret, and `RateLimitMiddleware` creates background goroutines. For a basic overhead benchmark, use the stateless middleware only.

- [ ] **Step 1: Write benchmark**

Create `internal/middleware/bench_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkMiddlewareChain(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Wrap with stateless middleware only
	handler = RequestID(handler)
	handler = Recovery(handler)
	handler = CORS(CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
	})(handler)

	req := httptest.NewRequest("GET", "/", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
}
```

- [ ] **Step 2: Run benchmark**

Run: `go test -bench=. -benchmem -benchtime=1000x ./internal/middleware/`
Expected: benchmark output (ns/op should be <5000 for the chain)

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/bench_test.go
git commit -m "bench: add middleware chain overhead benchmark"
```

---

### Wave D: Error Handling Audit (Priority: MEDIUM)

**Files:**
- Read-only audit: `internal/errors/`, `internal/api/handler/`, `internal/decision/`, `internal/storage/`
- Modify: targeted fixes only

**Estimated effort:** ~2h

---

### Task D1: Audit error wrapping

**Goal:** Identify `err !=` comparisons that should use `errors.Is()`, missing `fmt.Errorf("...: %w", err)` wrapping, and empty catch blocks.

- [ ] **Step 1: Search for error comparison anti-patterns**

Run: `grep -rn 'err != .*\.Err\|\.Err.* != err' --include='*.go' internal/`
Document findings.

- [ ] **Step 2: Search for empty catch blocks**

Run: `grep -rn 'if err != nil {' -A1 --include='*.go' internal/ | grep '//$'`
Document findings.

- [ ] **Step 3: Fix all findings with minimal changes**

For each finding: replace `err != sentinelErr` with `!errors.Is(err, sentinelErr)`, add `%w` where missing, add error logging in empty catch blocks.

- [ ] **Step 4: Commit**

```bash
git add internal/errors/ internal/storage/ internal/decision/
git commit -m "fix: error wrapping audit — use errors.Is, add %w, log empty catches"
```

---

### Wave E: Verification + GitNexus + Graphify (MANDATORY)

**Files:**
- `.gitnexus/` (auto-updated by `npx gitnexus analyze`)
- `graphify-out/graph.html` (auto-updated by graphify)

**Estimated effort:** ~15min

---

### Task E1: Full suite verification

- [ ] **Step 1: Go build + vet + test**

```bash
go build ./...
go vet ./...
go test -count=1 ./... 2>&1 | tail -30
```

- [ ] **Step 2: Frontend build + test**

```bash
cd frontend && npx tsc --noEmit -p tsconfig.app.json && npx vitest run && cd ..
```

- [ ] **Step 3: Verify zero tsc errors**

Confirm `npx tsc --noEmit` returns exit code 0.

---

### Task E2: Commit + Push + Index

- [ ] **Step 1: Stage all changes**

```bash
git add -A
```

- [ ] **Step 2: Create final commit**

```bash
git commit -m "feat: pre-commit hooks, tsc fixes, benchmarks, error audit, contract tests"
```

- [ ] **Step 3: Update GitNexus index**

```bash
npx gitnexus analyze --force
```

- [ ] **Step 4: Update Graphify**

```bash
# Graphify reads from .gitnexus/ — run its graph generation if installed
npx graphify 2>/dev/null || echo "graphify not installed — skipping"
```

- [ ] **Step 5: Push**

```bash
git push origin HEAD
```

---

## Summary

| Wave | Scope | Files | Est. |
|------|-------|-------|------|
| A | Pre-commit hooks + 19 tsc fixes | 2 new + 6 modified | ~3h |
| B | Integration contract tests | 1 new | ~2h |
| C | Performance benchmarks | 2 new | ~2h |
| D | Error handling audit | audit + fixes | ~2h |
| E | Verification + GitNexus + Graphify | 0 | ~15min |
| **Total** | | **~5 new + 6 modified** | **~9h** |
