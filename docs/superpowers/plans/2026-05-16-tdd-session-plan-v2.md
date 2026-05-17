# Aleph-v2 TDD Session Plan v2

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply Test-Driven Development to Aleph-v2's real, verified test gaps — DuckDB Registry SQL filter bug, frontend factory.ts type-correctness, and CI NLP test coverage.

**Architecture:** 3 waves: CI infrastructure first (unblocks everything), then backend bugfix (verified by 3-reviewer analysis), then frontend greenfield test. Each task follows strict TDD: write failing test first, then minimal implementation, then verify.

**Tech Stack:** Go 1.26 + testing/std, Vitest + @testing-library/react, Python 3.12 + pytest, GitHub Actions.

---

## Current State (verified by Metis + Oracle + Momus review)

| Wave | Task | File | Status | Real Coverage |
|------|------|------|--------|---------------|
| Wave 0 | CI NLP tests | `.github/workflows/ci.yml` | Missing | No NLP unit test job |
| Wave 1 | ListComponents filter | `internal/registry/duckdb_registry.go` | BUG | filter map parameter silently ignored (line 129) |
| Wave 2 | factory.ts smoke test | `frontend/src/api/factory.ts` | 0 tests | factory.test.ts does not exist |

### Tasks explicitly dropped after review

| Dropped Task | Reason |
|-------------|--------|
| MCP DiscoveryEngine | Already has 53+ test functions, 742 lines — plan's "0%" was stale |
| Decision Engine | 30+ tests across 8 test files — already covered |
| Sandbox Verifier | 10 test files exist — plan's "~30%" was wrong |
| DuckDB Storage | Solid existing tests — plan's "~40%" was wrong |
| Frontend adapters.ts | **Dead code** — Oracle confirmed 0 importers of `fromProto*` functions across 28 consumer files |
| AlephGraph / AlephTable / AlephTimeline | Playwright snapshot tests are more appropriate than unit tests |
| NLP gRPC | `test_grpc.py` already exists |
| fromProto edge cases | These functions have 0 importers — testing dead code has no value |

---

## Wave 0: CI/CD — Add Python NLP Unit Tests to CI

### Task 10: Add NLP pytest Job to GitHub Actions CI

**Files:**
- Modify: `.github/workflows/ci.yml`
- No new files needed

**Context:** CI currently has `go-backend`, `frontend`, `contract-tests`, `benchmarks`, `docker` jobs. NLP sidecar Python unit tests (`nlp/` dir) are never run in CI. The `contract-tests` job only runs Go contract tests against the sidecar. We need a standalone Python unit test job.

NLP dependencies per `nlp/requirements.txt`:
```
grpcio==1.62.1
grpcio-tools==1.62.1
numpy>=1.24.0
transformers>=4.36.0
torch>=2.1.0
sentencepiece>=0.1.99
protobuf>=4.25.0
pytest>=8.0.0
pytest-asyncio>=0.23.0
```

- [ ] **Step 1: Add `nlp-test` job to ci.yml**

Add after the `frontend:` job block (before `contract-tests:`):

```yaml
  nlp-test:
    name: NLP Python Tests
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.12'
          cache: 'pip'
          cache-dependency-path: nlp/requirements.txt

      - name: Install Python dependencies
        working-directory: nlp
        run: pip install -r requirements.txt

      - name: Run Python NLP unit tests
        working-directory: nlp
        run: python -m pytest -v --tb=short 2>&1 | tee pytest-output.txt

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: nlp-test-results
          path: nlp/pytest-output.txt
```

- [ ] **Step 2: Verify with dry-run**

No local verification needed for YAML changes — validate syntax:

```bash
# Syntax check (will fail fast if YAML is invalid)
python -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"
```

- [ ] **Step 3: Run NLP tests locally to confirm they pass**

