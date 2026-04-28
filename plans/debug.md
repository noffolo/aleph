# P3 — Remediation Plan (Debug Wave) — REVISED v3

> **Audit date:** 27 Apr 2026  
> **Base commit:** e9ce39e (W0-02 env hardening)  
> **Build:** `go build ✅` | `go test ❌ 35 FAIL (5 packages)` | `npx tsc ❌ 37 errors` | `vite build ✅`  
> **Estimate:** ~11 days (v2: 8gg + ~3gg per new W9/W10/W11)  
> **Reviewers:** Oracle, Metis, Momus, Ultrabrain — all objections integrated.

---

## Pre-Execution Verification (0.25gg)

**MUST do before W1:**

```bash
go test ./... -v > /tmp/go-test-full.log 2>&1
npx tsc --noEmit -p frontend/tsconfig.json > /tmp/tsc-errors.log 2>&1
```

Read both logs. Catalog exact failures per package. If >15 non-nil-wrap failures found, descope W6 (Decision Loop) and use time for test stabilization.

---

## Summary of Findings

| Area | Issues Found | Severity |
|------|-------------|----------|
| **SQL injection** | query.go (1 vector) + memory/store.go (9 fmt.Sprintf vectors) | CRITICAL |
| **Test failures** | 35 tests fail across 5 packages (handler 5, integration 5, registry 1, adaptation ~4, repo ~20) | CRITICAL |
| **tsc errors** | 37 total: 8 production (6 proto↔store, 1 cva, 1 asList) + 29 test deps/style | HIGH |
| **metadata.go nil-wrap** | **15** functions wrap err with `%w` without `if err != nil` check → `%!w(<nil>)` at runtime | HIGH |
| **DuckDB.Close() nil-wrap** | Same pattern missed at duckdb.go:107 | HIGH |
| **SSE** | broker + handler nil in routes.go (TODO), nil-deref crash on `/api/v1/events` | CRITICAL |
| **NotificationService** | 3 workers with NO Stop(), NO stop channel — goroutine leak | CRITICAL |
| **healthChecker** | goroutine started, Stop() exists but NEVER called (local var in Serve()) | CRITICAL |
| **discoveryEngine** | goroutine started, Stop() exists but NEVER called (local var in Serve()) | CRITICAL |
| **watchSidecar** | fire-and-forget goroutine, never awaited in Close() | HIGH |
| **enrichPredictiveMetadata** | fire-and-forget goroutine (engine.go:202), no lifecycle | HIGH |
| **DecisionEngine** | interface + impl exist but NEVER wired in production (only in test) | HIGH |
| **Memory subsystem** | NewMemoryStore + NewEmbedder NEVER instantiated in production | MED |
| **Panics** | 3 panics in init()/helpers — crash on malformed config, no recovery | MED |
| **Sentinel errors** | Reflect(), ProcessText() return `nil, nil` with ZERO production callers | LOW |
| **as any in production** | 25 occurrences: 17 in .tsx, 8 in useAppActions.ts (plan claimed ~17) | MED |
| **as any in test files** | ~97 in __tests__/ (low priority) | LOW |
| **db.Cleanup() in Chat loop** | Called after EVERY tool iteration (query.go:711) — nukes DuckDB state mid-conversation | CRITICAL |
| **NewToolExecutor nil var** | Package-level `var` that will panic if called before assignment (decision.go:141) | HIGH |
| **Config gaps** | No DuckDBSchema or EmbeddingModel fields | LOW |
| **BackupInterval dead config** | Declared in struct but NEVER populated in LoadConfig() | LOW |
| **Duplicate Agent interface** | AgentsView.tsx lines 10-30 defines its own `interface Agent` that shadows store type | HIGH |
| **context.Background() misuse** | 5 places: engine.go (3), registry.go (1), memory/store.go (1) — no cancellation, no timeout | CRITICAL |
| **Goroutine leak: enrichment in decision** | `go e.enrichPredictiveMetadata()` at engine.go:202 — no ctx, no wg | HIGH |
| **Goroutine leak: synthesis** | Potential deadlock from non-responsive observer (Observer is keyword-only, no LLM fallback) | HIGH |
| **API key in localStorage** | Plaintext key in localStorage, no httpOnly cookie option | MED |
| **SSE API key in query param** | API key exposed in URL query params (logged, cached) | MED |
| **API key masking** | Key masking shows first 8 chars (should be last 4) | MED |
| **AbortController missing** | No AbortController for any API calls in frontend | MED |
| **DuckDB semaphore inconsistency** | sem.Acquire vs AcquireContext in different paths | MED |
| **useEffect missing deps** | App.tsx useEffect missing dependencies | MED |
| **fromProto is no-op** | adapters.ts:7 casts `as any` instead of actual mapping | MED |
| **Zod schemas built but unused** | Frontend has Zod validation schemas that are never called | LOW |

