# SPEC-08: Concurrency Hardening — Goroutine Lifecycles, Context Propagation

**Spec version**: 1.0  
**Date**: 2 May 2026  
**Plan reference**: `docs/plans/audit-remediation.md` Wave 4, tasks W4-1 through W4-5  
**Findings addressed**: G1-G6 (concurrency cluster), Q1-Q4 (code quality)  
**Depends on**: `docs/specs/wave1-injection-spec.md` (context propagation), `docs/specs/wave2-database-spec.md` (concurrent database access)  
**Related specs**: `docs/specs/wave4-infra-spec.md` (Docker/nginx shares shutdown path)  
**Status**: ✅ Approved — ready for execution

---

## 1. Goroutine Lifecycle Contract

### Required Pattern for ALL Goroutines

```go
// Every goroutine MUST follow this contract:
go func() {
    defer func() {
        if r := recover(); r != nil {
            logger.Error("goroutine panic", "error", r, "stack", debug.Stack())
        }
    }()
    
    for {
        select {
        case <-ctx.Done():
            logger.Debug("goroutine shutting down", "reason", ctx.Err())
            return
        case <-ticker.C:
            // Do work
        }
    }
}()
```

### Goroutine Inventory — All goroutines must implement this contract

| Location | Goroutine | Status | Fix |
|----------|-----------|--------|-----|
| `internal/sandbox/dev_mode.go:115` | File watcher | NO ctx.Done() | Add context to select loop |
| `internal/middleware/ratelimit.go:55-74` | Cleanup | Partially fixed (has ticker.Stop) | Verify ctx.Done() |
| `internal/storage/duckdb_backup.go:312` | AutoBackup | NO ctx.Done() | Add context cancellation |
| `internal/mcp/discovery.go` | MCP discovery | Silently fails | Add retry + context cancellation |
| `internal/api/handler/chat_session.go` | Chat goroutines | Verify | Audit: all paths call wg.Done() + close channels |
| `internal/app/app.go` | Main server goroutines | Verify | Graceful shutdown within 5s |

---

## 2. Graceful Shutdown

### Contract

```go
// app.Close() must:
// 1. Stop accepting new connections (http.Server.Shutdown with 5s timeout)
// 2. Cancel root context (triggers all child goroutines)
// 3. Wait for all goroutines to finish (sync.WaitGroup with 5s timeout)
// 4. Close database connections
// 5. Close any open files/temp dirs

func (a *App) Close() error {
    // Step 1: Stop HTTP
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    a.httpServer.Shutdown(shutdownCtx)
    
    // Step 2: Cancel root context
    a.cancel()  // Triggers ctx.Done() on all child contexts
    
    // Step 3: Wait for goroutines
    done := make(chan struct{})
    go func() {
        a.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        a.logger.Info("all goroutines completed")
    case <-time.After(5 * time.Second):
        a.logger.Warn("goroutine shutdown timed out", "remaining", runtime.NumGoroutine())
    }
    
    // Step 4: Close databases
    if a.duckDB != nil {
        a.duckDB.Close()
    }
    if a.postgres != nil {
        a.postgres.Close()
    }
    
    // Step 5: Stop notification service
    if a.notificationService != nil {
        a.notificationService.Stop()
    }
    
    return nil
}
```

---

## 3. Context.Background() — Elimination Plan

### Rule: NEVER use `context.Background()` in handler code paths

```go
// ❌ WRONG
func doSomething() {
    conn, _ := db.Conn(context.Background())  // No cancellation, no deadline
}

// ✅ CORRECT
func doSomething(ctx context.Context) {
    conn, _ := db.Conn(ctx)  // Parent context propagates deadline/cancellation
}
```

### Must-Fix Sites

