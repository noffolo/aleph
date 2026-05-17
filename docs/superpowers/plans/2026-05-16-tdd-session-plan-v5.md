# TDD Session Plan v5 — Aleph-v2 Final Coverage Gap Closure

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the last two remaining coverage gaps identified by GitNexus (22,763 nodes): `internal/app` (6.0%, 1 test file, wiring/DI) and `internal/sandbox` (58.7%, 10 test files, security/OS branches). Target: internal/app >40%, internal/sandbox >70%.

**Architecture:** 2 independent waves — Wave A targets `internal/app` testable paths (watchSidecar state machine, setupDemoData early-return guards, Close nil-safety). Wave B targets `internal/sandbox` highest-value untested functions (container lifecycle, security scanner, namespace isolation).

**Tech Stack:** Go 1.26 (testing + testify/suite for watchSidecar), Docker (for container_sandbox tests), DuckDB in-memory (for setupDemoData schema).

**GitNexus baseline:** 22,763 nodes, 56,228 edges, 805 clusters, 300 processes.

---

## Baseline: Previously Completed

| Wave | Task | Status |
|------|------|--------|
| C1 | Remove adapters.ts (dead code) | ✅ `git rm` + tsc 0 new errors + vitest pass |
| A1 | Workflow tests (errors.Is bugfix) | ✅ `errors.Is` fix committed, 9/9 tests pass |
| v2 Wave 0 | CI NLP job | ✅ NLP test in .github/workflows/ci.yml |
| v2 Wave 1 | DuckDB Registry ListComponents filter | ✅ Dynamic WHERE clause + 16 tests |
| v2 Wave 2 | factory.ts smoke test | ✅ 54 lines, 3/3 vitest pass |
| v4 review | probe.go, App.tsx, useSSE, CopilotChat | ✅ Already tested (plan had stale assumptions) |

---

## Wave A: internal/app — Wiring/DI Testing (target >40%)

**Package overview:** `internal/app` (974 total lines): `AlephApp` struct (22 fields), `NewAlephApp` factory (95 lines), `Serve` wiring (229 lines), `Close` shutdown (62 lines), `runSecurityScan` (30 lines), `makeSentimentHelper` (18 lines), `makeTrustScoreHelper` (18 lines), `makeComponentByIDHelper` (18 lines), `newH2CClient` (18 lines), `newTLSClient` (24 lines), `watchSidecar` (85 lines), `setupDemoData` (140 lines).

**Currently tested (119 lines of app_test.go):** newH2CClient SSRF, newTLSClient SSRF, makeSentimentHelper nil case, makeTrustScoreHelper nil case, makeComponentByIDHelper nil case.

**Untested high-value targets (by lines):** watchSidecar (85), setupDemoData (140), Close (62), runSecurityScan (30), NewAlephApp edge cases (95, partial).

### Task A5.1: watchSidecar — goroutine health-check state machine

**Files:**
- Create: `internal/app/watch_sidecar_test.go`
- No modification to app.go

**Context:** `watchSidecar` (app.go:608-692) is a 85-line goroutine with a health check loop, gRPC `HealthCheck` call, backoff with restart counting, max-restarts-with-window guard, context cancellation, MarkHealthy/MarkUnhealthy on nlpHandler, and panic recovery. This is the highest-value test target in the package — it contains non-trivial state machine logic.

**Strategy:** Refactor the health-check loop body into a testable function `func (a *AlephApp) checkSidecarOnce(client grpc_health_v1.HealthClient, nlpHandler *handler.NLPHandler, consecutiveErr *bool, restartCount *int, restartStart *time.Time) bool` that returns `continueLoop bool`. The goroutine wrapper then just calls this in a loop with ticker + ctx.Done(). This decomposition makes the state machine testable without goroutine coordination.

- [ ] **Step 1: Impact analysis**
  Run: `npx gitnexus impact --target watchSidecar --direction downstream`
  Verify: only called from `Serve` (line 397).

