# TDD Session Plan v3 — Aleph-v2 Comprehensive Coverage

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close ALL P0/P1 coverage gaps identified by GitNexus deterministic analysis (22,716 symbols, 56,148 edges, 2,752 test functions). Target: 100% of packages have at least a _test.go file, zero packages with <50% symbol coverage.

**Architecture:** 5 vertical waves — Go P0 packages (gnn + workflow), Go P1-P2 gaps (probe.go + humanecosystems), Frontend priority (adapters dead code removal + App.tsx smoke test + CopilotChat), integration verification, then an Oracle eval pass to identify remaining gaps.

**Tech Stack:** Go 1.26 (testing + testify), React 18 + TypeScript (vitest + @testing-library/react), Python 3.12 (pytest), DuckDB, PostgreSQL.

**GitNexus baseline (16 May 2026):** 22,716 symbols, 56,148 edges, 799 communities, 300 execution flows, 2,752 test functions.

---

## Baseline: Already Done (skip / verify only)

| Area | Status | Evidence |
|------|--------|----------|
| CI NLP test job (ci.yml:130-158) | ✅ DONE | `.github/workflows/ci.yml` lines 130-158 |
| DuckDB Registry ListComponents filter bug | ✅ DONE | `duckdb_registry.go` + `duckdb_registry_test.go` (16 test funcs) |
| Frontend API factory.ts smoke test | ✅ DONE | `factory.test.ts` (3 tests, 3/3 pass) |
| MCP DiscoveryEngine | ✅ EXISTING | `discovery_test.go` + `discovery_supplement_test.go` (742 lines, 53+ tests) |
| Decision Engine | ✅ EXISTING | 8 test files, ~30 test functions, 160% symbol coverage |
| Sandbox/Verifier | ✅ EXISTING | 10 test files, solid coverage |
| DuckDB Storage (VSS/Schema) | ✅ EXISTING | 285% symbol coverage |
| NLP gRPC service | ✅ EXISTING | `test_grpc.py` exists, 155 tests total across NLP |
| Packages with >100% coverage | ✅ EXISTING | storage (285%), mcp (255%), llm (221%), handler (195%), registry (189%), repository (186%), decision (160%) |

---

## Coverage Gap Summary (GitNexus-Determined)

### Go P0 — No _test.go file OR <50% symbol coverage

| Package | Implemented Symbols | Test Coverage | Priority | Gap |
|---------|-------------------|---------------|----------|-----|
| `internal/engine/gnn/` | 614 | 379 (62%) — **NO _test.go** | **P0** | Entire package untested at file level |
| `internal/engine/workflow/` | 206 | 87 (42%) | **P0** | _test.go exists but covers <50% |

### Go P1-P2 — Low coverage areas

| Package | Lines | Coverage Estimate | Priority | Gap |
|---------|-------|-------------------|----------|-----|
| `internal/ingestion/probe.go` | 558 | ~0% | **P1** | No test function references |
| `internal/tools/humanecosystems/` | ~350 | ~55% | **P2** | Missing integration tests |

### Frontend — Files without ANY test

| File | Lines | Risk | Priority |
|------|-------|------|----------|
| `frontend/src/api/adapters.ts` | 44 | **DEAD CODE** (0 importers across 28 factory.ts consumers) | **P0** |
| `frontend/src/App.tsx` | ~200 | Routes + SlideOver switch (critical) | **P1** |
| `frontend/src/hooks/useSSE.ts` | ~60 | SSE connection logic | **P1** |
| `frontend/src/hooks/useAppActions.ts` | ~80 | Action dispatch | **P1** |
| `frontend/src/hooks/useServices.ts` | ~40 | Service init | **P1** |
| `frontend/src/components/CopilotChat.tsx` | ~200 | Core chat UI | **P1** |
| `frontend/src/components/CommandPalette.tsx` | ~312 | Command interface | **P1** |
| `frontend/src/components/SlideOver*.tsx` (14 forms) | ~50-150 each | All form CRUD | **P1** |
| `frontend/src/components/ui/*.tsx` (12 primitives) | ~20-80 each | Button, Input, Modal, etc. | **P2** |
| `frontend/src/scenes/*.tsx` (6 scenes) | ~50-200 each | View pages | **P2** |

