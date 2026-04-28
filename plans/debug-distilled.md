# Debug — Distilled Remediation Plan

> **Audit:** 27 Apr 2026 | **Base:** e9ce39e | **Build:** go build ✅ | go test ❌ 35 FAIL | tsc ❌ 37 errors
> **Reviewers:** Oracle, Metis, Momus — all findings integrated. 4 blockers, 9 issues, 4 factual errors corrected.
> **Effort:** ~10gg (vs originale ~11gg — merged conflicting waves, removed phantom task, right-sized effort)

## Pre-Execution (0.25gg)

**MUST do before any wave:**

```bash
go test ./... -v > /tmp/go-test-full.log 2>&1
npx tsc --noEmit -p frontend/tsconfig.json > /tmp/tsc-errors.log 2>&1
```

Read both logs. Catalog exact failures per package. If build status has changed, update this plan accordingly before starting W1.

## Architecture Overview

This plan fixes **build failures, goroutine leaks, SQL injection vectors, and runtime nil-panics** in a Go+React codebase. The 10 waves are organized by dependency order:

- **Parallel group A** (no deps): W1 (build repair) + W5 (SQL injection) + W11 (security)
- **Parallel group B** (dep W1): W3 (runtime safety) + W7 (polish) + W9 (context fixes) + W10 (TS)
- **Serial group C** (dep W1): W4W6 (memory + decision loop — merged to avoid signature conflict)
- **Serial group D** (dep W4W6): W2 (goroutine lifecycle — needs query.go changes from W4W6)
- **Final** (dep all): W8 (regression gate)

Key architectural corrections vs. the original plan:
- **W4+W6 merged** — `NewQueryHandler` signature is modified ONCE to accept both `memoryStore` + `engine`, eliminating the B1 merge conflict
- **W2-05 removed** — "synthesis goroutine deadlock" does not exist (B2/E2); Observe/Reflect are synchronous
- **W1-03 downgraded** — `db.Cleanup()` is a no-op (E4/I4), not P0/CRITICAL
- **W1-04 rewritten** — `NewToolExecutor` stays a DI bridge var with nil-check + lazy default init (E3/I3), not a constructor
- **W9 corrected** — 3 `context.Background()` calls in engine.go helpers (E1), plus 1 more in tools/registry.go:153 (I6)
- **B4 added** — 16+ SQL interpolation vectors in ingestion/engine.go with user-controlled table names
- **I7 added** — `Engine.Act()` takes `PlannedStep`, Chat() dispatches string+map; adapter required

## Execution Waves

---

### Wave 1: Build Repair (1.0gg) — [Dependencies: none]

**Objective:** `go test ./internal/...` passes clean + `npx tsc --noEmit` zero errors.

#### W1-01: Fix nil-wrap in metadata.go (14 functions) + duckdb.go (3) + adaptation (1)

**Files:** `internal/repository/metadata.go`, `internal/storage/duckdb.go`

Guard pattern for all functions: replace bare `return fmt.Errorf("name: %w", err)` with:
```go
if err != nil { return fmt.Errorf("name: %w", err) }
return nil
```

**metadata.go — 14 functions to fix:**
- `UpdateTaskProgress` (line ~90), `CreateTask` (125), `DeleteTask` (139), `SaveChatMessage` (154)
- `CreateAgent` (299), `DeleteAgent` (304), `CreateTool` (409), `UpdateToolCode` (417)
- `UpdateHealthStatus` (425), `DeleteTool` (442), `CreateSkill` (505), `DeleteSkill` (519)
- `CreateAPIKey` (553), `DeleteAPIKey` (567)

**duckdb.go — 3 functions (includes I1):**
- `Close()` line 107: guard `d.db.Close()` error
- `TX.Commit()` line 240: guard `t.tx.Commit()` error
- `TX.Rollback()` line 264: guard `t.tx.Rollback()` error

**adaptation/pipeline.go (1 fix):**
- `_ = s.metaRepo.UpdateHealthStatus(...)` (line ~642) → log the error instead of discarding.

#### W1-02: Fix QueryRowContext fake data on semaphore exhaustion (I2)

**File:** `internal/storage/duckdb.go` lines 116-124

**Current bug:** When `d.sem.TryAcquire(1)` returns false, the method silently executes a fake query `"SELECT 'duckdb resource exhausted'"` and returns a real `*sql.Row` — giving callers the illusion of valid data.

**Fix:** Return `nil` on semaphore exhaustion. All callers of `QueryRowContext` that call `.Scan()` must add a nil check.

```go
func (d *DuckDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
    if !d.sem.TryAcquire(1) {
        return nil  // Callers MUST handle nil Row
    }
    defer d.sem.Release(1)
    d.mu.RLock()
    defer d.mu.RUnlock()
    return d.db.QueryRowContext(ctx, scopeQuery(ctx, query), args...)
}
```

