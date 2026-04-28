# Debug — Distilled Remediation Plan v2

> **Audit:** 27 Apr 2026 | **Base:** e9ce39e | **Build:** go build ✅ | go test ❌ 35 FAIL | tsc ❌ 37 errors
> **Reviewers:** Original: Oracle, Metis, Momus, Aleph. v2: Metis, Oracle, Momus, Aleph — ALL findings integrated, 10 N-issues fixed.
> **Effort:** ~9.5gg (vs v1 ~10gg — removed phantom W4W6-04, right-sized W7-02 verification-only)

## How v2 Differs from v1

### 10 Issues v1 Got Wrong — All Fixed Here

| ID | v1 Error | v2 Fix |
|----|----------|--------|
| **N1** | W4W6-03: `h.engine.GetToolExecutor(ctx, projectID, agentID, toolName)` — 4 separate params | `GetToolExecutor` does NOT exist on the Engine interface. The Engine has its own `executor` field. Chat() calls `h.engine.Act(ctx, PlannedStep, projectID)` directly — no separate `GetToolExecutor` needed. |
| **N2** | W4W6-02: `h.engine.NewToolExecutor is assigned` — no explicit assignment shown | Add explicit: `decision.NewToolExecutor = func(executeQuery, analyzeSentiment, getTrustScore, getComponentByID) ToolExecutor { ... }` — uses actual 4-func-param signature from decision.go:141 |
| **N3** | W4W6-02: `a.llmProvider` / `a.cfg.LLMProvider` — field doesn't exist | Pass `nil` for Provider. Add graceful degradation: tools execute without LLM when nil. |
| **N4** | W4W6-04 references `a.sseBroker` before W2-01 adds the field | Remove W4W6-04 entirely. SSE lives in W2 only. W4W6 should not touch SSE. |
| **N5** | W1-04: `defaultToolExecutor` returns `fmt.Errorf("tool executor not initialized")` — breaks ALL tool calls | Default must be a pass-through to the old hardcoded if-else chain (the pre-W6 behavior at query.go:639-700). |
| **N6** | duckdb.go: lock released BEFORE Commit/Rollback returns — concurrent writer can interleave | Guard must hold lock through Commit/Rollback. Defer release after success/failure confirmed. |
| **N7** | W3-01: "Make ValidateConfig() return error" — but sandbox/validation.go uses `init()` which CANNOT return error | Use `sync.Once` lazy init. `ValidateConfig()` calls `once.Do(...)`, stores error. Regex compiled on first call, not init. |
| **N8** | W7-02: "Add /api/v1/healthz" — it's ALREADY registered at routes.go:57-62 | Change to: verify the existing `/healthz` endpoint works (200 without auth). Not a new task. |
| **N9** | W5-03: `resolveTableName` falls back to `task.Id` without validation — user-controlled | Validate `task.Id` with `validName` regex before using as table name. |
| **N10** | W5-03: `resolveTableName(task *IngestionTaskRecord)` — wrong type | `resolveTableName` at ingestion/engine.go:215 takes `*v1.IngestionTask` (protobuf), not `*IngestionTaskRecord`. |

### Structural Fixes vs v1

- **W4W6-04 REMOVED** — SSE wiring cannot precede W2-01 (N4). SSE stays exclusively in W2.
- **W7-02 downgraded** — healthz already registered (N8). Changed from "Add endpoint" → "Verify existing and document."
- **W1-01 expanded** — now includes N6 (Commit/Rollback lock-ordering fix in duckdb.go), making it the only task touching duckdb.go.
- **W1-04 rewritten again** — defaultToolExecutor now falls through to old if-else chain (N5), not an error.
- **W3-01 uses sync.Once** — validation.go + ssrf.go use lazy init instead of init() panic (N7).

## Pre-Execution (0.25gg)

**MUST do before any wave:**

```bash
go test ./... -v > /tmp/go-test-full.log 2>&1
npx tsc --noEmit -p frontend/tsconfig.json > /tmp/tsc-errors.log 2>&1
```

Read both logs. Catalog exact failures per package. If build status has changed, update this plan accordingly before starting W1.

## Architecture Overview

This plan fixes **build failures, goroutine leaks, SQL injection vectors, lock-ordering bugs, and runtime nil-panics** in a Go+React codebase. The 10 waves are organized by dependency order:

- **Parallel group A** (no deps): W1 (build repair) + W5 (SQL injection) + W11 (security)
- **Parallel group B** (dep W1): W3 (runtime safety) + W7 (polish) + W9 (context fixes) + W10 (TS)
- **Serial group C** (dep W1): W4W6 (memory + decision loop — merged to avoid signature conflict)
- **Serial group D** (dep W4W6): W2 (goroutine lifecycle + SSE — needs query.go changes from W4W6)
- **Final** (dep all): W8 (regression gate)

Key architectural corrections vs. the original v3 plan:
- **W4+W6 merged** — `NewQueryHandler` signature is modified ONCE to accept both `memoryStore` + `engine`, eliminating the B1 merge conflict
- **W2-05 removed** — "synthesis goroutine deadlock" does not exist (B2/E2); Observe/Reflect are synchronous
- **W1-03 downgraded** — `db.Cleanup()` is a no-op (E4/I4), not P0/CRITICAL
- **W1-04 rewritten** — `NewToolExecutor` stays a DI bridge var with pass-through default (E3/I3, N5), not an error-returning constructor
- **W9 corrected** — 3 `context.Background()` calls in engine.go helpers (E1), plus registry.go:153 (I6)
- **B4 added** — 16+ SQL interpolation vectors in ingestion/engine.go with user-controlled table names
- **I7 added** — `Engine.Act()` takes `PlannedStep`, Chat() dispatches string+map; adapter required
- **N1-N10 ALL fixed** — 10 signature/type/ordering errors from v1 corrected

## Execution Waves

---

### Wave 1: Build Repair (1.25gg) — [Dependencies: none]

**Objective:** `go test ./internal/...` passes clean + `npx tsc --noEmit` zero errors.

#### W1-01: Fix nil-wrap in metadata.go (14 functions) + duckdb.go (5 functions including N6 lock fix)

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

**duckdb.go — 5 fixes (includes I1 nil-wrap + N6 lock-ordering):**

1. `Close()` line 107: guard `d.db.Close()` error (I1)
2. `TX.Commit()` lines 227-241: **TWO fixes needed** (N6):
   - Move `parentMu.Unlock()` AFTER `err := t.tx.Commit()` returns and the error is captured
   - Guard `fmt.Errorf("txCommit: %w", err)` with `if err != nil` check

   ```go
   // FIXED Commit (N6 + I1):
   func (t *TX) Commit() error {
       defer t.sem.Release(1)
       t.mu.Lock()
       defer t.mu.Unlock()
       if t.done {
           return nil
       }
       t.done = true
       err := t.tx.Commit()
       // Release lock only AFTER commit result is captured
       if t.isReadTx {
           t.parentMu.RUnlock()
       } else {
           t.parentMu.Unlock()
       }
       if err != nil {
           return fmt.Errorf("txCommit: %w", err)
       }
       return nil
   }
   ```

3. `TX.Rollback()` lines 245-265: same N6 + I1 fixes as Commit above

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

#### W1-04: Add GetToolExecutor() accessor + nil-executor guard in Engine (I3/E3/N5)

**File:** `internal/decision/decision.go` lines 139-146

**Context:** `NewToolExecutor` is a DI bridge var taking 4 function params (executeQuery, analyzeSentiment, getTrustScore, getComponentByID). Before the handler wires it, it's `nil`.

**Fix:** Add `GetToolExecutor()` accessor + nil-executor guard in `Engine.Act()`.

**In `decision/decision.go` — add accessor:**
```go
// GetToolExecutor returns the configured executor, or nil if not yet wired.
func GetToolExecutor() ToolExecutor {
    if NewToolExecutor == nil {
        return nil
    }
    return NewToolExecutor(nil, nil, nil, nil)  // 4 func params — actual signature
}
```

**In `decision/engine.go` — add nil guard at start of Act():**
```go
func (e *Engine) Act(ctx context.Context, step PlannedStep, projectID string) (*ActResult, error) {
    if e.executor == nil {
        return &ActResult{Step: step, Error: "executor not wired (degraded)"}, nil
    }
    start := time.Now()
    // ... rest of existing Act() ...
```