### Python NLP

| File | Lines | Coverage | Gap |
|------|-------|----------|-----|
| `nlp/main.py` | ~400 | `serve()` and `StreamPredictions` at 0% | Per prior analysis — low priority |
| `nlp/test_grpc.py` | ~200 | Exists but missing `TestNLPIntegration` for end-to-end | Low priority |

---

## Plan Structure

5 waves, sequential (subagent-driven-development — many have interdependencies):

1. **Wave A (Go P0):** gnn + workflow packages
2. **Wave B (Go P1-P2):** probe.go + humanecosystems
3. **Wave C (Frontend):** adapters dead code removal + critical UI tests
4. **Wave D (Verification):** Full suite + coverage report
5. **Wave E (Oracle Eval):** Re-analyze with GitNexus, find remaining gaps

**Work strategy decision (via work-strategy skill):**
- **Complexity:** high (3 subsystems, multiple languages, risk of regression on gnn)
- **Risk:** medium-high (gnn is touched by multiple consumers)
- **Dependencies:** moderate (Waves A→B are independent of C, but A and B must precede D-E)
- **Strategy:** `subagent-driven-development` within each wave, but Waves A+B can run in `orchestrate` since they're independent Go packages
- **Mitigation:** Run `gitnexus_impact` before any edit, `gitnexus_detect_changes` after each task

---

## Wave A: Go P0 — gnn + workflow TDD

### Task A1: gnn package — Create _test.go with core tests

**GitNexus impact check:** MUST run `gitnexus_impact({target: "gnn", direction: "downstream"})` before editing.

**Files:**
- Create: `internal/engine/gnn/gnn_test.go`
- No modification to existing gnn files (tests only)

**Context:** `internal/engine/gnn/` has 614 implemented symbols with 379 (62%) having test coverage via their callers, but ZERO `_test.go` files. This means there is no file-level test package — tests exist only through other packages that consume gnn. Core functions like Graph construction, node embedding, similarity search, and community detection have no direct unit tests.

**Test plan (4 test functions):**

- [ ] **Step 1A: Impact analysis**
  Run: `npx gitnexus impact --target gnn --direction downstream`
  Review: list of direct callers, decide if adding tests changes anything.

- [ ] **Step 1B: Write TestGraphConstruction**

  ```go
  package gnn

  import (
      "testing"
      "github.com/stretchr/testify/assert"
  )

  func TestGraphConstruction(t *testing.T) {
      g := NewGraph()
      assert.NotNil(t, g)
      assert.Equal(t, 0, g.NumNodes())

      n1 := g.AddNode("node1", []float32{0.1, 0.2, 0.3})
      n2 := g.AddNode("node2", []float32{0.4, 0.5, 0.6})
      assert.Equal(t, 2, g.NumNodes())
      assert.Equal(t, "node1", g.GetNode(n1).ID)
      assert.Equal(t, "node2", g.GetNode(n2).ID)

      err := g.AddEdge(n1, n2, 0.85)
      assert.NoError(t, err)
      assert.Equal(t, 1, g.NumEdges())

      neighbors := g.GetNeighbors(n1)
      assert.Contains(t, neighbors, n2)
  }
  ```

- [ ] **Step 1C: Write TestNodeEmbedding**

  ```go
  func TestNodeEmbedding(t *testing.T) {
      g := NewGraph()
      n1 := g.AddNode("a", []float32{1.0, 0.0})
      n2 := g.AddNode("b", []float32{0.0, 1.0})
      g.AddEdge(n1, n2, 1.0)

      emb := NewEmbedding(g)
      vec, err := emb.Embed(n1)
      assert.NoError(t, err)
      assert.NotNil(t, vec)
      assert.Greater(t, len(vec), 0)

      vec2, err := emb.Embed(n2)
      assert.NoError(t, err)
      // Similar nodes should have similar embeddings
      sim := cosineSimilarity(vec, vec2)
      assert.Greater(t, sim, 0.5)
  }
  ```