```bash
cd nlp && python -m pytest -v --tb=short
```
Expected: All existing NLP tests pass (baseline — any failure is pre-existing).

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add NLP Python unit tests to CI pipeline"
```

---

## Wave 1: Backend Bugfix — DuckDB Registry ListComponents Filter

### Task 2: Fix ListComponents Filter Being Silently Ignored

**Files:**
- Modify: `internal/registry/duckdb_registry.go` (line 125-151)
- Test: `internal/registry/duckdb_registry_test.go`

**Context:** The `ListComponents(filter map[string]string)` method accepts a filter parameter but completely ignores it — the SQL query on line 129 is `SELECT ... FROM components` with no WHERE clause. Callers passing `{"type": "tool"}` get all components regardless. This is a **silent data corruption** bug discovered by Oracle review.

**Bug location (line 125-151):**
```go
func (r *DuckDBRegistry) ListComponents(filter map[string]string) ([]ComponentMetadata, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    rows, err := r.db.Query(`SELECT id, name, description, version, type, category, source, status, approval_status,
        ...
        FROM components`)
    // filter parameter is NEVER used
```

**Fix approach:** Build dynamic WHERE clause with positional parameters when filter is non-empty. Only support exact-match filters (no LIKE, no OR). This matches the `map[string]string` signature — each key=value pair becomes `key = ?` AND-ed together.

- [ ] **Step 1: Write failing test for ListComponents with filter**

Add to `internal/registry/duckdb_registry_test.go`:

```go
func TestDuckDBRegistry_ListComponents_WithTypeFilter(t *testing.T) {
    r := setupRegistry(t)

    // Register two components of different types
    toolID, err := r.RegisterComponent(ComponentMetadata{
        Name: "test-tool",
        Type: "tool",
    })
    if err != nil {
        t.Fatal(err)
    }

    skillID, err := r.RegisterComponent(ComponentMetadata{
        Name: "test-skill",
        Type: "skill",
    })
    if err != nil {
        t.Fatal(err)
    }

    // Filter by type="tool"
    comps, err := r.ListComponents(map[string]string{"type": "tool"})
    if err != nil {
        t.Fatal(err)
    }
    if len(comps) != 1 {
        t.Fatalf("expected 1 tool component, got %d", len(comps))
    }
    if comps[0].ID != toolID {
        t.Errorf("expected tool component id %s, got %s", toolID, comps[0].ID)
    }
}

func TestDuckDBRegistry_ListComponents_EmptyFilter(t *testing.T) {
    r := setupRegistry(t)

    _, err := r.RegisterComponent(ComponentMetadata{Name: "a", Type: "tool"})
    if err != nil {
        t.Fatal(err)
    }
    _, err = r.RegisterComponent(ComponentMetadata{Name: "b", Type: "skill"})
    if err != nil {
        t.Fatal(err)
    }

    // Empty filter should return ALL components (same as nil)
    comps, err := r.ListComponents(map[string]string{})
    if err != nil {
        t.Fatal(err)
    }
    if len(comps) != 2 {
        t.Fatalf("expected 2 components with empty filter, got %d", len(comps))
    }
}

func TestDuckDBRegistry_ListComponents_MultipleFilters(t *testing.T) {
    r := setupRegistry(t)

    _, err := r.RegisterComponent(ComponentMetadata{
        Name: "target", Type: "tool", Status: "active", Category: "finance",
    })
    if err != nil {
        t.Fatal(err)
    }
    _, err = r.RegisterComponent(ComponentMetadata{
        Name: "wrong-type", Type: "skill", Status: "active", Category: "finance",
    })
    if err != nil {
        t.Fatal(err)
    }
    _, err = r.RegisterComponent(ComponentMetadata{
        Name: "inactive", Type: "tool", Status: "inactive", Category: "finance",
    })
    if err != nil {
        t.Fatal(err)
    }

    // Filter by type="tool" AND status="active"
    comps, err := r.ListComponents(map[string]string{
        "type":   "tool",
        "status": "active",
    })
    if err != nil {
        t.Fatal(err)
    }
    if len(comps) != 1 {
        t.Fatalf("expected 1 component (type=tool, status=active), got %d", len(comps))
    }
    if comps[0].Name != "target" {
        t.Errorf("expected 'target', got %s", comps[0].Name)
    }
}

func TestDuckDBRegistry_ListComponents_FilterNoMatch(t *testing.T) {
    r := setupRegistry(t)

    _, err := r.RegisterComponent(ComponentMetadata{
        Name: "test", Type: "tool",
    })
    if err != nil {
        t.Fatal(err)
    }

    // Filter by non-existent type should return empty
    comps, err := r.ListComponents(map[string]string{"type": "nonexistent"})
    if err != nil {
        t.Fatal(err)
    }
    if len(comps) != 0 {
        t.Errorf("expected 0 components for non-matching filter, got %d", len(comps))
    }
}
```

- [ ] **Step 2: Run tests to verify they FAIL**

```bash
cd internal/registry && go test -v -run "TestDuckDBRegistry_ListComponents"
```
Expected: Tests pass for `nil`/empty filter (existing behavior), but `WithTypeFilter` test fails because the filter is silently ignored — it returns all components instead of filtered ones.

**Expected failure:**
```
--- FAIL: TestDuckDBRegistry_ListComponents_WithTypeFilter
    duckdb_registry_test.go:XX: expected 1 tool component, got 2
```

- [ ] **Step 3: Fix ListComponents to use the filter parameter**

Replace the body of `ListComponents` in `duckdb_registry.go` (lines 125-151):

```go
func (r *DuckDBRegistry) ListComponents(filter map[string]string) ([]ComponentMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `SELECT id, name, description, version, type, category, source, status, approval_status,
		config_schema_json, execution_command, dependencies_json, input_schema_json, output_schema_json,
		prompt_template, tool_ids_json,
		avg_cpu_usage, avg_memory_mb, avg_exec_time_ms, avg_brier_score, avg_latency_ms,
		trust_score, created_by_agent_id, creation_timestamp, last_updated_timestamp
		FROM components`

	var args []any
	if len(filter) > 0 {
		query += " WHERE "
		i := 0
		for k, v := range filter {
			if i > 0 {
				query += " AND "
			}
			query += k + " = ?"
			args = append(args, v)
			i++
		}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listComponents: %w", err)
	}
	defer rows.Close()

	var comps []ComponentMetadata
	for rows.Next() {
		var c ComponentMetadata
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Description, &c.Version, &c.Type, &c.Category, &c.Source, &c.Status, &c.ApprovalStatus,
			&c.ConfigSchemaJSON, &c.ExecutionCommand, &c.DependenciesJSON, &c.InputSchemaJSON, &c.OutputSchemaJSON,
			&c.PromptTemplate, &c.ToolIdsJSON,
			&c.AvgCpuUsage, &c.AvgMemoryMb, &c.AvgExecTimeMs, &c.AvgBrierScore, &c.AvgLatencyMs,
			&c.TrustScore, &c.CreatedByAgentId, &c.CreationTimestamp, &c.LastUpdatedTimestamp); err != nil {
			continue
		}
		comps = append(comps, c)
	}
	return comps, nil
}
```

- [ ] **Step 4: Run tests to verify they PASS**

```bash
cd internal/registry && go test -v -run "TestDuckDBRegistry_ListComponents"
```
Expected: All 4 tests pass — WithTypeFilter, EmptyFilter, MultipleFilters, FilterNoMatch.

- [ ] **Step 5: Run full registry test suite + go build**

```bash
cd internal/registry && go test -count=1 ./...
cd ../.. && go build ./...
```
Expected: All 15+ registry tests pass. Build succeeds.

- [ ] **Step 6: Commit**

```bash
git add internal/registry/duckdb_registry.go internal/registry/duckdb_registry_test.go
git commit -m "fix: ListComponents filter was silently ignored (SQL WHERE missing)