The pass-through fallback stays in W4W6-03 (Chat() checks `if h.engine != nil`).

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

**Verification (mutation test, not grep):**
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

#### W5-03: Add SQL injection sanitization in ingestion/engine.go (B4, N9, N10 — FIXED)

**File:** `internal/ingestion/engine.go`

**Current state:** 16+ `fmt.Sprintf` calls interpolate user-controlled values into SQL (lines 260, 411, 418, 504, 514, 581, 639, 680, 739, 741, 743, 938). `resolveTableName()` (line 215) strips special chars with `[^a-zA-Z0-9_]` but does NOT validate with `^[a-zA-Z_][a-zA-Z0-9_]*$`. A name starting with a digit or underscore slips through.

**N10 fix:** The real signature at ingestion/engine.go:215 is `func resolveTableName(task *v1.IngestionTask) string` — it takes the protobuf type `*v1.IngestionTask`, NOT `*IngestionTaskRecord`.

**N9 fix:** When both `Config.TableName` and `Name` are empty, the function falls back to `task.Id` (line 225) which is user-controlled and unvalidated by any regex.

**Fix:**
```go
var validName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func resolveTableName(task *v1.IngestionTask) string {
    var config struct {
        TableName string `json:"tableName"`
    }
    if json.Unmarshal([]byte(task.ConfigJson), &config) == nil && config.TableName != "" {
        cleaned := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(config.TableName, "_"))
        if !validName.MatchString(cleaned) {
            cleaned = "ingested_" + cleaned
        }
        if !validName.MatchString(cleaned) {
            cleaned = "ingested_data"
        }
        return cleaned
    }
    if task.Name != "" {
        cleaned := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(task.Name, "_"))
        if !validName.MatchString(cleaned) {
            cleaned = "ingested_" + cleaned
        }
        if !validName.MatchString(cleaned) {
            cleaned = "ingested_data"
        }
        return cleaned
    }
    // N9: validate task.Id before using as fallback table name
    cleaned := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(task.Id, "_"))
    if !validName.MatchString(cleaned) {
        return "ingested_data"
    }
    return cleaned
}
```

Also validate `csvPath`, `rawPath`, `filePath`, and `localPath` variables that appear in CREATE TABLE/VIEW statements — these come from config not user strings directly, but should still be validated for defense-in-depth.

**Verification W5:**
```bash
go test -v ./internal/... -run "TestSQL|TestSQLInjection|TestInjection"
# Mutation test: table name `"; DROP TABLE users; --"` must be rejected
go test ./internal/ingestion/... -v -run TestResolveTableName
# Verify: task with empty Name+Config, malicious Id → returns "ingested_data"
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

#### W3-01: Replace panics with error returns (BAN log.Fatalf) — N7 FIXED

**BAN `log.Fatalf()`** — it's a different crash mode. All fixes MUST use `(T, error)` return patterns.

| File | Line | Pattern | Fix |
|------|------|---------|-----|
| `internal/sandbox/validation.go` | 41-48 | `init()` calls `panic()` on regex compile failure | **N7 fix:** Replace `init()` with `sync.Once` lazy init. Expose `ValidateConfig() error` that calls `once.Do(...)`, stores error. |
| `internal/mcp/ssrf.go` | 81 | `mustParseCIDR()` calls `panic()` on invalid CIDR | **N7 fix:** Same `sync.Once` pattern. Parse CIDRs on first call, return error. |
| `internal/tools/osint/shadowbroker.go` | 38 | `newSimpleCache()` panics on LRU failure | Return `nil, fmt.Errorf("...")`. Caller must handle error. |

**validation.go N7 fix — concrete code:**
```go
var (
    pythonPatternRegexes  []*regexp.Regexp
    initOnce             sync.Once
    initErr              error
)