- [ ] **Step 2: Extract checkSidecarOnce function**
  Add to `app.go`:

  ```go
  // checkSidecarOnce performs a single health-check iteration.
  // Returns true if the loop should continue, false to stop.
  // Extracted for testability from watchSidecar.
  func (a *AlephApp) checkSidecarOnce(
      client grpc_health_v1.HealthClient,
      nlpHandler *handler.NLPHandler,
      consecutiveErr *bool,
      restartCount *int,
      restartStart *time.Time,
  ) bool {
      ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
      defer cancel()
      resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: "aleph.nlp.v1.NLPService"})
      if err != nil {
          a.logger.Warn("sidecar non risponde", "error", err)
          nlpHandler.MarkUnhealthy()
          if !*consecutiveErr {
              *consecutiveErr = true
              *restartCount = 0
              *restartStart = time.Now()
          }
          *restartCount++
          if *restartCount > maxRestarts && time.Since(*restartStart) < restartWindow {
              a.logger.Error("sidecar watchdog: too many failures in window, giving up",
                  "restarts", *restartCount, "window", restartWindow)
              return false
          }
          step := *restartCount - 1
          if step >= len(backoffSteps) {
              step = len(backoffSteps) - 1
          }
          a.logger.Info("sidecar watchdog: will retry", "attempt", *restartCount, "backoff", backoffSteps[step])
          time.Sleep(backoffSteps[step])
      } else {
          *consecutiveErr = false
          *restartCount = 0
          if resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
              nlpHandler.MarkHealthy()
          } else {
              a.logger.Warn("sidecar non SERVING", "status", resp.GetStatus())
          }
      }
      return true
  }
  ```

  Move `maxRestarts`, `restartWindow`, `backoffSteps` to package-level vars:

  ```go
  var (
      sidecarMaxRestarts     = 3
      sidecarRestartWindow   = 5 * time.Minute
      sidecarBackoffSteps    = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}
      sidecarHealthInterval  = 5 * time.Second
      sidecarHealthTimeout   = 3 * time.Second
  )
  ```

- [ ] **Step 3: Write failing test — TestCheckSidecarOnce_HealthOK**

  ```go
  package app

  import (
      "context"
      "testing"
      "time"
      "google.golang.org/grpc/health/grpc_health_v1"
      "github.com/stretchr/testify/assert"
  )

  func TestCheckSidecarOnce_HealthOK(t *testing.T) {
      a := &AlephApp{
          logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
          ctx:    context.Background(),
          nlpHandler: &handler.NLPHandler{},
      }
      mockClient := &mockHealthClient{status: grpc_health_v1.HealthCheckResponse_SERVING}
      var consecutiveErr bool
      var restartCount int
      var restartStart time.Time

      shouldContinue := a.checkSidecarOnce(mockClient, a.nlpHandler, &consecutiveErr, &restartCount, &restartStart)
      assert.True(t, shouldContinue)
      assert.False(t, consecutiveErr)
      assert.Equal(t, 0, restartCount)
  }
  ```

  Note: `mockHealthClient` and `handler.NLPHandler` types need to be available. If `MarkHealthy`/`MarkUnhealthy` are unexported, use interface mocking.

- [ ] **Step 4: Write failing test — TestCheckSidecarOnce_ErrorThenRecovery**

  ```go
  func TestCheckSidecarOnce_ErrorThenRecovery(t *testing.T) {
      a := &AlephApp{
          logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
          ctx:    context.Background(),
      }
      mockClient := &mockHealthClient{err: assert.AnError}
      var consecutiveErr bool
      var restartCount int
      restartStart := time.Now()

      shouldContinue := a.checkSidecarOnce(mockClient, nil, &consecutiveErr, &restartCount, &restartStart)
      assert.True(t, shouldContinue)
      assert.True(t, consecutiveErr)
      assert.Equal(t, 1, restartCount)
  }
  ```