| File | Line | Context | Fix |
|------|------|---------|-----|
| `storage/duckdb_backup.go` | 74 | `ExportDatabase` | Accept ctx parameter from caller |
| `storage/duckdb_backup.go` | 312 | `AutoBackup` | Accept ctx parameter |
| `registry/duckdb_registry.go` | — | Multiple methods | Add ctx parameter to RegisterComponent, UpdateComponentStatus |
| `app/app.go` | — | Startup code | Use `context.WithTimeout(ctx, 30*time.Second)` |
| `ingestion/engine.go` | 296 | Data fetch | Already has ctx from request — verify propagation |

### Allowable Exceptions

```go
// ✅ OK: Startup/shutdown code (no request context available)
func main() {
    ctx := context.Background()
    initDatabases(ctx)
    // ...
}

// ✅ OK: Health checks (background, not per-request)
func healthCheck() {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    db.PingContext(ctx)
}
```

### Target: < 5 remaining `context.Background()` in non-test, non-startup code

---

## 4. json.Unmarshal Error Handling

### Audit Rule: ALL calls checked

```bash
# Find all unmarshal calls
grep -rn "json.Unmarshal\|json.NewDecoder" internal/ --include="*.go" | grep -v "_test.go"

# Expected: EVERY call has:
# if err != nil { return/handle err }
```

### Fix Pattern

```go
// ❌ WRONG
json.Unmarshal(data, &cfg)  // Error silently swallowed

// ✅ CORRECT  
if err := json.Unmarshal(data, &cfg); err != nil {
    return fmt.Errorf("parse config: %w", err)
}
```

### Known Sites to Fix

- `internal/sandbox/validation.go`: Tool metadata JSON parsing
- `internal/dsl/compiler_tool.go`: Template JSON parsing
- Any `_ = json.Unmarshal(...)` → must be fixed

---

## 5. Race on nextID

### Problem

```go
// ❌ RACE CONDITION
var nextID int
func generateID() int {
    nextID++  // Data race: multiple goroutines
    return nextID
}
```

### Fix: Use UUID

```go
// ✅ THREAD-SAFE
import "github.com/google/uuid"

func generateID() string {
    return uuid.New().String()
}
```

### Or: PostgreSQL SERIAL/BIGSERIAL

```sql
-- For sequential IDs, use database-level sequences
ALTER TABLE system_agents 
    ALTER COLUMN id SET DEFAULT gen_random_uuid();
```

---

## 6. Context Deadline Enforcement

### All network operations must have deadlines

```go
// ❌ WRONG
resp, err := http.Get(url)  // No timeout — could hang forever

// ✅ CORRECT
ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
defer cancel()
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := client.Do(req)
```

### Deadlines by Operation Type

| Operation | Deadline | Rationale |
|-----------|----------|-----------|
| DuckDB query | 30s | Complex queries can take time |
| PostgreSQL query | 10s | Metadata queries are simple |
| LLM API call | 5min | LLM inference is slow |
| NLP sidecar | 30s | Text processing |
| Docker API | 10s | Container management |
| HTTP outbound | 30s | External service calls |
| File I/O | 10s | Local filesystem should be fast |

---

## 7. Verification

### Test Coverage

- [ ] `goroutine_leak_test.go` (NEW): Start/stop app 10 times — goroutine count returns to baseline (< 5 leaked)
- [ ] `context_propagation_test.go` (NEW): Static analysis — grep for `context.Background()` in handler code
- [ ] `race_nextid_test.go` (NEW): 100 goroutines generating IDs concurrently with `-race`
- [ ] `shutdown_test.go` (NEW): SIGTERM → app.Close() → all goroutines exit within 5s

### Gate

```
go test -race -count=3 ./...
→ ALL pass, ZERO race detector warnings

go test -run TestShutdown ./internal/app/
→ goroutine count < 10 after shutdown

grep -rn "context.Background()" internal/ --include="*.go" | grep -v "_test.go" | grep -v "main.go"
→ < 5 remaining (all justified: startup, health checks, backup init)

grep -rn "json.Unmarshal\|json.NewDecoder" internal/ --include="*.go" | grep -v "_test.go" | grep -v "if err"
→ 0 matches (all error-checked)
```