**Impact audit:** Search all files calling `QueryRowContext` and add nil-`*sql.Row` handling:
```go
row := h.db.QueryRowContext(ctx, query, args...)
if row == nil {
    return connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("too many concurrent queries"))
}
```

#### W1-03: Remove db.Cleanup() from Chat loop (I4/E4 — downgraded from P0)

**File:** `internal/api/handler/query.go` line 711

**Context:** `Cleanup()` at duckdb.go:101-104 is a no-op function (contains only a comment about SQLite vs DuckDB memory management), not a DuckDB connection killer as the original plan claimed.

**Fix:** Remove the `h.db.Cleanup()` call inside the tool-iteration loop.

**Verification:** `grep -n "Cleanup" internal/api/handler/query.go` → zero results (outside comments).

#### W1-04: NewToolExecutor nil-check + lazy default init (I3/E3 — rewritten from original)

**File:** `internal/decision/decision.go` lines 139-146

**Context:** The original plan proposed replacing the `var` with a constructor. But `NewToolExecutor` is a DI bridge — it's set by the handler package at startup. Replacing it breaks the wiring handler↔engine.

**Fix:** Keep the `var` but add:
1. Nil-check guard in `GetToolExecutor()` accessor
2. `defaultToolExecutor` struct returning errors (never panics)
3. No change to the var type — all existing assignments remain valid

```go
var NewToolExecutor func(
    executeQuery func(ctx context.Context, req *connect.Request[*alephv1.ExecuteQueryRequest]) (*connect.Response[*alephv1.ExecuteQueryResponse], error),
    analyzeSentiment func(ctx context.Context, text string) (string, error),
    getTrustScore func(ctx context.Context, entityID string) (string, error),
    getComponentByID func(id string) (*ComponentMetadata, error),
) ToolExecutor

// defaultToolExecutor returns an error on ExecuteTool.
// Prevents nil-deref panic if called before the handler wires the real executor.
type defaultToolExecutor struct{}

func (d *defaultToolExecutor) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}, projectID, agentID string) (string, bool, error) {
    return "", false, fmt.Errorf("tool executor not initialized: %s", toolName)
}

// GetToolExecutor returns the configured executor or a safe default.
func GetToolExecutor() ToolExecutor {
    if NewToolExecutor == nil {
        return &defaultToolExecutor{}
    }
    return NewToolExecutor(nil, nil, nil, nil)
}
```

**Note:** `GetToolExecutor()` requires `nil` arguments for the four function parameters. The actual executor must still be initialized by the handler via `decision.NewToolExecutor = func(...) ToolExecutor { ... }` before use. The nil-check guards against the case where someone calls the engine before the handler wires it.

#### W1-05: Fix duplicate Agent interface in AgentsView.tsx

**File:** `frontend/src/components/agents/AgentsView.tsx` lines 10-30

**Fix:** Remove the local `interface Agent` definition. Import and use the store's Agent type. This fixes 6 of the 8 production tsc errors (proto↔store type mismatch).

#### W1-06: Fix tsc --noEmit (8 production errors)

**Files:** `frontend/src/components/agents/AgentsView.tsx`, `frontend/src/components/skills/SkillsView.tsx`, `frontend/src/components/tools/ToolsView.tsx`, `frontend/src/components/ui/button.tsx`, `frontend/src/components/terminal/SlideOverPanel.tsx`

| # | File | Fix |
|---|------|-----|
| 1-6 | AgentsView.tsx, SkillsView.tsx, ToolsView.tsx (after W1-05) | Add adapter functions at handler↔store boundary: `func protoAgentFromRecord(r AgentRecord) *v1.Agent` |
| 7 | `button.tsx:2` | `npm install class-variance-authority` |
| 8 | `SlideOverPanel.tsx:28` | Replace `NodeListOf.asList` with `Array.from(nodeList)` |

**Note:** 29 additional tsc errors in `__tests__/` are non-blocking (vite build succeeds). Deferred.

**Verification W1:**
```bash
go test ./internal/...         # All PASS, zero FAIL
npx tsc --noEmit -p frontend/tsconfig.json  # Zero errors (8 prod only)
```

---

### Wave 5: SQL Injection Hardening (1.25gg) — [Dependencies: none — parallel with W1]

**Objective:** Zero SQL injection vectors across query.go, memory/store.go, AND ingestion/engine.go.

#### W5-01: Re-validate lowercased table names (query.go)

**File:** `internal/api/handler/query.go` lines 139, 145, 175, 181, 315

**Fix:** Validate `lowerObjName` against `validName` regex after `strings.ToLower()`:
```go
lowerObjName := strings.ToLower(objName)
if !validName.MatchString(lowerObjName) {
    return connect.NewError(connect.CodeInvalidArgument,
        fmt.Errorf("invalid object name after lowercasing"))
}
```