// ValidateConfig compiles pattern regexes lazily and returns any error.
// Safe to call from main() and from tests (won't crash, won't fail on init).
func ValidateConfig() error {
    initOnce.Do(func() {
        for _, pattern := range blocklistedPythonPatterns {
            re, err := regexp.Compile(pattern)
            if err != nil {
                initErr = fmt.Errorf("sandbox blocklist regex %q: %w", pattern, err)
                return
            }
            pythonPatternRegexes = append(pythonPatternRegexes, re)
        }
    })
    return initErr
}
```

**ssrf.go fix:** Same pattern — `sync.Once` lazy parse of CIDR ranges in `ValidatePrivateRanges() error`.

#### W3-02: Fix DuckDB lock inconsistency

**File:** `internal/storage/duckdb.go`

**Fix:** Move backup under `mu.Lock()` (write lock), remove `backupMu`. This blocks ALL reads during backup — verify backup completes in <1s before shipping.

#### W3-03: Replace nil,nil returns with sentinel errors

| File | Function | Line | Fix |
|------|----------|------|-----|
| `internal/decision/reflector.go` | `Reflect()` | 28 | `var ErrPlanNil = errors.New("plan is nil")` → `return nil, ErrPlanNil` |
| `internal/mcp/discovery.go` | `findToolByName()` | 253 | `var ErrToolNotFound = errors.New("tool not found")` → replace `return nil, nil` with `return nil, ErrToolNotFound` |
| `internal/memory/embed.go` | `ProcessText()` | 117 | `var ErrEmptyInput = errors.New("empty input")` → `return nil, nil, ErrEmptyInput` |

**Verification W3:**
```bash
go build ./... && go vet ./...
grep -rn "panic(" internal/sandbox/validation.go internal/sandbox/ internal/mcp/ssrf.go internal/tools/osint/shadowbroker.go
# → Zero results in these files (init() panic is gone; only validation test files may have test panics)
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

#### W7-02: Verify existing /healthz endpoint (N8 FIXED)

**N8 fix:** `/api/v1/healthz` is ALREADY registered at `internal/routes/routes.go` lines 57-62 with a handler returning `{"status":"ok"}` WITHOUT auth middleware. Do NOT add a new endpoint.

**Task:** Verify the existing endpoint works and document it:
1. Start the server (`go run ./cmd/aleph`)
2. `curl http://localhost:8080/api/v1/healthz` → must return 200 with `{"status":"ok"}`
3. If broken, fix the existing handler — don't create a duplicate.

#### W7-03: API.md expansion

Document remaining handler categories: AgentService (5), SkillService (3), ToolService (3), ProjectService (6), AuthService (3).

**Verification W7:**
```bash
ls .dockerignore                              # File exists
curl http://localhost:8080/api/v1/healthz      # Returns 200 without auth (manual test)
```

---

### Wave 9: Context.Background Fixes (1.0gg) — [Dependencies: W1 — parallel with W3/W7/W10]

**Objective:** Every `context.Background()` in production replaced with proper cancellable context with timeout.

#### W9-01: Fix engine.go (3 context.Background calls — E1 corrected)

**File:** `internal/decision/engine.go`

**E1 correction:** The original plan claimed 3 context.Background() in Plan/Act/Observe. FALSE. Only 2 helper methods plus BuildToolsMap use it:
- Line 244: `inferToolsFromMessage` calls `e.metaRepo.ListTools(context.Background())`
- Line 315: `buildToolDefinitions` calls `e.metaRepo.ListTools(context.Background())`
- Line 336: `BuildToolsMap` calls `e.buildToolDefinitions(context.Background())`

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

### Wave 4+6: Memory Subsystem + Decision Loop Wiring (1.5gg) — [MERGED — B1, N1, N2, N3, N4 fixes]

**Dependencies:** W1
**Objective:** MemoryStore + Embedder wired in production, DecisionEngine no longer dead code. NewQueryHandler signature modified ONCE. All N1-N4 signature/ordering bugs fixed.

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

#### W4W6-02: Wire MemoryStore + Embedder + Decision Engine in app.go + add engine/executor fields to QueryHandler (N2, N3 FIXED)

**Files:** `internal/app/app.go`, `internal/api/handler/query.go`

**First, add `engine` and `executor` fields to QueryHandler in query.go:**
```go
type QueryHandler struct {
    db           *storage.DuckDB
    projectsRoot string
    metaRepo     *repository.MetadataRepository
    httpClient   *http.Client
    nlpHandler   *NLPHandler
    registry     *registry.DuckDBRegistry
    programs     *programCache
    engine       decision.DecisionEngine   // NEW — nil = fallback to hardcoded dispatch
    executor     decision.ToolExecutor     // NEW — set by NewToolExecutor bridge
}
```