- [ ] **Step 5: Write failing test — TestCheckSidecarOnce_MaxRestartsExceeded**

  ```go
  func TestCheckSidecarOnce_MaxRestartsExceeded(t *testing.T) {
      a := &AlephApp{
          logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
          ctx:    context.Background(),
      }
      mockClient := &mockHealthClient{err: assert.AnError}
      consecutiveErr := true
      restartCount := 4  // > maxRestarts
      restartStart := time.Now()

      shouldContinue := a.checkSidecarOnce(mockClient, nil, &consecutiveErr, &restartCount, &restartStart)
      assert.False(t, shouldContinue) // watchdog triggered
  }
  ```

- [ ] **Step 6: Implement checkSidecarOnce and mockHealthClient**

  ```go
  // In test file
  type mockHealthClient struct {
      grpc_health_v1.HealthClient
      status grpc_health_v1.HealthCheckResponse_ServingStatus
      err    error
  }

  func (m *mockHealthClient) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest, opts ...grpc.ClientOption) (*grpc_health_v1.HealthCheckResponse, error) {
      if m.err != nil {
          return nil, m.err
      }
      return &grpc_health_v1.HealthCheckResponse{Status: m.status}, nil
  }
  ```

- [ ] **Step 7: Run tests**
  Run: `go test -count=1 -run TestCheckSidecarOnce ./internal/app/`
  Expected: `ok` (3/3 pass)

- [ ] **Step 8: Run full app test suite**
  Run: `go test -count=1 ./internal/app/...`
  Expected: pre-existing + new tests pass

- [ ] **Step 9: Commit**
  ```bash
  git add internal/app/app.go internal/app/watch_sidecar_test.go
  git commit -m "test(app): extract checkSidecarOnce, add health-check state machine tests"
  ```

### Task A5.2: setupDemoData — early-return guard tests

**Files:**
- Create: `internal/app/onboarding_test.go`
- No modification to onboarding.go

- [ ] **Step 1: Write TestSetupDemoData_NilMetaRepo**

  ```go
  func TestSetupDemoData_NilMetaRepo(t *testing.T) {
      a := &AlephApp{
          logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
      }
      // Should return immediately without panic
      a.setupDemoData("/tmp/nonexistent")
  }
  ```

- [ ] **Step 2: Write TestSetupDemoData_CountError**

  ```go
  func TestSetupDemoData_CountError(t *testing.T) {
      mockMeta := &mockMetaRepo{countErr: assert.AnError}
      a := &AlephApp{
          logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
          metaRepo: mockMeta,
      }
      a.setupDemoData("/tmp/nonexistent")
  }
  ```

- [ ] **Step 3: Write TestSetupDemoData_ProjectsExist**

  ```go
  func TestSetupDemoData_ProjectsExist(t *testing.T) {
      mockMeta := &mockMetaRepo{count: 5}
      a := &AlephApp{
          logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
          metaRepo: mockMeta,
      }
      a.setupDemoData("/tmp/nonexistent")
      // Should log "projects found" and return
  }
  ```

- [ ] **Step 4: Run tests + commit**
  Run: `go test -count=1 -run TestSetupDemoData ./internal/app/`
  Commit: `test(app): add setupDemoData early-return guard tests`

### Task A5.3: AlephApp.Close — nil-safety and sequencing tests

**Files:**
- Modify: `internal/app/app_test.go` (add tests)

- [ ] **Step 1: Write TestClose_NilAlephApp**

  ```go
  func TestClose_NilFields(t *testing.T) {
      // All nil fields — should not panic
      a := &AlephApp{}
      ctx := context.Background()
      err := a.Close(ctx)
      assert.NoError(t, err)
  }

  func TestClose_WithCancelOnly(t *testing.T) {
      ctx, cancel := context.WithCancel(context.Background())
      a := &AlephApp{
          cancel: cancel,
          ctx:    ctx,
          logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
      }
      err := a.Close(context.Background())
      assert.NoError(t, err)
  }
  ```