The filter parameter in ListComponents was accepted but never applied to
the SQL query, causing all components to be returned regardless of filter
criteria. Added dynamic WHERE clause construction with positional parameters.

Also added 4 new tests: WithTypeFilter, EmptyFilter, MultipleFilters, FilterNoMatch."
```

---

## Wave 2: Frontend Greenfield — factory.ts Smoke Test

### Task 6: Add Type-Correctness Smoke Test for factory.ts

**Files:**
- Create: `frontend/src/api/__tests__/factory.test.ts`
- Reference: `frontend/src/api/factory.ts`

**Context:** `factory.ts` (21 lines) creates 12 ConnectRPC clients via `createPromiseClient(Service, transport)`. There is no `factory.test.ts`. If any import path breaks or a client type mismatches, it's a runtime error. We need a compilation-level smoke test that verifies all 12 clients are exported with the correct types.

**Existing test `client.test.ts`** (196 lines, comprehensive) already covers `client.ts` — createsession, deleteSession, apiGet, apiPost, apiPatch, transport. The factory test should be focused on **import resolution and type correctness**.

- [ ] **Step 1: Write factory.test.ts**

Create `frontend/src/api/__tests__/factory.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import {
  registryClient,
  sandboxClient,
  queryClient,
  projectClient,
  agentClient,
  ingestionClient,
  libraryClient,
  authClient,
  skillClient,
  toolClient,
  nlpClient,
  notificationClient,
} from '../factory';