---

## W1 — Build Repair (0.75gg)

**Dipendenze:** Nessuna  
**Obiettivo:** `go test ./internal/...` passa pulito + `npx tsc --noEmit` zero errori.

### W1-01 Fix nil-wrap in metadata.go (15 functions) + duckdb.go (1)

**Pattern fix:** Replace `return fmt.Errorf("name: %w", err)` with:
```go
if err != nil { return fmt.Errorf("name: %w", err) }
return nil
```

**15 functions in internal/repository/metadata.go:**

| Line | Function | Impact |
|------|----------|--------|
| 90 | `UpdateTaskProgress` | Called from ingestion engine |
| 125 | `CreateTask` | Called from ingestion handler |
| 139 | `DeleteTask` | Called from ingestion handler |
| 154 | `SaveChatMessage` | Called from Chat() every turn |
| 299 | `CreateAgent` | Called from agent handler |
| 304 | `DeleteAgent` | Called from agent handler |
| 409 | `CreateTool` | 13 callers, 8 propagate un-checked |
| 417 | `UpdateToolCode` | 5 callers in repair + adaptation |
| 425 | `UpdateHealthStatus` | 3 callers (1 discards err) |
| 442 | `DeleteTool` | Called from tool handler |
| 505 | `CreateSkill` | Called from skill handler |
| 519 | `DeleteSkill` | Called from skill handler |
| 553 | `CreateAPIKey` | Called from auth handler |
| 567 | `DeleteAPIKey` | Called from auth handler |

**Also fix duckdb.go:107:**
```go
// Before:
return fmt.Errorf("duckdbClose: %w", d.db.Close())
// After:
if err := d.db.Close(); err != nil {
    return fmt.Errorf("duckdbClose: %w", err)
}
return nil
```

**Also fix adaptation/pipeline.go:642:** `_ = s.metaRepo.UpdateHealthStatus(...)` → log the error.

### W1-02 Fix tsc --noEmit (8 production errors)

| # | File | Line | Error | Fix |
|---|------|------|-------|-----|
| 1-6 | `AgentsView.tsx`, `SkillsView.tsx`, `ToolsView.tsx` | 34-52 | Store types missing index signature | Add adapter functions at handler↔store boundary: `func protoAgentFromRecord(r AgentRecord) *v1.Agent` — NOT a shared mapper library |
| 7 | `button.tsx` | 2 | `class-variance-authority` module not found | `npm install class-variance-authority` |
| 8 | `SlideOverPanel.tsx` | 28 | `NodeListOf.asList` not on DOM types | Replace with `Array.from(nodeList)` |

**Note:** 29 additional tsc errors in test files (`__tests__/`) are non-blocking for CI because `vite build` succeeds. Deferred to future polish.

### W1-03 Fix db.Cleanup() in Chat loop (P0)

**query.go:711** — `h.db.Cleanup()` called after EVERY tool iteration inside the Chat loop. This closes DuckDB connections mid-conversation, potentially killing ongoing queries.

**Fix:** Move `Cleanup()` to AFTER the Chat loop exits (post-turn), not after each tool iteration. Or remove it entirely — `Cleanup()` is called on server shutdown via `app.Close()`.

### W1-04 Promote NewToolExecutor from nil var to constructor

**decision.go:141** — Package-level `var NewToolExecutor func(ToolExecutorConfig) ToolExecutor` is nil. Will panic if called before assignment.

**Fix:** Replace with a proper constructor function:
```go
func NewToolExecutor(cfg ToolExecutorConfig) ToolExecutor {
    // return default implementation
}
```