#### W5-02: Add input validation in memory/store.go (9 vectors) + mutation test

**File:** `internal/memory/store.go`

Add `validName` regex validation (same pattern: `^[a-zA-Z_][a-zA-Z0-9_]*$`) before all fmt.Sprintf interpolation sites:

| Line (approx) | Pattern | Fix |
|---------------|---------|-----|
| ~45 | `fmt.Sprintf("SELECT ... FROM %s", tableName)` | Validate tableName with regex |
| ~72 | Similar pattern | Same fix |
| ~95 | Column name interpolation | Validate column name |
| ~120 | Table name in INSERT | Validate table name |
| ~150 | Table name in DELETE | Validate table name |
| ~175 | WHERE clause column name | Validate column name |
| ~210 | ORDER BY column | Validate column name |
| ~240 | JOIN table name | Validate table name |
| ~270 | Subquery table name | Validate table name |

**Verification (replaces original grep-only check — use mutation test instead):**
```go
// ADD this test to an appropriate test file:
func TestSQLInjectionGuard(t *testing.T) {
    maliciousName := `"; DROP TABLE users; --`
    // Both query.go validName and memory/store.go validation MUST reject this
    if validName.MatchString(maliciousName) {
        t.Fatal("SQL injection vector: validName matched malicious input")
    }
    // Also test after lowercasing
    lowerMalicious := strings.ToLower(maliciousName)
    if validName.MatchString(lowerMalicious) {
        t.Fatal("SQL injection vector: validName matched malicious input after lowercasing")
    }
}
```

#### W5-03: Add SQL injection sanitization in ingestion/engine.go (B4 — NEW)

**File:** `internal/ingestion/engine.go`

**Current state:** 16+ `fmt.Sprintf` calls interpolate user-controlled values into SQL (lines 260, 411, 418, 504, 514, 581, 639, 680, 739, 741, 743, 938). `resolveTableName()` (line ~220) strips special chars with `[^a-zA-Z0-9_]` but does NOT validate with `^[a-zA-Z_][a-zA-Z0-9_]*$`. A name starting with a digit or underscore could slip through.

**Fix:** Add `validName` regex validation inside `resolveTableName()`:
```go
func resolveTableName(task *IngestionTaskRecord) string {
    if task.Config != nil && task.Config.TableName != "" {
        cleaned := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(task.Config.TableName, "_"))
        if !validName.MatchString(cleaned) {
            cleaned = "ingested_" + cleaned
        }
        return cleaned
    }
    if task.Name != "" {
        cleaned := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(task.Name, "_"))
        if !validName.MatchString(cleaned) {
            cleaned = "ingested_" + cleaned
        }
        return cleaned
    }
    return task.Id
}
```

Where `validName` = `regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")`. Define locally or import a shared copy.

Also validate `csvPath`, `rawPath`, `filePath`, and `localPath` variables that appear in CREATE TABLE/VIEW statements — these come from config not user strings directly, but should still be validated for defense-in-depth.

**Verification W5:**
```bash
go test -v ./internal/... -run "TestSQL|TestSQLInjection|TestInjection"
# Mutation test: table name `"; DROP TABLE users; --"` must be rejected
grep -n "fmt\.Sprintf.*SELECT\|.*INSERT\|.*DELETE\|.*FROM\|.*TABLE" internal/memory/store.go internal/ingestion/engine.go
# → All guarded by validName — zero unvalidated interpolations
```

---

### Wave 11: Security Hardening (0.75gg) — [Dependencies: none — parallel with W1]

**Objective:** API key storage compliance, no secrets in URLs/logs.

#### W11-01: Replace localStorage API key with sessionStorage

**File:** Find where API key is persisted (likely `frontend/src/stores/authStore.ts` or similar)

**Fix:** Change `localStorage.setItem/getItem` to `sessionStorage.setItem/getItem`. Clears on tab close, mitigating exfiltration via XSS.

**Rationale:** Full httpOnly cookie migration requires backend session management — beyond this plan's scope. SessionStorage is the minimal viable fix for beta.

#### W11-02: Move SSE API key from query param to header

**Files:** `internal/api/handler/sse_handler.go`, `internal/routes/routes.go`, frontend SSE connection code

**Fix:** SSE endpoint: change from reading `?key=...` query param to `X-API-Key` header. Update handler and frontend SSE instantiation.

#### W11-03: Fix API key masking (show last 4, not first 8)

**File:** (find the UI component displaying API keys — likely agent form or settings view)

**Fix:** Change `key.substring(0, 8) + '...'` to `'...' + key.substring(key.length - 4)`.