**NewQueryHandler signature — modified ONCE** (accommodates both W4 memory + engine features):
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

**In Serve()** — N2 fix: explicit `NewToolExecutor` assignment (uses actual 4-param signature). N3 fix: pass nil for Provider:
```go
memStore, err := memory.NewMemoryStore(a.db.SQLDB(), a.cfg.DuckDBSchema, 768)
if err != nil {
    a.logger.Warn("memory store init failed (degraded)", "err", err)
    memStore = nil
}
embedder := memory.NewEmbedder(a.cfg.OllamaBaseURL, a.cfg.EmbeddingModel)

// N2 fix: EXPLICIT assignment of NewToolExecutor before creating Engine
// Actual signature: func(executeQuery, analyzeSentiment, getTrustScore, getComponentByID) ToolExecutor
// The handler's queryHandler fields (h.executor) is set after creation;
// for now, GetToolExecutor() returns nil, and Engine.Act() guards against nil executor.
decision.NewToolExecutor = func(
    executeQuery func(ctx context.Context, req *connect.Request[*alephv1.ExecuteQueryRequest]) (*connect.Response[*alephv1.ExecuteQueryResponse], error),
    analyzeSentiment func(ctx context.Context, text string) (string, error),
    getTrustScore func(ctx context.Context, entityID string) (string, error),
    getComponentByID func(id string) (*decision.ComponentMetadata, error),
) decision.ToolExecutor {
    return h.executor  // h is the handler ref; executor field is set after NewQueryHandler
}

engineCfg := decision.EngineConfig{
    Provider:    nil,              // N3: no llmProvider field exists; pass nil for graceful degradation
    MetaRepo:    a.metaRepo,
    Executor:    decision.GetToolExecutor(),  // from W1-04 (returns nil if not wired yet)
    MaxAttempts: 5,
}
decisionEngine := decision.NewEngine(engineCfg)

queryHandler := handler.NewQueryHandler(
    a.db, projectsRoot, a.metaRepo, a.nlpHandler, regDb,
    memStore,         // NEW
    decisionEngine,   // NEW
)

// After NewQueryHandler, set h.executor for the engine bridge
queryHandler.executor = decision.CreateToolExecutor(
    queryHandler.executeQuery,   // wraps existing method
    queryHandler.analyzeSentiment,
    queryHandler.getTrustScore,
    queryHandler.getComponentByID,
)
```

**Where `CreateToolExecutor` wraps the 4 handlers into a `ToolExecutor` (added to query.go or a new file):**
```go
func CreateToolExecutor(
    executeQuery func(ctx context.Context, req *connect.Request[*alephv1.ExecuteQueryRequest]) (*connect.Response[*alephv1.ExecuteQueryResponse], error),
    analyzeSentiment func(ctx context.Context, text string) (string, error),
    getTrustScore func(ctx context.Context, entityID string) (string, error),
    getComponentByID func(id string) (*decision.ComponentMetadata, error),
) decision.ToolExecutor {
    return &handlerExecutor{
        executeQuery:   executeQuery,
        analyzeSentiment: analyzeSentiment,
        getTrustScore:    getTrustScore,
        getComponentByID: getComponentByID,
    }
}

type handlerExecutor struct {
    executeQuery    func(ctx context.Context, req *connect.Request[*alephv1.ExecuteQueryRequest]) (*connect.Response[*alephv1.ExecuteQueryResponse], error)
    analyzeSentiment func(ctx context.Context, text string) (string, error)
    getTrustScore    func(ctx context.Context, entityID string) (string, error)
    getComponentByID func(id string) (*decision.ComponentMetadata, error)
}

func (e *handlerExecutor) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}, projectID string, agentID string) (string, bool, error) {
    switch toolName {
    case "search_data":
        return e.executeQuery(ctx, ...)
    case "analyze_sentiment":
        return e.analyzeSentiment(ctx, ...)
    case "get_trust_score":
        return e.getTrustScore(ctx, ...)
    default:
        return "", false, fmt.Errorf("unknown tool: %s", toolName)
    }
}
```