- [ ] **Step 1D: Write TestSimilaritySearch**

  ```go
  func TestSimilaritySearch(t *testing.T) {
      g := NewGraph()
      nodes := []string{"target", "similar", "different"}
      for _, id := range nodes {
          g.AddNode(id, randomVec(10))
      }
      g.AddEdge(0, 1, 0.95) // target <-> similar: strong edge

      results := g.SearchSimilar(g.GetNode("target"), 2)
      assert.Len(t, results, 2)
      assert.Equal(t, "similar", results[0].ID)
  }
  ```

- [ ] **Step 1E: Write TestCommunityDetection**

  ```go
  func TestCommunityDetection(t *testing.T) {
      g := NewGraph()
      // Community A: 3 tightly-connected nodes
      a1 := g.AddNode("a1", nil)
      a2 := g.AddNode("a2", nil)
      a3 := g.AddNode("a3", nil)
      g.AddEdge(a1, a2, 1.0)
      g.AddEdge(a2, a3, 1.0)
      g.AddEdge(a1, a3, 1.0)
      // Community B: 2 tightly-connected nodes, weakly linked to A
      b1 := g.AddNode("b1", nil)
      b2 := g.AddNode("b2", nil)
      g.AddEdge(b1, b2, 1.0)
      g.AddEdge(a3, b1, 0.1) // weak bridge

      communities := g.DetectCommunities()
      assert.Len(t, communities, 2)
      // a1, a2, a3 should be in one community
      // b1, b2 should be in another
  }
  ```

- [ ] **Step 2: Run test to verify compilation**
  Run: `cd internal/engine/gnn && go test -count=1 -run TestGraphConstruction ./...`
  Expected: PASS (or compiler error indicating correct function signatures needed — implement adapters)

- [ ] **Step 3: Run all gnn tests**
  Run: `go test -count=1 ./internal/engine/gnn/...`
  Expected: `ok` (4/4 pass)

- [ ] **Step 4: Commit**
  ```bash
  git add internal/engine/gnn/gnn_test.go
  git commit -m "test(gnn): add core unit tests for Graph, Embedding, SimilaritySearch, CommunityDetection"
  ```

---

### Task A2: workflow package — Raise coverage to >80%

**GitNexus impact check:** MUST run `gitnexus_impact({target: "workflow", direction: "downstream"})` before editing.

**Files:**
- Create or modify: `internal/engine/workflow/workflow_test.go`
- No modification to existing workflow source files (tests only)

**Context:** `internal/engine/workflow/` has 206 symbols with only 87 (42%) tested. An existing `_test.go` exists but only covers some paths. Missing coverage likely includes: step execution, conditional branching, error propagation, retry logic, and parallel execution.

**Test plan (3 test functions):**

- [ ] **Step 1A: Impact analysis**
  Run: `npx gitnexus impact --target workflow --direction downstream`
  Review: affected processes.

- [ ] **Step 1B: Write TestStepExecution**

  ```go
  package workflow

  import (
      "testing"
      "github.com/stretchr/testify/assert"
      "github.com/stretchr/testify/require"
  )

  func TestStepExecution(t *testing.T) {
      // Create a simple workflow with 3 steps: A → B → C
      wf := NewWorkflow("test")
      stepA := wf.AddStep("stepA", func(ctx Context) error {
          ctx.Set("result", "done")
          return nil
      })
      stepB := wf.AddStep("stepB", func(ctx Context) error {
          val, _ := ctx.Get("result")
          if val != "done" {
              return ErrMissingDependency
          }
          return nil
      })
      stepC := wf.AddStep("stepC", func(ctx Context) error {
          return nil
      })
      wf.DependsOn(stepB, stepA)
      wf.DependsOn(stepC, stepB)

      err := wf.Execute(Context{})
      assert.NoError(t, err)
      assert.Equal(t, StatusCompleted, stepA.Status())
      assert.Equal(t, StatusCompleted, stepB.Status())
      assert.Equal(t, StatusCompleted, stepC.Status())
  }
  ```

