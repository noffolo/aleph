# TDD Session Plan v4 — Aleph-v2 Corrected Coverage

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Close verified P0/P1 coverage gaps based on GitNexus + actual source inspection. Execute only targets confirmed by Metis/Oracle/Momus review cycle.

**Previous plan (v3) was reviewed by Metis/Oracle/Momus and found to have hallucinated package paths and APIs. This v4 is corrected against real source code.**

**Corrected baseline (16 May 2026):**

| Claim in v3 | Reality (verified) | Action |
|---|---|---|
| gnn at `internal/engine/gnn/` NO _test.go | `internal/gnn/`, **14 tests** (379 lines) | Drop — already covered |
| workflow at `internal/engine/workflow/` | `internal/workflow/`, **4 tests** (87 lines) | Augment — real gaps: multi-step, cancellation |
| probe.go 0% coverage | **5+ tests** in co-existing files | Target pure functions only |
| humanecosystems ~55% | **39 tests** across 3 files | Drop — already covered |
| adapters.ts 0 importers | Verified **0 importers** | ✅ Keep — safe removal |
| CopilotChat ~200 lines | **57 lines** actual | Keep — quick win |
| useSSE — not in v2 scope | **341 lines**, SSE parsing logic | Keep — high value |

---

## Work Strategy (via work-strategy skill)

| Dimension | Rating | Rationale |
|-----------|--------|-----------|
| Complexity | **medium** | Corrected paths resolve hallucinations; 1-2 files per task |
| Risk | **low** | Most work is test-only; adapters removal verified zero risk |
| Dependencies | **weak** | C1 (adapters) independent; A1-B1 independent; C2-C4 independent |
| Strategy | **`subagent-driven-development`** | Each task has clear boundaries; spec review gates per task |

---

## Wave A: Go — Real Gaps

### Task A1: Workflow — Multi-step + Cancellation Tests

**Files:**
- Modify: `internal/workflow/engine_test.go` (add test functions)

**Context:** `internal/workflow/engine.go` has 4 existing tests. Missing: multi-step input chaining, context cancellation mid-execution, orphan status query, concurrent execution.

**Real API:**
```go
eng := NewEngine()
eng.RegisterStep("step1", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    return map[string]interface{}{"result": "done"}, nil
})
err := eng.Execute(ctx, &Workflow{
    ID: NewID(),
    Steps: []Step{{Name: "step1", Fn: stepFn}},
})
status, err := eng.GetStatus(id)
```

- [ ] **Step 1: Write TestMultiStepInputChaining**

  ```go
  func TestMultiStepInputChaining(t *testing.T) {
      eng := NewEngine()
      eng.RegisterStep("generate", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
          return map[string]interface{}{"value": 42}, nil
      })
      eng.RegisterStep("transform", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
          prior, ok := input["generate"]
          if !ok {
              return nil, fmt.Errorf("missing prior step output")
          }
          priorMap, ok := prior.(map[string]interface{})
          if !ok {
              return nil, fmt.Errorf("prior output not a map")
          }
          val := priorMap["value"].(int)
          return map[string]interface{}{"doubled": val * 2}, nil
      })

      ctx := context.Background()
      w := &Workflow{
          ID:    NewID(),
          Steps: []Step{{Name: "generate"}, {Name: "transform"}},
      }
      err := eng.Execute(ctx, w)
      assert.NoError(t, err)
      assert.Equal(t, StatusCompleted, w.Status)
      assert.Len(t, w.Result, 2)
      // transform received generate's output
      transformOutput := w.Result[1].Output
      assert.Equal(t, 84, transformOutput["doubled"])
  }
  ```

- [ ] **Step 2: Write TestContextCancellation**

  ```go
  func TestContextCancellation(t *testing.T) {
      eng := NewEngine()
      eng.RegisterStep("slow", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
          select {
          case <-ctx.Done():
              return nil, ctx.Err()
          case <-time.After(10 * time.Second):
              return map[string]interface{}{"done": true}, nil
          }
      })

      ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
      defer cancel()
      time.Sleep(5 * time.Millisecond) // let timeout trigger

      w := &Workflow{
          ID:    NewID(),
          Steps: []Step{{Name: "slow"}},
      }
      err := eng.Execute(ctx, w)
      assert.Error(t, err)
      assert.Equal(t, StatusCancelled, w.Status)
  }
  ```