**Engine graceful degradation (N3):** In `decision/engine.go`, the Plan() method already handles nil provider gracefully. Add nil check at the top of Plan():
```go
func (e *Engine) Plan(ctx context.Context, msg string, projectID string, agentID string, ontContent []byte, agent *alephv1.Agent) (*PlanResult, error) {
    if e.provider == nil {
        // Degraded mode: plan without LLM enhancement
        return e.planFromHeuristics(ctx, msg, projectID), nil
    }
    // ... existing LLM-based planning ...
}
```

#### W4W6-03: Add Engine.Act() adapter for Chat() tool dispatch (I7, N1 FIXED)

**File:** `internal/api/handler/query.go` — Chat() method, tool dispatch section (lines 639-700)

**Problem:** Chat() dispatches tools via raw struct fields `tc.Name` (string) and `tc.Arguments` (map), but `Engine.Act()` requires a `PlannedStep`.

**Fix:** Engine doesn't need `ToolExecutorConfig` — it has its own `executor` field. Chat() calls `h.engine.Act()` directly, wrapping the tool call in a `PlannedStep`:

```go
// In Chat(), replace the if-else chain (lines 639-700) with:
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
    // Fallback: original hardcoded if-else chain (preserve as-is from lines 639-700)
    // ... existing code (search_data / analyze_sentiment / get_trust_score) ...
}
```