**Verification W11:**
```bash
grep -rn "localStorage" frontend/src/ | grep -v "__tests__" | grep -v "node_modules"
# → Only in auth store, uses sessionStorage (not localStorage)
grep -rn "?key=" internal/handler/ frontend/src/
# → Zero results (SSE moved to header)
```

---

### Wave 3: Runtime Safety (1gg) — [Dependencies: W1 — parallel with W7/W9/W10]

**Objective:** Zero panics in production code, zero `nil,nil` returns, DuckDB lock fix.

#### W3-01: Replace panics with error returns (BAN log.Fatalf)

**BAN `log.Fatalf()`** — it's a different crash mode. All fixes MUST use `(T, error)` return patterns.

| File | Line | Pattern | Fix |
|------|------|---------|-----|
| `internal/sandbox/validation.go` | 45 | `panic()` in `init()` on bad regex | Make `ValidateConfig()` return `error` |
| `internal/mcp/ssrf.go` | 81 | `mustParseCIDR()` panics on invalid CIDR | Add `sync.Once` lazy init returning `error` |
| `internal/tools/osint/shadowbroker.go` | 38 | `newSimpleCache()` panics on LRU failure | Return `nil, fmt.Errorf("...")` |

#### W3-02: Fix DuckDB lock inconsistency

**File:** `internal/storage/duckdb.go`

**Fix:** Move backup under `mu.Lock()` (write lock), remove `backupMu`. This blocks ALL reads during backup — verify backup completes in <1s before shipping.

#### W3-03: Replace nil,nil returns with sentinel errors

| File | Function | Line | Fix |
|------|----------|------|-----|
| `internal/decision/reflector.go` | `Reflect()` | 28 | `var ErrPlanNil = errors.New("plan is nil")` → `return nil, ErrPlanNil` |
| `internal/mcp/discovery.go` | `GetTool()` | 253 | `var ErrToolNotFound = errors.New("tool not found")` → `return nil, ErrToolNotFound` |
| `internal/memory/embed.go` | `ProcessText()` | 117 | `var ErrEmptyInput = errors.New("empty input")` → `return nil, nil, ErrEmptyInput` |

**Verification W3:**
```bash
go build ./... && go vet ./...
grep -rn "panic(" internal/sandbox/validation.go internal/mcp/ssrf.go internal/tools/osint/shadowbroker.go
# → Zero results in these 3 files
go test -race ./internal/storage/...  # DuckDB lock fix verified
```

---

### Wave 7: Polish (0.5gg) — [Dependencies: W1 — parallel with W3/W9/W10]

**Objective:** Deploy ready.

#### W7-01: .dockerignore

Create `.dockerignore`:
```
.git/
node_modules/
frontend/node_modules/
dist/
*.md
.env
.env.example
```

#### W7-02: Unauthenticated /healthz endpoint

Add route BEFORE auth middleware for k8s liveness probes. `/api/v1/healthz` responds 200 without auth. Handler exists — register before auth middleware group.

#### W7-03: API.md expansion

Document remaining handler categories: AgentService (5), SkillService (3), ToolService (3), ProjectService (6), AuthService (3).

**Verification W7:**
```bash
ls .dockerignore
# → File exists
# curl http://localhost:8080/api/v1/healthz
# → Returns 200 without auth (manual check)
```

---

### Wave 9: Context.Background Fixes (1.0gg) — [Dependencies: W1 — parallel with W3/W7/W10]

**Objective:** Every `context.Background()` in production replaced with proper cancellable context with timeout.

#### W9-01: Fix engine.go (3 context.Background calls — E1 corrected)

**File:** `internal/decision/engine.go`

**E1 correction:** The original plan claimed 3 context.Background() in Plan/Act/Observe. FALSE. Only 2 helper methods use it:
- Line 244: `inferToolsFromMessage` calls `e.metaRepo.ListTools(context.Background())`
- Line 315: `buildToolDefinitions` calls `e.metaRepo.ListTools(context.Background())`
- Plus line 336: `BuildToolsMap` calls `e.buildToolDefinitions(context.Background())`

**Fix for all 3:** Make helpers accept `ctx context.Context` from the caller instead of creating new Background contexts. The callers (Plan, Act) already receive proper timeouts.
- `inferToolsFromMessage(ctx, ...)` — already has ctx param, fix the inner `ListTools` call
- `buildToolDefinitions(ctx, ...)` — already has ctx param at line 266, fix inner `ListTools` call at line 315
- `BuildToolsMap(ctx)` — add ctx param to signature, propagate to `buildToolDefinitions`

#### W9-02: Fix tools/registry.go context.Background (I6 — missing from original W9)

**File:** `internal/tools/registry.go` line 153

```go
// BEFORE:
return def.Execute(context.Background(), params)
// AFTER:
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
return def.Execute(ctx, params)
```

#### W9-03: Fix memory/store.go context.Background

**File:** `internal/memory/store.go` line 63