### W1-05 Fix duplicate Agent interface in AgentsView.tsx

**AgentsView.tsx lines 10-30** defines a local `interface Agent { id: string; name: string; ... }` that shadows the store's Agent type. This causes proto↔store type mismatch in the 6 tsc errors (W1-02 items 1-6).

**Fix:** Remove the local interface. Use the store type directly. This is a PREREQUISITE for W1-02 items 1-6.

**Verifica W1:**
```bash
go test ./internal/...         # All PASS, zero FAIL
npx tsc --noEmit -p frontend/tsconfig.json  # Zero errors (8 prod + 29 test resolved)
```

---

## W6 — Decision Loop Integration (0.5gg) ← MOVED UP

**Dipendenze:** W1  
**Obiettivo:** DecisionEngine non più dead code. **Scoped down** — the Engine already exists at `internal/decision/engine.go` with Plan/Act/Observe/Reflect/Admit. Only the wiring + Chat() delegation is needed (~2h).

### W6-1 Wire DecisionEngine in QueryHandler.Chat()

**Current:** Hardcoded if-else tool dispatch in Chat() (lines 639-700). Three hardcoded tools (search_data, analyze_sentiment, get_trust_score) with raw `map[string]interface{}` definitions.

**Fix:** Keep Chat() as the streaming orchestrator but delegate tool dispatch to `Engine.Act()`:
```go
// In Chat(), replace:
// switch toolName { case "search_data": ... }  (lines 639-700)
// With:
toolResult := h.engine.Act(ctx, toolName, args, projectID, agentID)
```

The `Engine` already exists at `internal/decision/engine.go` with the `ToolExecutor` interface. The Chat() method's if-else chain is the ONLY place tools are dispatched — replacing it delegates to the engine.

**Note:** This wires ONLY the Act phase. Plan/Observe/Reflect/Admit remain as engine methods available for future integration. The full Decision Loop (all 5 phases in Chat()) is a separate concern.

### W6-2 Assign ToolExecutor in app.go

```go
// Where QueryHandler is created, pass the engine reference
queryHandler := handler.NewQueryHandler(/*...*/, h.engine)
```

**Verifica W6:**
```bash
go build ./...                                          # Compiles
go test ./internal/decision/ -v                         # Existing engine tests pass
```

---

## W2 — Goroutine Lifecycle + SSE Wiring (1.75gg)

**Dipendenze:** W1 + W6  
**Obiettivo:** Zero goroutine leak on shutdown, SSE funzionante.  
**CRITICAL:** W2 modifies Chat() context (SSE wiring), W6 modifies Chat() dispatch (DecisionEngine). W6 first avoids merge conflict on `query.go` lines 630-700.

### W2-01 Store leaked services as AlephApp struct fields

**AlephApp struct** — add:
```go
type AlephApp struct {
    // ... existing fields
    healthChecker    *health.HealthChecker
    discoveryEngine  *mcp.DiscoveryEngine
    notificationSvc  *notification.NotificationService
    sseBroker        *sse.Broker
}
```

**Serve()** — change from local vars to struct fields. ALSO fix `watchSidecar` (app.go:230) — start it as struct field with ctx tracking:
```go
a.healthChecker = health.NewHealthChecker(a.logger, a.metaRepo)
go a.healthChecker.Start(a.ctx)
a.discoveryEngine = mcp.NewDiscoveryEngine(a.ollamaClient, a.metaRepo, a.logger, "")
go a.discoveryEngine.Start(a.ctx)
```

**Close()** — add cleanup for ALL goroutines:
```go
if a.healthChecker != nil { a.healthChecker.Stop() }
if a.discoveryEngine != nil { a.discoveryEngine.Stop() }
if a.notificationSvc != nil { a.notificationSvc.Stop() }
if a.sseBroker != nil { a.sseBroker.Close() }
// watchSidecar and enrichment goroutines stop via a.ctx cancellation
```

### W2-02 Add NotificationService.Stop() + wire in app.go

**CRITICAL:** Current code uses `for job := range s.jobs` (notification.go:39) — a receive-only range that NEVER observes a stop channel. The fix MUST change the loop structure:

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