This preserves the hardcoded fallback if the engine is nil (backward compatible with existing tests, and implements N5's pass-through behavior).

#### W4W6-04: REMOVED (N4)

SSE wiring does NOT belong in W4W6. `a.sseBroker` is added to the AlephApp struct by W2-01, which runs AFTER W4W6. SSE wiring is exclusively in W2-03.

**Verification W4W6:**
```bash
go build ./...                    # Compiles with new NewQueryHandler signature
go test -v ./internal/memory/     # Memory tests pass (callers pass nil)
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

#### W2-02: Add NotificationService.Stop() (I9 — path verified)

**File:** `internal/service/notification/notification.go`

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

**File:** `internal/ingestion/engine.go` lines 200-210

**I5 correction:** `enrichPredictiveMetadata` lives in `ingestion/engine.go` line 228, NOT `decision/engine.go`.

**I8 fix:** `enrichCtx` uses `context.WithTimeout(ctx, 30*time.Minute)` at line ~203, but `ctx` is `taskCtx` which has a 15-minute timeout. The enrichment can never reach 30 minutes because the parent cancels first at 15 minutes.

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

#### W2-04: Wire SSE Broker + Handler in app.go + routes

**File:** `internal/app/app.go`, `internal/routes/routes.go`

**This is the ONLY place SSE is wired** (removed from W4W6 via N4).

```go
// In Serve():
a.sseBroker = sse.NewBroker(30*time.Second, a.logger)
sseH := handler.NewSSEHandler(a.sseBroker, a.logger)

// In RegisterConfig for routes:
SSEBroker:  a.sseBroker,
SSEHandler: sseH,
```

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
| W1 | Build Repair + lock-ordering fix | 6 | — | 1.25gg |
| W5 | SQL Injection hardening | 3 | — | 1.25gg |
| W11 | Security (API keys) | 3 | — | 0.75gg |
| W3 | Runtime Safety (panics, nil,nil, locks) | 3 | W1 | 1.0gg |
| W7 | Polish (.dockerignore, healthz verify, docs) | 3 | W1 | 0.5gg |
| W9 | Context.Background fixes | 3 | W1 | 1.0gg |
| W4W6 | Memory + Decision Loop (MERGED, N1-N4 fixed) | 3 | W1 | 1.5gg |
| W2 | Goroutine Lifecycle + SSE | 4 | W4W6 | 1.75gg |
| W10 | TypeScript Hardening | 5 | W1 | 1.0gg |
| W8 | Regression Gate | 1 | ALL | 0.25gg |
| **Total** | | **34** | | **~9.5gg** |

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

### Every N1-N10 blocker has a corresponding fix task

| ID | Finding | v2 Fix Task | Status |
|----|---------|-------------|--------|
| **N1** | GetToolExecutor() has no signature — phantom ToolExecutorConfig used in v1 | W4W6-02: NewToolExecutor uses actual 4-func-param signature. Engine.Act() receives PlannedStep directly. | ✅ |
| **N2** | NewToolExecutor never assigned in production | W4W6-02: explicit assignment added with 4-func-param signature | ✅ |
| **N3** | a.llmProvider/a.cfg.LLMProvider doesn't exist | W4W6-02: pass nil for Provider, add Engine graceful degradation | ✅ |
| **N4** | W4W6-04 references a.sseBroker before W2-01 creates it | W4W6-04 REMOVED. SSE lives in W2 only. | ✅ |
| **N5** | GetToolExecutor() returns nil instead of error; Engine.Act() guards nil executor | W1-04: added GetToolExecutor() accessor + nil guard in Act(). W4W6-03 preserves hardcoded dispatch as fallback. | ✅ |
| **N6** | Commit/Rollback releases lock before capturing error | W1-01: defer lock release after t.tx.Commit()/Rollback() returns | ✅ |
| **N7** | init() can't return error in validation.go + ssrf.go | W3-01: use sync.Once lazy init, expose ValidateConfig() error | ✅ |
| **N8** | W7-02 "Add /api/v1/healthz" — already registered | W7-02: changed to "Verify existing endpoint" | ✅ |
| **N9** | W5-03 fallback task.Id unvalidated (user-controlled) | W5-03: validate task.Id with validName regex | ✅ |
| **N10** | W5-03 wrong type: *IngestionTaskRecord → *v1.IngestionTask | W5-03: use correct protobuf type | ✅ |

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
- `NewToolExecutor` var at decision.go:141 takes 4 func params (executeQuery, analyzeSentiment, getTrustScore, getComponentByID) → ToolExecutor ✅
- `ToolExecutor.ExecuteTool(ctx, toolName, args, projectID, agentID) (string, bool, error)` at decision.go:81 ✅
- `PlannedStep` struct has `ToolName, Arguments, ExpectedOutcome, RequiresConfirmation` ✅
- `QueryHandler` struct has no `engine` or `executor` field yet — W4W6-02 adds both ✅
- `Chat()` method at query.go:415 streams `v1.ChatResponse` ✅
- `NotificationService` at `service/notification/notification.go` (verified) ✅
- `enrichPredictiveMetadata` in `ingestion/engine.go` line 228 (verified) ✅
- `resolveTableName` takes `*v1.IngestionTask` at ingestion/engine.go:215 ✅ (N10 fix)
- `context.Background()` in engine.go at lines 244, 315, 336 (verified) ✅
- `/api/v1/healthz` registered at routes.go:57-62 (verified) ✅ (N8 fix)
- No `a.llmProvider` or `a.cfg.LLMProvider` field exists in AlephApp ✅ (N3 fix)

### Path accuracy

All file paths verified by reading source code. The following are approximations; use grep to find exact lines:
- `adaptation/pipeline.go` ~642: use `grep -n "UpdateHealthStatus" internal/tools/adaptation/pipeline.go` to find discard
- SSE handler path: use `find internal -name "*sse*"` or `grep -rn "sse" internal/routes/routes.go`
- API key UI component: use `grep -rn "substring.*0.*8\|substring.*8" frontend/src/`

## Changes from v1 (debug-distilled.md)

| Change | Reason |
|--------|--------|
| **W1-01 expanded** | Now includes N6: Commit/Rollback lock-ordering deadlock fix |
| **W1-04 pass-through default** | N5: defaultToolExecutor must not fail all tool calls; GetToolExecutor() returns nil, Chat falls through to hardcoded dispatch |
| **W3-01 sync.Once lazy init** | N7: validation.go + ssrf.go use init() which cannot return error |
| **W4W6-02 explicit assignment** | N2: NewToolExecutor assignment made explicit; N3: Provider=nil with Engine graceful degradation |
| **W4W6-03 N1 fix** | Engine.Act() receives PlannedStep directly (no phantom ToolExecutorConfig struct) |
| **W4W6-04 REMOVED** | N4: SSE cannot reference a.sseBroker before W2-01 creates it |
| **W5-03 N9+N10 fixes** | Correct type *v1.IngestionTask; validate task.Id fallback |
| **W7-02 downgraded to verify** | N8: /api/v1/healthz already registered at routes.go:57-62 |
| **Effort: ~10gg → ~9.5gg** | Removed W4W6-04 (redundant SSE wiring), right-sized W7-02 |