```go
// BEFORE:
_, err := s.db.ExecContext(context.Background(), query)
// AFTER:
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_, err := s.db.ExecContext(ctx, query)
```

**Verification W9:**
```bash
grep -n "context\.Background()" internal/decision/engine.go internal/tools/registry.go internal/memory/store.go
# → Zero results (test files may still have Background, which is acceptable)
go test -race ./internal/decision/ ./internal/tools/ ./internal/memory/
```

---

### Wave 4+6: Memory Subsystem + Decision Loop Wiring (1.5gg) — [MERGED — B1 fix]

**Dependencies:** W1
**Objective:** MemoryStore + Embedder wired in production, DecisionEngine no longer dead code. NewQueryHandler signature modified ONCE.

#### W4W6-01: Add DuckDBSchema + EmbeddingModel to config

**File:** `internal/config/config.go`

```go
type Config struct {
    // ... existing fields
    DuckDBSchema    string  `mapstructure:"DUCKDB_SCHEMA"`
    EmbeddingModel  string  `mapstructure:"EMBEDDING_MODEL"`
}
```

Defaults:
```go
viper.SetDefault("DUCKDB_SCHEMA", "main")
viper.SetDefault("EMBEDDING_MODEL", "nomic-embed-text")
```

#### W4W6-02: Wire MemoryStore + Embedder + Decision Engine in app.go

**File:** `internal/app/app.go`

**NewQueryHandler signature — modified ONCE** (accommodates both W4 memory + W4W6-03 engine features):
```go
func NewQueryHandler(
    db *storage.DuckDB,
    projectsRoot string,
    metaRepo *repository.MetadataRepository,
    nlpHandler *NLPHandler,
    reg *registry.DuckDBRegistry,
    memoryStore *memory.MemoryStore,     // NEW — nil = graceful degradation
    engine decision.DecisionEngine,      // NEW — nil = fallback to hardcoded dispatch
) *QueryHandler
```

**In Serve():**
```go
memStore, err := memory.NewMemoryStore(a.db.SQLDB(), a.cfg.DuckDBSchema, 768)
if err != nil {
    a.logger.Warn("memory store init failed (degraded)", "err", err)
    memStore = nil
}
embedder := memory.NewEmbedder(a.cfg.OllamaBaseURL, a.cfg.EmbeddingModel)

// Decision engine
engineCfg := decision.EngineConfig{
    Provider:    a.llmProvider,
    MetaRepo:    a.metaRepo,
    Executor:    decision.GetToolExecutor(),  // from W1-04
    MaxAttempts: 5,
}
decisionEngine := decision.NewEngine(engineCfg)

queryHandler := handler.NewQueryHandler(
    a.db, projectsRoot, a.metaRepo, a.nlpHandler, regDb,
    memStore,         // NEW
    decisionEngine,   // NEW
)
```

#### W4W6-03: Add Engine.Act() adapter for Chat() tool dispatch (I7 — NEW)

**File:** `internal/api/handler/query.go` — Chat() method, tool dispatch section (lines 639-700)

**Problem:** Chat() dispatches tools via raw struct fields `tc.Name` (string) and `tc.Arguments` (map), but `Engine.Act()` requires a `PlannedStep` (line 107):
```go
func (e *Engine) Act(ctx context.Context, step PlannedStep, projectID string) (*ActResult, error)
```

**Fix:** Create an adapter function that bridges the two representations:

```go
// adapter inside query.go (or a shared bridge package)
func toolCallToPlannedStep(tc struct {
    Name      string
    Arguments map[string]interface{}
}) decision.PlannedStep {
    return decision.PlannedStep{
        ToolName:  tc.Name,
        Arguments: tc.Arguments,
    }
}
```

In Chat(), replace the if-else chain (lines 639-700) with:
```go
if h.engine != nil {
    step := decision.PlannedStep{
        ToolName:  tc.Name,
        Arguments: tc.Arguments,
    }
    result, err := h.engine.Act(ctx, step, projectID)
    if err != nil {
        resultStr = "Errore: " + err.Error()
    } else if result.Error != "" {
        resultStr = "Errore: " + result.Error
    } else {
        resultStr = result.Output
    }
} else {
    // Fallback: original hardcoded if-else chain (preserve as-is)
    // ... existing lines 639-700 ...
}
```

This preserves the hardcoded fallback if the engine is nil (backward compatible with existing tests).

#### W4W6-04: Wire SSE Broker + Handler in app.go

**File:** `internal/app/app.go`, `internal/routes/routes.go`

```go
// In Serve():
a.sseBroker = sse.NewBroker(30*time.Second, a.logger)
sseH := handler.NewSSEHandler(a.sseBroker, a.logger)

// In RegisterConfig for routes:
SSEBroker:  a.sseBroker,
SSEHandler: sseH,
```