// REWRITE: for job := range s.jobs → for { select { ... } }
func (s *NotificationService) worker() {
    defer s.wg.Done()
    for {
        select {
        case job, ok := <-s.jobs:
            if !ok { return }
            // ... send webhook ...
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

**Serve():** Store as `a.notificationSvc`. **Close():** Call `a.notificationSvc.Stop()`.

### W2-03 Wire SSE Broker + Handler in app.go + routes

**CRITICAL:** Must set BOTH SSEBroker AND SSEHandler simultaneously — if only one is set, `routes.go:123` will nil-deref panic on `/api/v1/events`.

```go
// In Serve():
a.sseBroker = sse.NewBroker(30*time.Second, a.logger)
sseH := handler.NewSSEHandler(a.sseBroker, a.logger)

// In RegisterConfig:
SSEBroker:  a.sseBroker,
SSEHandler: sseH,
```

### W2-04 Fix enrichment goroutine lifecycle (engine.go:202)

```go
// BEFORE: go func() { e.enrichPredictiveMetadata(...) }()  // fire-and-forget
// AFTER: add to WaitGroup
e.wg.Add(1)
go func() {
    defer e.wg.Done()
    select {
    case <-e.stopCh:
        return
    default:
        e.enrichPredictiveMetadata(...)
    }
}()
```

### W2-05 Fix synthesis goroutine deadlock risk

Observer is keyword-only (no LLM). When synthesis engine waits for observer to respond with "LLM feedback" and no LLM is configured, the goroutine blocks forever. Add timeout or graceful degradation when no observer is available.

**Verifica W2:**
```bash
go test -race ./internal/...  # Zero race warnings
grep -n "go " internal/app/app.go  # Each goroutine must have matching Cleanup
go test -race ./internal/...  # Run ONCE after all goroutine changes (race detector is slow)
```

---

## W3 — Runtime Safety (1gg)

**Dipendenze:** W1  
**Obiettivo:** Zero panics in production code, zero `nil,nil` returns, DuckDB lock fix.

### W3-01 Replace panics with error returns (BAN log.Fatalf)

**BAN `log.Fatalf()`** as a fix — it's a different crash mode. All fixes MUST use `(T, error)` return patterns.

| File | Line | Pattern | Fix |
|------|------|---------|-----|
| `internal/sandbox/validation.go` | 45 | `panic()` in `init()` on regex invalido | Make `ValidateConfig()` return `error` instead of panicking. Caller handles error. |
| `internal/mcp/ssrf.go` | 81 | `mustParseCIDR()` calls `panic()` on invalid CIDR | Add `sync.Once` lazy init that returns `error`. Caller logs fatal on failure. |
| `internal/tools/osint/shadowbroker.go` | 38 | `newSimpleCache()` calls `panic()` on LRU failure | Return `nil, fmt.Errorf("...")`. Caller must handle error. |

### W3-02 Fix DuckDB lock inconsistency

**internal/storage/duckdb.go** — `mu.RWMutex` for reads, `backupMu sync.Mutex` for backup. Reads can proceed during backup (reads hold `RLock`, backup holds `backupMu.Lock()` + `mu.RLock()`).

**Fix:** Move backup under `mu.Lock()` (write lock), remove `backupMu`. **NOTE:** This blocks ALL reads during backup — verify backup completes in <1s before shipping.

### W3-03 Replace nil,nil returns with sentinel errors

| File | Function | Line | Fix |
|------|----------|------|-----|
| `internal/decision/reflector.go` | `Reflect()` | 28 | `var ErrPlanNil = errors.New("plan is nil")` → `return nil, ErrPlanNil` |
| `internal/mcp/discovery.go` | `GetTool()` | 253 | `var ErrToolNotFound = errors.New("tool not found")` → `return nil, ErrToolNotFound` |
| `internal/memory/embed.go` | `ProcessText()` | 117 | `var ErrEmptyInput = errors.New("empty input")` → `return nil, nil, ErrEmptyInput` |

**Verifica W3:**
```bash
go build ./... && go vet ./...   # No panics/unreachable code
grep -rn "panic(" internal/sandbox/validation.go internal/mcp/ssrf.go internal/tools/osint/shadowbroker.go
# → Zero results in these 3 files
go test -race ./internal/storage/...  # DuckDB lock fix verified
```

---

## W4 — Memory Subsystem + Config (0.75gg)

**Dipendenze:** W1  
**Obiettivo:** Memory embedder + store wireati in produzione.

### W4-01 Add DuckDBSchema + EmbeddingModel to config

```go
type Config struct {
    // ... existing
    DuckDBSchema    string  // DuckDB schema for memory tables
    EmbeddingModel  string  // Ollama embedding model name
}
```

Defaults:
```go
viper.SetDefault("DUCKDB_SCHEMA", "main")
viper.SetDefault("EMBEDDING_MODEL", "nomic-embed-text")
```

### W4-02 Wire MemoryStore + Embedder in app.go

**Concrete signature** (not vague "pass to handler"):

```go
// In NewAlephApp() or Serve():
memStore, err := memory.NewMemoryStore(a.db.SQLDB(), a.cfg.DuckDBSchema, 768)
if err != nil {
    return fmt.Errorf("memory store: %w", err)
}
embedder := memory.NewEmbedder(a.cfg.OllamaBaseURL, a.cfg.EmbeddingModel)

// Update NewQueryHandler to accept memoryStore (nil = graceful degradation)
queryHandler := handler.NewQueryHandler(
    a.db, a.cfg.ProjectsRoot, a.metaRepo, nlpHandler, reg,
    memStore,    // NEW — last param, nil for graceful degradation
    h.engine,    // NEW — from W6
)
```

**NewQueryHandler signature update:** Add `memoryStore *memory.MemoryStore` and `engine *decision.Engine` as last parameters with nil default. Existing test callers pass `nil` for now.

**Verifica W4:**
```bash
go test -v ./internal/memory/     # Existing 10 tests pass with wired config
go build ./...                    # Config struct compiles
```

---

## W5 — SQL Injection Hardening (0.5gg)

**Dipendenze:** Nessuna  
**Obiettivo:** Remove fmt.Sprintf SQL injection vectors in query.go AND memory/store.go.

### W5-01 Re-validate lowercased table names (query.go)

**query.go lines 139, 145, 175, 181, 315:** Table names validated with `validName` regex on original case, then `strings.ToLower()` WITHOUT re-validation. A name like `"Users; DROP TABLE"` passes the first check (uppercase) but becomes dangerous after lowercasing.

**Fix:** Validate `lowerObjName` against the same regex:
```go
lowerObjName := strings.ToLower(objName)
if !validName.MatchString(lowerObjName) {
    return connect.NewError(connect.CodeInvalidArgument,
        fmt.Errorf("invalid object name after lowercasing"))
}
```

### W5-02 Fix SQL injection in memory/store.go (9 vectors)

**memory/store.go** — 9 `fmt.Sprintf` calls interpolate user-controlled values (table names, column names) directly into SQL queries. Unlike query.go, there is NO regex validation.

**Fix:** Add the same `validName` regex validation from query.go. All 9 interpolation sites must validate before use:

| Line | Pattern | Fix |
|------|---------|-----|
| ~45 | `fmt.Sprintf("SELECT ... FROM %s", tableName)` | Validate tableName with regex |
| ~72 | Similar pattern | Same fix |
| ~95 | Column name interpolation | Validate column name |
| ~120 | Table name in INSERT | Validate table name |
| ~150 | Table name in DELETE | Validate table name |
| ~175 | WHERE clause column name | Validate column name |
| ~210 | ORDER BY column | Validate column name |
| ~240 | JOIN table name | Validate table name |
| ~270 | Subquery table name | Validate table name |

**Verifica W5:**
```bash
go test -v ./internal/storage/ -run TestSQL   # Injection test
# Add test case: table name `"; DROP TABLE users; --"` must be rejected
grep -n "fmt\.Sprintf.*SELECT\|fmt\.Sprintf.*INSERT\|fmt\.Sprintf.*DELETE" internal/memory/store.go
# → Zero results (all interpolations validated)
```

---

## W9 — Context.Background Fixes (0.75gg) ← NEW

**Dipendenze:** W1  
**Obiettivo:** Every `context.Background()` replaced with a proper cancellable context with timeout.

### W9-01 Fix engine.go (3 places)

**internal/decision/engine.go** — 3 uses of `context.Background()` in Plan/Act/Observe:

```go
// BEFORE:
ctx := context.Background()
// AFTER:
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### W9-02 Fix registry.go (1 place)

**internal/registry/registry.go** — 1 use of `context.Background()`:

```go
// Same pattern: add timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### W9-03 Fix memory/store.go (1 place)

**internal/memory/store.go** — 1 use of `context.Background()`:

```go
// Same pattern: add timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```

**Verifica W9:**
```bash
grep -n "context\.Background()" internal/ internal/decision/ internal/registry/ internal/memory/
# → Zero results
go test -race ./internal/decision/ ./internal/registry/ ./internal/memory/
```

---

## W10 — TypeScript Hardening (1gg) ← NEW

**Dipendenze:** W1, W9  
**Obiettivo:** Reduce `as any` in production to <5, add proper types for API layer.

### W10-01 Fix `as any` in InlineRenderer.tsx (13 occurrences)

**frontend/src/components/terminal/InlineRenderer.tsx** — 13 `as any` casts. Most are in render logic where element types are force-cast. Replace with proper type guards or `unknown` + assertion pattern.

### W10-02 Fix `as any` in useAppActions.ts (8 occurrences)

**frontend/src/hooks/useAppActions.ts** — 8 `as any` casts in API call handling. Replace with typed response wrappers.

### W10-03 Replace no-op fromProto with real mapping

**frontend/src/api/adapters.ts:7** — `fromProto` currently does `record as any as Agent`. Implement actual field-by-field mapping:
```typescript
export function fromProtoAgent(proto: v1.Agent): Agent {
    return {
        id: proto.id,
        name: proto.name,
        description: proto.description,
        // ... all fields mapped
    };
}
```

### W10-04 Wire Zod schemas into form validation

**Unused Zod schemas** exist but are never called in form submission paths. Wire them into AgentFormSlideOver, SkillFormSlideOver, and ToolFormSlideOver before form submission:
```typescript
const result = agentSchema.safeParse(formData);
if (!result.success) {
    setErrors(result.error.flatten().fieldErrors);
    return;
}
```

### W10-05 Add AbortController to all API calls

Every `fetch`/`axios` call should support cancellation via AbortController. Wire a per-request signal to `useEffect` cleanup in all components that fire API calls.

**Verifica W10:**
```bash
grep -c "as any" frontend/src/**/*.tsx frontend/src/**/*.ts
# Target: <10 total in production files (not test files)
npx tsc --noEmit  # Zero errors
```

---

## W11 — Security Hardening (0.75gg) ← NEW

**Dipendenze:** W1  
**Obiettivo:** API key storage compliance, no secrets in URLs/logs.

### W11-01 Replace localStorage API key with secure option

**Current:** API key stored in `localStorage` in plaintext. If an XSS vulnerability exists (e.g., InlineRenderer's dangerouslySetInnerHTML), the key is exfiltratable.

**Fix options (choose one):**
1. **httpOnly cookie** (recommended) — Set by backend on auth, not accessible to JS
2. **SessionStorage** (minimal) — Cleared on tab close, but still accessible to JS
3. **Memory-only** — Key held in Zustand store state, never persisted

### W11-02 Move SSE API key from query param to header

**Current:** SSE endpoint accepts API key as a query parameter (`/api/v1/events?key=...`). This exposes the key in server logs, browser history, referrer headers.

**Fix:** Move to custom HTTP header (`X-API-Key`) for SSE connection. Update the SSE handler to read from header instead of query param.

### W11-03 Fix API key masking (show last 4, not first 8)

**Current:** API key display shows first 8 characters. Security best practice is to show the LAST 4 characters (so attackers see less of the key entropy).

**Fix:** In the API key management UI, change from `key.substring(0, 8) + '...'` to `'...' + key.substring(key.length - 4)`.

**Verifica W11:**
```bash
grep -rn "localStorage" frontend/src/ | grep -v "__tests__" | grep -v "node_modules"
# → Only in auth store (acceptable if non-sensitive usage)
grep -rn "query.*key\|?key=" internal/handler/
# → Zero results (SSE moved to header)
grep -rn "substring.*0.*8\|slice.*0.*8" frontend/src/
# → Zero results (masking shows last 4)
```

---

## W7 — Polish (0.5gg)

**Dipendenze:** W1 (API docs depend on stable signatures)  
**Obiettivo:** Deploy ready.

### W7-01 .dockerignore

```
.git/
node_modules/
frontend/node_modules/
dist/
*.md
.env
.env.example
```

### W7-02 Unauthenticated /healthz endpoint

Add a route BEFORE the auth middleware for k8s liveness probes. `/api/v1/healthz` responds 200 without auth headers. The existing `/healthz` handler already exists — just needs to be registered before the auth middleware group.

### W7-03 API.md expansion

Document remaining handler categories:
- AgentService (5 methods)
- SkillService (3 methods)
- ToolService (3 methods)
- ProjectService (6 methods)
- AuthService (3 methods)

**Verifica W7:**
```bash
ls .dockerignore                              # File exists
curl http://localhost:8080/api/v1/healthz      # Returns 200 without auth
```

---

## W8 — Regression Gate (0.25gg)

**Dipendenze:** W1-W7, W9-W11  
**Obiettivo:** Catch cross-wave regressions before deploy.

```bash
go vet ./... && \
go test -race ./... && \
npx tsc --noEmit -p frontend/tsconfig.json && \
npx vite build && \
go build ./...
```

**Single pass, all must pass.** If any fails, identify which wave introduced the regression and fix before declaring P3 complete.

---

## Riepilogo

| Wave | Cosa | Task | Dipende da | Effort |
|------|------|------|-----------|--------|
| W1 | Build Repair + P0 fixes | 5 | — | 0.75gg |
| W6 | Decision Loop Wiring (scoped) | 2 | W1 | 0.5gg |
| W2 | Goroutine Lifecycle + SSE | 5 | W1+W6 | 1.75gg |
| W3 | Runtime Safety | 3 | W1 | 1gg |
| W4 | Memory Subsystem | 2 | W1 | 0.75gg |
| W5 | SQL Injection | 2 | — | 0.5gg |
| W9 | Context.Background fixes | 3 | W1 | 0.75gg |
| W10 | TypeScript Hardening | 5 | W1, W9 | 1gg |
| W11 | Security Hardening | 3 | W1 | 0.75gg |
| W7 | Polish | 3 | W1 | 0.5gg |
| W8 | Regression Gate | 1 | W1-W7, W9-W11 | 0.25gg |
| **Totale** | | **34** | | **~11gg** |

### Parallel execution opportunities
- **W1 + W5 + W7 + W11** possono andare in parallelo (nessuna dipendenza comune)
- **W3** (Runtime Safety) è indipendente dopo W1 — può correre in parallelo con W2/W4
- **W9** (Context fixes) è indipendente dopo W1 — parallelo con W2/W3/W4
- **W10** (TS hardening) parallelo con W2/W3/W4/W6
- **W8** è l'unico bloccante seriale (dipende da TUTTI i precedenti)

### Key changes from v2

| Change | Reason |
|--------|--------|
| Test failures: 30→35 | Verify scan found adaptation tests + storage tests failing |
| `as any`: 17→25 production | Direct grep found 17 in .tsx + 8 in useAppActions.ts |
| Added W1-05: duplicate Agent interface | Blocks W1-02 items 1-6 (proto↔store type mismatch) |
| Added W2-05: synthesis deadlock fix | Observer is keyword-only with no LLM — blocks forever |
| Added W5-02: SQL injection in memory/store.go | 9 fmt.Sprintf vectors with NO validation (worse than query.go) |
| Added W9: context.Background fixes (NEW wave) | 5 places with no cancellation/timeout |
| Added W10: TypeScript Hardening (NEW wave) | 25 `as any` in prod, no-op fromProto, unused Zod, no AbortController |
| Added W11: Security Hardening (NEW wave) | API key in localStorage, SSE query param, wrong masking |
| Effort: 8gg→11gg | +3gg for 3 new waves (W9/W10/W11) |