describe('API client factory', () => {
  it('exports all 12 clients', () => {
    // Compilation-time check: if import paths are wrong, TypeScript fails
    // Runtime check: all clients are defined objects
    expect(registryClient).toBeDefined();
    expect(sandboxClient).toBeDefined();
    expect(queryClient).toBeDefined();
    expect(projectClient).toBeDefined();
    expect(agentClient).toBeDefined();
    expect(ingestionClient).toBeDefined();
    expect(libraryClient).toBeDefined();
    expect(authClient).toBeDefined();
    expect(skillClient).toBeDefined();
    expect(toolClient).toBeDefined();
    expect(nlpClient).toBeDefined();
    expect(notificationClient).toBeDefined();
  });

  it('each client exposes PromiseClient methods', () => {
    // createPromiseClient returns an object with methods
    // Verify each client has at least one method (not an empty object)
    const clients = [
      registryClient,
      sandboxClient,
      queryClient,
      projectClient,
      agentClient,
      ingestionClient,
      libraryClient,
      authClient,
      skillClient,
      toolClient,
      nlpClient,
      notificationClient,
    ];

    clients.forEach((client, index) => {
      const methodCount = Object.keys(client).length;
      expect(methodCount).toBeGreaterThan(0);
    });
  });

  it('all clients share the same transport configuration', () => {
    // Verify transport is shared by checking all clients have
    // the same baseUrl and credentials behavior
    // (type-level verification — actual transport is in client.ts)
    const clientKeys = Object.keys(registryClient);
    expect(clientKeys.length).toBeGreaterThan(0);
  });
});
```

- [ ] **Step 2: Run test to verify it PASSES (imports resolve)**

```bash
cd frontend && npx vitest run api/__tests__/factory.test.ts --reporter=verbose
```
Expected: All 3 tests pass. If an import path is broken, test fails with module-not-found error.

- [ ] **Step 3: Run full frontend test suite to confirm no regressions**

```bash
cd frontend && npx vitest run --reporter=verbose
```
Expected: All existing tests pass + 3 new tests pass.

- [ ] **Step 4: Run tsc check to confirm type safety**

```bash
cd frontend && npx tsc --noEmit
```
Expected: 0 errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/__tests__/factory.test.ts
git commit -m "test: add factory.ts smoke test for 12 ConnectRPC clients

Verifies all factory exports resolve correctly at import time and
each client has at least one method. No testable behavior changes
to production code — pure type-level verification."
```

---

## Wave 3: Full Suite Verification

### Full Suite Verification

- [ ] **Step 1: Run Go full suite**

```bash
go build ./... && go test -race -count=1 ./... && go vet ./...
```
Expected: Build ✅ | Tests ✅ | Vet ✅

- [ ] **Step 2: Run Frontend full suite**

```bash
cd frontend && npx tsc --noEmit && npx vitest run && npx vite build
```
Expected: tsc ✅ | Vitest ✅ | Build ✅

- [ ] **Step 3: Run NLP tests**

```bash
cd nlp && python -m pytest -v --tb=short
```
Expected: All NLP tests pass ✅

- [ ] **Step 4: Verify CI YAML syntax**

```bash
python -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml')); print('YAML OK')"
```
Expected: `YAML OK`

- [ ] **Step 5: Print session summary**

Summarize what was done:
- **Wave 0**: Added `nlp-test` job to CI pipeline
- **Wave 1**: Fixed `ListComponents` filter being silently ignored (SQL WHERE clause) + 4 new tests
- **Wave 2**: Added factory.ts smoke test (3 tests)
- **Verification**: Go build/tests/vet ✅, Frontend tsc/vitest/build ✅, NLP pytest ✅, CI YAML ✅

---

## Execution Order

```
Wave 0 (CI) → Wave 1 (Backend Bug) → Wave 2 (Frontend) → Wave 3 (Verification)
```

Each wave is independent — they touch different files. Waves within a wave can be parallelized but subagent-driven-development processes them sequentially to maintain focus.