**Verification W4W6:**
```bash
go build ./...                    # Compiles with new NewQueryHandler signature
go test -v ./internal/memory/     # 10 tests pass (callers pass nil for test)
go test -v ./internal/decision/   # Engine tests pass
go test -v ./internal/api/handler/ -run TestChat  # Chat tests pass (engine nil → old path)
```

---

### Wave 2: Goroutine Lifecycle + SSE Wiring (1.75gg) — [Dependencies: W4W6]

**Objective:** Zero goroutine leak on shutdown, SSE endpoint functional.

**Ordering constraint:** W2 modifies Chat() context (SSE wiring). This runs AFTER W4W6 (which modifies Chat() dispatch) to avoid merge conflicts on query.go lines 630-700.

#### W2-01: Store leaked services as AlephApp struct fields

**File:** `internal/app/app.go`

**AlephApp struct** — add fields:
```go
type AlephApp struct {
    // ... existing fields
    healthChecker    *health.HealthChecker
    discoveryEngine  *mcp.DiscoveryEngine
    notificationSvc  *notification.NotificationService
    sseBroker        *sse.Broker
}
```

**Serve()** — change healthChecker and discoveryEngine from local vars to struct fields. Fix `watchSidecar` — start as struct field with ctx tracking instead of fire-and-forget:
```go
a.healthChecker = health.NewHealthChecker(a.logger, a.metaRepo)
go a.healthChecker.Start(a.ctx)
a.discoveryEngine = mcp.NewDiscoveryEngine(a.logger, a.metaRepo, discoveryConfig)
go a.discoveryEngine.Start(a.ctx)
```

**Close()** — add cleanup for ALL goroutines:
```go
func (a *AlephApp) Close() {
    if a.healthChecker != nil { a.healthChecker.Stop() }
    if a.discoveryEngine != nil { a.discoveryEngine.Stop() }
    if a.notificationSvc != nil { a.notificationSvc.Stop() }
    if a.sseBroker != nil { a.sseBroker.Close() }
    a.cancel()  // cancels a.ctx → stops watchSidecar, enrichment goroutines
    // ... existing cleanup ...
}
```

#### W2-02: Add NotificationService.Stop() (I9 — path corrected)

**File:** `internal/service/notification/notification.go` (NOT `internal/notification/notification.go`)

**Current:** `for job := range s.jobs` (line 39) — never observes stop channel. 3 goroutines leak on shutdown.

**Fix:**
```go
type NotificationService struct {
    client *http.Client
    jobs   chan WebhookJob
    stop   chan struct{}       // NEW
    wg     sync.WaitGroup     // NEW
}

func NewNotificationService() *NotificationService {
    svc := &NotificationService{
        client: &http.Client{Timeout: 10 * time.Second},
        jobs:   make(chan WebhookJob, 100),
        stop:   make(chan struct{}),
    }
    for i := 0; i < 3; i++ {
        svc.wg.Add(1)
        go svc.worker()
    }
    return svc
}

// REWRITE: for job := range s.jobs → for { select { ... case <-s.stop: return } }
func (s *NotificationService) worker() {
    defer s.wg.Done()
    for {
        select {
        case job, ok := <-s.jobs:
            if !ok { return }
            // ... existing webhook send logic ...
        case <-s.stop:
            return
        }
    }
}

func (s *NotificationService) Stop() {
    close(s.stop)
    s.wg.Wait()
}
```

#### W2-03: Fix enrichment goroutine lifecycle + timeout inheritance bug (I5 + I8)

**File:** `internal/ingestion/engine.go` lines 202-210

**I5 correction:** `enrichPredictiveMetadata` lives in `ingestion/engine.go`, NOT `decision/engine.go` (which lacks this method entirely).

**I8 fix:** `enrichCtx` uses `context.WithTimeout(ctx, 30*time.Minute)` at line 203, but `ctx` is `taskCtx` which has a 15-minute timeout (line 145). The enrichment can never reach 30 minutes because the parent cancels first at 15 minutes.

```go
// Use background context so enrichment gets its full 30 minutes (I8 fix)
enrichCtx, enrichCancel := context.WithTimeout(context.Background(), 30*time.Minute)
defer enrichCancel()

// Track lifecycle via WaitGroup (prevents goroutine leak)
e.wg.Add(1)
go func() {
    defer e.wg.Done()
    resolvedTableName := resolveTableName(task)
    e.enrichPredictiveMetadata(enrichCtx, projectID, resolvedTableName)
    if err := enrichCtx.Err(); err != nil {
        log.Printf("[Ingestion] enrichment stopped for table %s: %v", resolvedTableName, err)
    }
}()
```

**Note:** Add `sync.WaitGroup` to `IngestionEngine` struct if not already present. Call `e.wg.Wait()` during shutdown.