- [ ] **Step 1C: Write TestConditionalBranching**

  ```go
  func TestConditionalBranching(t *testing.T) {
      wf := NewWorkflow("conditional")
      cond := wf.AddConditional("check", func(ctx Context) bool {
          val, _ := ctx.Get("value")
          return val == "branch_a"
      })
      branchA := wf.AddStep("branchA", func(ctx Context) error { return nil })
      branchB := wf.AddStep("branchB", func(ctx Context) error { return nil })
      wf.Branch(cond, branchA, branchB)

      // Test branch A
      ctx := Context{values: map[string]any{"value": "branch_a"}}
      err := wf.Execute(ctx)
      assert.NoError(t, err)
      assert.Equal(t, StatusCompleted, branchA.Status())
      assert.Equal(t, StatusSkipped, branchB.Status())

      // Test branch B
      wf.Reset()
      ctx = Context{values: map[string]any{"value": "other"}}
      err = wf.Execute(ctx)
      assert.NoError(t, err)
      assert.Equal(t, StatusSkipped, branchA.Status())
      assert.Equal(t, StatusCompleted, branchB.Status())
  }
  ```

- [ ] **Step 1D: Write TestErrorPropagation**

  ```go
  func TestErrorPropagation(t *testing.T) {
      wf := NewWorkflow("error-test")
      failing := wf.AddStep("fail", func(ctx Context) error {
          return assert.AnError
      })
      dependent := wf.AddStep("dependent", func(ctx Context) error { return nil })
      wf.DependsOn(dependent, failing)

      err := wf.Execute(Context{})
      assert.Error(t, err)
      assert.Equal(t, StatusFailed, failing.Status())
      assert.Equal(t, StatusSkipped, dependent.Status())
  }
  ```

- [ ] **Step 2: Run tests**
  Run: `go test -count=1 -run 'TestStepExecution|TestConditionalBranching|TestErrorPropagation' ./internal/engine/workflow/...`
  Expected: `ok` (3/3 pass)

- [ ] **Step 3: Run full workflow test suite**
  Run: `go test -count=1 ./internal/engine/workflow/...`
  Expected: all existing + new tests pass

- [ ] **Step 4: Commit**
  ```bash
  git add internal/engine/workflow/workflow_test.go
  git commit -m "test(workflow): add step execution, conditional branching, and error propagation tests"
  ```

---

## Wave B: Go P1-P2 — probe.go + humanecosystems

### Task B1: ingestion/probe.go unit tests

**GitNexus impact check:** Run `gitnexus_impact({target: "Probe", direction: "downstream"})` before editing.

**Files:**
- Create: `internal/ingestion/probe_test.go`
- No modification to `probe.go` (tests only)

**Context:** `internal/ingestion/probe.go` is 558 lines with 0% Go test coverage. Contains schema detection, data type inference, and column profiling logic. This is a data-quality critical path but low refactoring risk since we're adding tests only.

**Test plan (3 test functions):**

- [ ] **Step 1: Write TestSchemaDetection**
  Write test covering: empty input, single column, mixed types, nested structures, null handling — following patterns from `internal/ingestion/` existing tests.

- [ ] **Step 2: Write TestDataTypeInference**
  Test covering: integer detection, float detection, string truncation, date formats, boolean parsing.

- [ ] **Step 3: Write TestColumnProfiling**
  Test covering: null count, distinct count, min/max values, type distribution.

- [ ] **Step 4: Run tests + commit**
  Run: `go test -count=1 ./internal/ingestion/...`
  Commit: `test(ingestion): add probe.go unit tests for schema detection, type inference, and profiling`