- [ ] **Step 3: Write TestGetStatusOrphan**

  ```go
  func TestGetStatusOrphan(t *testing.T) {
      eng := NewEngine()
      _, err := eng.GetStatus("wf-nonexistent")
      assert.Error(t, err)
      assert.Contains(t, err.Error(), "not found")
  }
  ```

- [ ] **Step 4: Write TestConcurrentExecution**

  ```go
  func TestConcurrentExecution(t *testing.T) {
      eng := NewEngine()
      eng.RegisterStep("simple", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
          return map[string]interface{}{"ok": true}, nil
      })

      var wg sync.WaitGroup
      for i := 0; i < 10; i++ {
          wg.Add(1)
          go func() {
              defer wg.Done()
              w := &Workflow{
                  ID:    NewID(),
                  Steps: []Step{{Name: "simple"}},
              }
              err := eng.Execute(context.Background(), w)
              assert.NoError(t, err)
              assert.Equal(t, StatusCompleted, w.Status)
          }()
      }
      wg.Wait()
  }
  ```

- [ ] **Step 5: Run all workflow tests**
  Run: `go test -count=1 -race ./internal/workflow/...`
  Expected: `ok` (existing 4 + new 4 = 8 total)

- [ ] **Step 6: Commit**
  `git commit -m "test(workflow): add multi-step, cancellation, orphan, and concurrent execution tests"`

---

## Wave B: probe.go Pure Function Tests

### Task B1: probe.go — Table-driven tests for parser functions

**Files:**
- Create: `internal/ingestion/probe_test.go`
- No modification to `probe.go`

**Context:** `probe.go` (558 lines) has ProbeRunner tests via coverage_boost_test.go and engine_extended_test.go, but the pure parser functions (`classifySourceType`, `detectColumns`, `detectPagination`, `buildNextURLFn`, `goValueToColumnType`) have no direct unit tests. These are deterministic — no HTTP or LLM mocking needed.

- [ ] **Step 1: Read actual function signatures**
  Run: `grep -E "^func (classifySourceType|detectColumns|detectPagination|buildNextURLFn|nextPageURL|goValueToColumnType|columnsFromMap)" internal/ingestion/probe.go`
  Verify: function signatures and return types before writing tests.

- [ ] **Step 2: Write TestClassifySourceType** — table-driven
  Test cases: empty string, JSON content, CSV-like, HTML, XML, unknown, null bytes.

- [ ] **Step 3: Write TestDetectColumns** — table-driven
  Test cases: flat objects, nested objects, arrays of scalars, arrays of objects, empty array, null values.

- [ ] **Step 4: Write TestDetectPagination** — table-driven  
  Test cases: Link header, JSON next field, cursor-based, no pagination, malformed headers.

- [ ] **Step 5: Write TestBuildNextURLFn** — table-driven
  Test cases: numbered pages, cursor-based, terminated pagination, malformed URLs.

- [ ] **Step 6: Write TestGoValueToColumnType** — table-driven
  Test cases: int, float, string, bool, nil, map (JSON), slice (JSON array), time-like strings.

- [ ] **Step 7: Run ingestion tests**
  Run: `go test -count=1 ./internal/ingestion/... -v -run 'TestClassifySourceType|TestDetectColumns|TestDetectPagination|TestBuildNextURLFn|TestGoValueToColumnType'`
  Expected: `ok` (5+ new test functions)

- [ ] **Step 8: Commit**
  `git commit -m "test(ingestion): add table-driven tests for probe.go parser functions"`

---

## Wave C: Frontend — Adaptors + Critical Tests

### Task C1: Remove adapters.ts (verified dead code)

**Files:**
- Remove: `frontend/src/api/adapters.ts`

- [ ] **Step 1: Final import verification**
  Run: `grep -r "from.*adapters" frontend/src/ --include="*.ts" --include="*.tsx" | grep -v "nuqs/adapters" | grep -v node_modules`
  Expected: empty (confirmed 0 importers)

- [ ] **Step 2: Check barrel/index files**
  Run: `grep "adapters" frontend/src/api/index.ts 2>/dev/null || echo "no barrel file"`

- [ ] **Step 3: Remove file**
  Run: `git rm frontend/src/api/adapters.ts`