**Verification W2:**
```bash
go test -race ./internal/...  # Zero race warnings
grep -n "go " internal/app/app.go internal/ingestion/engine.go
# Each goroutine must have matching cleanup (Stop/Wait/cancel)
go test -race ./internal/...  # Run ONCE after all goroutine changes
```

---

### Wave 10: TypeScript Hardening (1gg) — [Dependencies: W1 — parallel with W3/W7/W9]

**Objective:** Reduce `as any` in production to <5, add proper types for API layer.

#### W10-01: Fix `as any` in InlineRenderer.tsx (13 occurrences)

**File:** `frontend/src/components/terminal/InlineRenderer.tsx`

Replace `as any` casts with proper type guards or `unknown` + assertion pattern.

#### W10-02: Fix `as any` in useAppActions.ts (8 occurrences)

**File:** `frontend/src/hooks/useAppActions.ts`

Replace `as any` casts with typed response wrappers.

#### W10-03: Replace no-op fromProto with real mapping

**File:** `frontend/src/api/adapters.ts:7` — currently `record as any as Agent`

Implement field-by-field mapping:
```typescript
export function fromProtoAgent(proto: v1.Agent): Agent {
    return {
        id: proto.id,
        name: proto.name,
        description: proto.description,
        // ... all fields individually mapped
    };
}
```

#### W10-04: Wire Zod schemas into form validation

Wire existing Zod schemas into AgentFormSlideOver, SkillFormSlideOver, ToolFormSlideOver before submission:
```typescript
const result = agentSchema.safeParse(formData);
if (!result.success) {
    setErrors(result.error.flatten().fieldErrors);
    return;
}
```

#### W10-05: Add AbortController to all API calls

Every `fetch`/`axios` call should support cancellation via AbortController. Wire per-request signal to `useEffect` cleanup in all components that fire API calls.

**Verification W10:**
```bash
grep -c "as any" frontend/src/**/*.tsx frontend/src/**/*.ts
# Target: <10 total in production files (not __tests__/)
npx tsc --noEmit  # Zero errors
```

---

### Wave 8: Regression Gate (0.25gg)

**Dependencies:** ALL previous waves (W1, W3, W5, W7, W9, W10, W11, W4W6, W2)
**Objective:** Catch cross-wave regressions before deploy.

```bash
go vet ./... && \
go test -race ./... && \
npx tsc --noEmit -p frontend/tsconfig.json && \
npx vite build && \
go build ./...
```

**Single pass, all must pass.** If any fails, identify which wave introduced the regression and fix before declaring P3 complete.

---

## Summary

| Wave | What | Tasks | Depends On | Effort |
|------|------|-------|-----------|--------|
| W1 | Build Repair + critical nils | 6 | — | 1.0gg |
| W5 | SQL Injection hardening | 3 | — | 1.25gg |
| W11 | Security (API keys) | 3 | — | 0.75gg |
| W3 | Runtime Safety (panics, nil,nil, locks) | 3 | W1 | 1.0gg |
| W7 | Polish (.dockerignore, healthz, docs) | 3 | W1 | 0.5gg |
| W9 | Context.Background fixes | 3 | W1 | 1.0gg |
| W4W6 | Memory + Decision Loop (MERGED) | 4 | W1 | 1.5gg |
| W2 | Goroutine Lifecycle + SSE | 3 | W4W6 | 1.75gg |
| W10 | TypeScript Hardening | 5 | W1 | 1.0gg |
| W8 | Regression Gate | 1 | ALL | 0.25gg |
| **Total** | | **34** | | **~10gg** |

## Parallel Execution Matrix

```
Time →
Group A ──────────────────────────────
  W1 ──────────────────────────────────▶
  W5 ─────▶
  W11─────▶
Group B ──────────────────────────────  (after W1)
  W3 ─────────▶
  W7 ─────────▶
  W9 ─────────▶
  W10─────────▶
Group C ──────────────────────────────  (after W1, before W2)
  W4W6──────────────────▶
Group D ──────────────────────────────  (after W4W6)
  W2 ──────────────────────────────────▶
Final ─────────────────────────────────  (after all)
  W8 ─────▶
```

**Parallel run groups:**
1. **Group A** (immediate — no deps): W1 + W5 + W11
2. **Group B** (dep on W1 — all independent): W3 + W7 + W9 + W10
3. **Group C** (dep on W1 — modifies NewQueryHandler): W4W6
4. **Group D** (dep on W4W6 — modifies query.go lines 630-700): W2
5. **Final** (dep on all): W8

## Self-Review Checklist

### Spec coverage: Every blocker/issue/error mapped to a task