---

### Task B2: humanecosystems package — integration test coverage

**Files:**
- Create: `internal/tools/humanecosystems/humanecosystems_test.go`
- No modification to existing source

**Context:** ~55% symbol coverage. Missing tests for: DuckDB integration layer, suggestion ranking, ecosystem simulation, and composite metric calculations.

**Test plan (2 test functions):**

- [ ] **Step 1: Write TestEcosystemSimulation**
  Test: population dynamics, resource allocation, interaction graphs with mock data.

- [ ] **Step 2: Write TestSuggestionRanking**
  Test: ranking with various weights, tie-breaking, empty input handling, max results boundary.

- [ ] **Step 3: Run tests + commit**
  Run: `go test -count=1 ./internal/tools/humanecosystems/...`
  Commit: `test(humanecosystems): add ecosystem simulation and suggestion ranking tests`

---

## Wave C: Frontend — Adaptors + Critical UI Tests

### Task C1: Remove adapters.ts dead code

**Files:**
- Remove: `frontend/src/api/adapters.ts`
- Update: Any imports-of-adapters (verify: 0 importers, but check barrel files)

**Context:** Oracle confirmed adapters.ts `fromProto*` functions have 0 importers across 28 consumers that import factory.ts directly. This is dead code.

- [ ] **Step 1: Verify 0 importers**
  Run: `grep -r "adapters" frontend/src/ --include="*.ts" --include="*.tsx" | grep -v ".test." | grep -v "adapters.ts"`
  Expected: empty (no imports)

- [ ] **Step 2: Check barrel/index exports**
  Check `frontend/src/api/index.ts` or `frontend/src/api/barrel.ts` for re-export of adapters.

- [ ] **Step 3: Remove file**
  Run: `git rm frontend/src/api/adapters.ts`
  Run: `rm frontend/src/api/adapters.ts`

- [ ] **Step 4: Verify build**
  Run: `cd frontend && npx tsc --noEmit`
  Expected: 0 errors (pre-existing 18 may remain, no NEW errors)
  Run: `cd frontend && npx vitest run`
  Expected: all 1358+ tests pass

- [ ] **Step 5: Commit**
  ```bash
  git commit -m "refactor(api): remove dead code adapters.ts — fromProto* functions had 0 importers across 28 consumers"
  ```

---

### Task C2: App.tsx smoke test

**Files:**
- Create: `frontend/src/__tests__/App.test.tsx`
- No modification to App.tsx

**Context:** App.tsx (~200 lines) is the root component with React.lazy imports, SlideOverContent switch/case, and Layout wrapper. No tests exist. At minimum, verify it renders without crashing.

- [ ] **Step 1: Write render smoke test**

  ```typescript
  import { describe, it, expect } from 'vitest';
  import { render, screen } from '@testing-library/react';
  import App from '../App';

  describe('App', () => {
    it('renders without crashing', () => {
      // Minimal render test — App has lazy-loaded children that need Suspense
      const { container } = render(<App />);
      expect(container).toBeTruthy();
    });

    it('renders the main layout', () => {
      render(<App />);
      // The app should render at least the layout shell
      expect(screen.getByTestId('app-shell')).toBeTruthy();
    });
  });
  ```

  Note: If App.tsx has side-effectful imports (store, router), tests may need mocking. Adapt as needed.

- [ ] **Step 2: Run test**
  Run: `cd frontend && npx vitest run src/__tests__/App.test.tsx`
  Expected: PASS (may need mock setup)

- [ ] **Step 3: Commit**
  ```bash
  git add frontend/src/__tests__/App.test.tsx
  git commit -m "test(frontend): add App.tsx smoke render test"
  ```

---

### Task C3: CopilotChat component test

**Files:**
- Create: `frontend/src/components/__tests__/CopilotChat.test.tsx`
- No modification to CopilotChat.tsx