- [ ] **Step 4: Verify build**
  Run: `cd frontend && npx tsc --noEmit 2>&1 | grep -i "adapters" || echo "0 adapters errors"`
  Run: `cd frontend && npx vitest run 2>&1 | tail -3`

- [ ] **Step 5: Commit**
  `git commit -m "refactor(api): remove dead code adapters.ts — 0 importers across 28 consumers"`

---

### Task C2: App.tsx Render Smoke Test

**Files:**
- Create: `frontend/src/__tests__/App.test.tsx`

**Context:** App.tsx is 195 lines with lazy imports (React.lazy), SlideOverContent switch/case, Layout wrapper. Needs mock for store and Suspense.

- [ ] **Step 1: Read App.tsx to identify mocks needed**
  Read: `frontend/src/App.tsx` — inspect useEffect hooks, lazy imports, store selectors.

- [ ] **Step 2: Write smoke test with proper mocks**

  ```typescript
  import { describe, it, expect, vi } from 'vitest';
  import { render, screen } from '@testing-library/react';
  import { Suspense } from 'react';

  // Mock lazy-loaded components
  vi.mock('../components/Component1', () => ({ default: () => <div>Mock1</div> }));
  // Add mocks for all lazy imports found in App.tsx

  describe('App', () => {
    it('renders without crashing', () => {
      const { container } = render(
        <Suspense fallback={<div>Loading...</div>}>
          <App />
        </Suspense>
      );
      expect(container).toBeTruthy();
    });
  });
  ```

- [ ] **Step 3: Run test**
  Run: `cd frontend && npx vitest run src/__tests__/App.test.tsx`
  Expected: PASS (iterate on mocks if needed)

- [ ] **Step 4: Commit**
  `git add frontend/src/__tests__/App.test.tsx && git commit -m "test(frontend): add App.tsx render smoke test"`

---

### Task C3: useSSE Hook Test (341 lines, high value)

**Files:**
- Create: `frontend/src/hooks/__tests__/useSSE.test.ts`

**Context:** useSSE.ts is 341 lines with `extractSSEEvents()`, `scheduleReconnect()`, EventSource lifecycle management, and state transitions. This is the highest-value frontend test target.

- [ ] **Step 1: Read useSSE.ts to understand API**
  Read: `frontend/src/hooks/useSSE.ts` — identify exported symbols, state types, and event handlers.

- [ ] **Step 2: Write tests**
  Tests to cover: connect/disconnect lifecycle, message parsing (`extractSSEEvents`), reconnection delay calculation, error handling, cleanup on unmount.

- [ ] **Step 3: Run + commit**
  Run: `cd frontend && npx vitest run src/hooks/__tests__/useSSE.test.ts`
  Commit: `test(frontend): add useSSE hook unit tests with EventSource mock`

---

### Task C4: CopilotChat Component Test (57 lines, quick win)

**Files:**
- Create: `frontend/src/components/__tests__/CopilotChat.test.tsx`

- [ ] **Step 1: Write render + scroll test**
  Test: renders TerminalOutput wrapper, passes content prop correctly, IntersectionObserver callback.

- [ ] **Step 2: Run + commit**
  Run: `cd frontend && npx vitest run src/components/__tests__/CopilotChat.test.tsx`
  Commit: `test(frontend): add CopilotChat render test`

---

## Wave D: Full Suite Verification

- [ ] **Step 1: Go build + vet**
  ```bash
  go build ./... && go vet ./... && echo "✅ Go OK"
  ```

- [ ] **Step 2: Go test (race)**
  ```bash
  go test -race -count=1 ./internal/workflow/... ./internal/ingestion/... 2>&1 | tail -10
  ```

- [ ] **Step 3: Frontend type check**
  ```bash
  cd frontend && npx tsc --noEmit 2>&1 | tail -5
  ```

- [ ] **Step 4: Frontend tests**
  ```bash
  cd frontend && npx vitest run 2>&1 | tail -5
  ```

- [ ] **Step 5: GitNexus detect_changes**
  ```bash
  npx gitnexus detect-changes
  ```

---

## Execution Order

```
C1 (adapters removal) — 5 min, zero risk → A1 (workflow tests) → B1 (probe tests)
→ C2 (App.tsx) → C3 (useSSE) → C4 (CopilotChat) → D (verification)
```

All subagents must READ the source files they're testing before writing test code — no hallucinated APIs.