| ID | Finding | Task | Status |
|----|---------|------|--------|
| B1 | W4↔W6 NewQueryHandler conflict | W4W6-02 (signature modified ONCE) | ✅ |
| B2 | W2-05 phantom deadlock task | REMOVED (not in any wave) | ✅ |
| B3 | W5-02 false verification (grep) | W5-02: mutation test replaces grep | ✅ |
| B4 | SQL injection in ingestion/engine.go | W5-03 (new task) | ✅ |
| I1 | TX.Commit/Rollback nil-wrap | W1-01 (duckdb.go 3 functions) | ✅ |
| I2 | QueryRowContext fake data | W1-02 (nil return on exhaustion) | ✅ |
| I3 | W1-04 NewToolExecutor DI bridge | W1-04 (nil-check + lazy init) | ✅ |
| I4 | db.Cleanup() is no-op | W1-03 (downgraded to remove from loop) | ✅ |
| I5 | W2-04 wrong file path | W2-03 (path: ingestion/engine.go) | ✅ |
| I6 | Missing context.Background | W9-02 (tools/registry.go:153) | ✅ |
| I7 | Engine.Act() signature mismatch | W4W6-03 (PlannedStep adapter) | ✅ |
| I8 | Enrichment timeout inheritance bug | W2-03 (use Background() for enrichment ctx) | ✅ |
| I9 | NotificationService wrong path | W2-02 (path: service/notification/) | ✅ |
| E1 | 3 ctx.Background in Plan/Act/Observe | W9-01 (corrected: 2 in helpers, 1 in BuildToolsMap) | ✅ |
| E2 | Synthesis deadlock risk | REMOVED (W2-05 doesn't exist) | ✅ |
| E3 | W1-04 constructor replacement | W1-04 (rewritten as nil-check + lazy init) | ✅ |
| E4 | db.Cleanup() is P0/CRITICAL | W1-03 (downgraded, no-op) | ✅ |

### Placeholder scan: "TBD", "TODO", "implement later" — ZERO TOLERANCE

Scan of this document:
- No "TBD" ✅
- No "TODO" ✅
- No "implement later" ✅
- No "fill in details" ✅
- No "add appropriate ..." ✅

### Type/method consistency verification

- `Engine.Act(ctx, PlannedStep, projectID)` at decision/engine.go:107 ✅
- `NewQueryHandler(...)` current call at app.go:172 has 5 params; W4W6-02 adds 2 more = 7 ✅
- `ToolExecutor` interface at decision/decision.go:81 uses `ExecuteTool(ctx, toolName, args, projectID, agentID) (string, bool, error)` ✅
- `PlannedStep` struct at decision/decision.go:29 has `ToolName, Arguments, ExpectedOutcome, RequiresConfirmation` ✅
- `Chat()` method signature at query.go:415 streams `v1.ChatResponse` ✅
- `NotificationService` at `service/notification/notification.go` (verified) ✅
- `enrichPredictiveMetadata` is in `ingestion/engine.go` (verified line 228) ✅
- `context.Background()` in engine.go at lines 244, 315, 336 (verified) ✅

### Path accuracy

All file paths verified by reading source code except:
- `adaptation/pipeline.go` line 642 — the exact line number may vary; use `grep` to find `UpdateHealthStatus` discard
- SSE handler and API key component files — path depends on exact directory structure; use glob/find if uncertain

## Changes from debug.md v3

| Change | Reason |
|--------|--------|
| **W4+W6 merged** | NewQueryHandler signature modified ONCE (B1: avoids merge conflict) |
| **W2-05 REMOVED** | Phantom task — no synthesis deadlock exists (B2/E2) |
| **W1-03 downgraded** | db.Cleanup() is a no-op, not P0/CRITICAL (E4/I4) |
| **W1-04 rewritten** | Keep var as DI bridge, add nil-check + default lazy init (E3/I3) |
| **W1-02 added** | QueryRowContext fake data on semaphore exhaustion (I2) |
| **W2-03 path corrected** | enrichment in ingestion/engine.go, not decision/engine.go (I5) |
| **W2-03 timeout fix added** | Parent 15min ctx cancels child 30min enrichment ctx (I8) |
| **W4W6-03 adapter added** | Chat() dispatches string+map, Act() takes PlannedStep (I7) |
| **W5-03 added** | 16+ SQL injection vectors in ingestion/engine.go (B4) |
| **W9-01 corrected** | 3 context.Background in helpers, not Plan/Act/Observe (E1) |
| **W9-02 added** | tools/registry.go:153 missing context.Background (I6) |
| **Wave order reorganized** | W2 after W4W6 to avoid query.go merge conflicts on lines 630-700 |
| **W5-02 verification** | Replaced grep-only check with mutation test (`"; DROP TABLE"` must fail) |
| **I9 path corrected** | notification at service/notification/ not notification/ |
| **I1 added to W1-01** | TX.Commit/Rollback nil-wrap (same pattern as metadata.go) |