**Context:** CopilotChat.tsx (~200 lines) is the core chat UI component. No tests exist.

- [ ] **Step 1: Write render + interaction test**
  Test: renders chat container, renders message list, renders input field, submit button disabled when empty.

- [ ] **Step 2: Run test**
  Run: `cd frontend && npx vitest run src/components/__tests__/CopilotChat.test.tsx`
  Expected: PASS

- [ ] **Step 3: Commit**
  ```bash
  git add frontend/src/components/__tests__/CopilotChat.test.tsx
  git commit -m "test(frontend): add CopilotChat render and interaction tests"
  ```

---

### Task C4: useSSE hook test

**Files:**
- Create: `frontend/src/hooks/__tests__/useSSE.test.ts`
- No modification to useSSE.ts

- [ ] **Step 1: Write hook test**
  Test: connect/disconnect lifecycle, message parsing, error handling, reconnection logic.

- [ ] **Step 2: Run + commit**
  Run: `cd frontend && npx vitest run src/hooks/__tests__/useSSE.test.ts`
  Commit: `test(frontend): add useSSE hook unit tests`

---

## Wave D: Full Suite Verification

- [ ] **Step 1: Go build + vet**
  ```bash
  go build ./... && go vet ./... && echo "✅ Go build+vet OK"
  ```

- [ ] **Step 2: Go test**
  ```bash
  go test -count=1 ./... 2>&1 | tail -20
  ```

- [ ] **Step 3: Frontend type check**
  ```bash
  cd frontend && npx tsc --noEmit 2>&1
  ```

- [ ] **Step 4: Frontend tests**
  ```bash
  cd frontend && npx vitest run 2>&1 | tail -10
  ```

- [ ] **Step 5: Python NLP tests**
  ```bash
  cd nlp && python -m pytest tests/ -v 2>&1 | tail -20
  ```

- [ ] **Step 6: GitNexus detect_changes**
  ```bash
  npx gitnexus detect-changes
  ```
  Verify: only expected symbols affected.

---

## Wave E: Oracle Evaluation Pass

- [ ] **Step 1: Run GitNexus reanalysis**
  ```bash
  npx gitnexus analyze --force
  ```

- [ ] **Step 2: Query remaining coverage gaps**
  Use GitNexus to find packages still below 50% test coverage.

- [ ] **Step 3: Generate coverage summary report**
  Produce structured report at `docs/superpowers/reports/2026-05-16-coverage-report.md`.

- [ ] **Step 4: Iterate if gaps remain**
  Add tasks for any remaining P0/P1 gaps.

---

## Execution Strategy (via work-strategy skill)

**Work strategy assessment:**

| Dimension | Rating | Rationale |
|-----------|--------|-----------|
| Complexity | **high** | 3 languages, 2 frameworks, DuckDB dependency |
| Risk | **medium** | gnn has downstream consumers; adapters removal is zero-risk (dead code) |
| Dependencies | **moderate** | Waves A+B are independent of C; D+E depend on all prior waves |
| Strategy | **hybrid** | Waves A-C: `subagent-driven-development` (sequential, review-gated). Waves A+B can parallelize within each wave if independent files |
| Mitigation | GitNexus | `gitnexus_impact` before edit, `gitnexus_detect_changes` after each task |

**Execution flow:**
```
Wave A (gnn + workflow) — subagent-driven-development, sequential
  → A1 gnn tests → review → A2 workflow tests → review
Wave B (probe.go + humanecosystems) — subagent-driven-development, sequential  
  → B1 probe tests → review → B2 humanecosystems tests → review
Wave C (Frontend) — subagent-driven-development, sequential
  → C1 remove adapters → review → C2 App.tsx → review → C3 CopilotChat → review → C4 useSSE → review
Wave D (Verification) — direct execution
  → Full suite build+test+lint
Wave E (Oracle Eval) — oracle consultation
  → Re-analyze with GitNexus → report → iterate if gaps remain
```