- [ ] **Step 2: Run tests + commit**
  Run: `go test -count=1 -run TestClose ./internal/app/`
  Commit: `test(app): add AlephApp.Close nil-safety and sequencing tests`

---

## Wave B: internal/sandbox — Security Gap Closure (target >70%)

**Package overview:** `internal/sandbox/` (22 files, 10 test files, 58.7% coverage). Key untested areas:
- `container_sandbox.go` — container lifecycle (Start, Stop, Exec) — needs Docker
- `security.go` — security scanner rules
- `seccomp_profile.go` — seccomp profile generation
- `namespace_isolated.go` — namespace isolation setup

### Task B5.1: SecurityScanner — rule-based scanning tests

**Files:**
- Create or modify: `internal/sandbox/security_test.go`
- No modification to security.go

**Context:** `security.go` has `SecurityScanner` with rule-based code scanning. Existing security tests likely cover the scanner framework but may miss specific rule logic or edge cases.

- [ ] **Step 1: Write table-driven tests for each security rule**
  Test: dangerous imports, exec calls, filesystem access, network access patterns.

- [ ] **Step 2: Run tests + commit**
  Run: `go test -count=1 -run TestSecurity ./internal/sandbox/...`
  Commit: `test(sandbox): add table-driven security rule tests`

### Task B5.2: seccomp profile tests

**Files:**
- Create or modify: `internal/sandbox/seccomp_profile_test.go`

- [ ] **Step 1: Test profile generation for various configurations**
  Test: default profile, minimal profile, custom syscall allowlists.

- [ ] **Step 2: Run + commit**

### Task B5.3: Namespace isolation edge cases

**Files:**
- Create or modify: `internal/sandbox/namespace_isolation_test.go`

- [ ] **Step 1: Test namespace setup error paths**
  Test: permission denied, unsupported OS, resource exhaustion.

- [ ] **Step 2: Run + commit**

---

## Wave C: Full Suite Verification

- [ ] **Step 1: Go build + vet**
  ```bash
  go build ./... && go vet ./... && echo "✅ Pass"
  ```

- [ ] **Step 2: Go test full**
  ```bash
  go test -count=1 ./... 2>&1 | tail -30
  ```

- [ ] **Step 3: GitNexus re-index**
  ```bash
  npx gitnexus analyze --force
  ```

- [ ] **Step 4: Verify coverage improvement**
  ```bash
  go test -coverpkg=./internal/app,./internal/sandbox/... -coverprofile=coverage.out ./internal/app ./internal/sandbox/...
  go tool cover -func=coverage.out | grep -E "internal/app|internal/sandbox"
  ```

- [ ] **Step 5: Generate final coverage report**
  Produce `docs/superpowers/reports/2026-05-16-coverage-report-v5.md` with before/after comparison.

---

## Execution Strategy (via work-strategy skill)

| Dimension | Rating | Rationale |
|-----------|--------|-----------|
| Complexity | **medium** | Mostly additive test code; watchSidecar extraction is the only refactor |
| Risk | **low** | Tests only + one extracted method. watchSidecar original stays as wrapper. |
| Dependencies | **weak** | Waves A and B are fully independent |
| Strategy | **orchestrate** | A and B can run in parallel since they target different packages |

**Execution flow:**
```
Wave A (internal/app) — subagent-driven-development
  → A5.1 watchSidecar extraction + tests → review
  → A5.2 setupDemoData tests → review
  → A5.3 Close tests → review
Wave B (internal/sandbox) — subagent-driven-development
  → B5.1 SecurityScanner tests → review
  → B5.2 seccomp profile tests → review
  → B5.3 Namespace isolation tests → review
Wave C (Verification) — direct
  → Full suite + coverage report
```
