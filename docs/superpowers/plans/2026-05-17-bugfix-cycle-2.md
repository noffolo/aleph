# Bugfix Cycle 2 — Nil Guards + Error Swallowing

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Fix verified production bugs found via systematic code analysis: nil-guard panics in AlephApp.Close() and swallowed rollback errors in DuckDB storage.

**Architecture:** Two independent waves: A (app.go nil guards — P0) and B (duckdb.go rollback error logging — P1). Both are safe, focused changes fixing concretely identified issues.

**Tech Stack:** Go 1.26, slog, DuckDB, safeident

---

### Task A1: Add nil guards to eng.Close(), pg.Close(), db.Close() in AlephApp.Close()

**Files:**
- Modify: `internal/app/app.go:471-479`

**Bug:** AlephApp.Close() at lines 471, 474, 477 calls `a.eng.Close()`, `a.pg.Close()`, `a.db.Close()` without nil guards. If NewAlephApp fails after partial initialization (e.g., NewDuckDB succeeds but NewPostgres fails), Close() panics on nil pointer dereference.

**Already documented:** TestClose_PanicsOnNilEngine at app_test.go:202-217 expects this panic.

- [ ] **Step 1: Update test to expect nil-guard instead of panic**

Edit `internal/app/app_test.go` line 202-217: remove TestClose_PanicsOnNilEngine and add TestClose_AllNilFields (which already exists at lines 202-217), verifying it succeeds instead of panicking.

- [ ] **Step 2: Add nil guards to app.go:471-479**

Replace lines 471-479:

```go
	if a.eng != nil {
		if err := a.eng.Close(); err != nil {
			errs = append(errs, fmt.Errorf("engine close: %w", err))
		}
	}
	if a.pg != nil {
		if err := a.pg.Close(); err != nil {
			errs = append(errs, fmt.Errorf("postgres close: %w", err))
		}
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("duckdb close: %w", err))
		}
	}
```

- [ ] **Step 3: Verify tests pass**

Run:
```
go test -count=1 ./internal/app/... -v 2>&1 | tail -20
```

Expected: `TestClose_AllNilFields` now PASSES (previously the test documented the panic but expected it).

- [ ] **Step 4: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go
git commit -m "fix(app): add nil guards to eng/pg/db Close() calls

AlephApp.Close() at app.go:471-477 called a.eng.Close(), a.pg.Close(),
and a.db.Close() without nil guards, causing panic on partially
initialized AlephApp instances. Add nil checks before each Close()
call matching the existing nlpHandler nil guard pattern at line 468.

TestClose_AllNilFields now verifies safe shutdown instead of expecting panic."
```

---

### Task B1: Log rollback errors instead of swallowing in duckdb.go

**Files:**
- Modify: `internal/storage/duckdb.go:228,260,325`

**Bug:** Three sites in duckdb.go use `_ = tx.Rollback()` which silently discards rollback errors. If rollback fails (e.g., connection lost), the error is invisible. `log/slog` is already imported in this file.

- [ ] **Step 1: Fix line 228**

Replace:
```go
_ = tx.Rollback()
```
With:
```go
if rerr := tx.Rollback(); rerr != nil {
    slog.Warn("rollback after set schema failed", "error", rerr)
}
```

- [ ] **Step 2: Fix line 260**

Replace:
```go
_ = tx.Rollback()
```
With:
```go
if rerr := tx.Rollback(); rerr != nil {
    slog.Warn("rollback after set schema in read tx failed", "error", rerr)
}
```

- [ ] **Step 3: Fix line 325**

Replace:
```go
_ = tx.Rollback()
```
With:
```go
if rerr := tx.Rollback(); rerr != nil {
    slog.Warn("rollback after exec error", "action", rerr)
}
```

- [ ] **Step 4: Verify tests pass**

Run:
```
go test -count=1 ./internal/storage/... 2>&1 | tail -5
```

Expected: Existing tests still pass (rollback swallows are now logged instead of discarded).

- [ ] **Step 5: Commit**

```bash
git add internal/storage/duckdb.go
git commit -m "fix(storage): log rollback errors instead of swallowing in duckdb.go

Three sites used _ = tx.Rollback() to silently discard rollback errors.
Replace with slog.Warn so failed rollbacks are visible in logs without
changing control flow (rollback error is secondary to the original error)."
```

---

### Task C1: Full suite verification + Commit + Push + GitNexus

- [ ] **Step 1: Run full verification**

```bash
go build ./... && go vet ./... && go test -count=1 ./... 2>&1 | tail -20
```

- [ ] **Step 2: Push to main**

```bash
git push origin main
```

- [ ] **Step 3: Update GitNexus**

```bash
npx gitnexus analyze
```

- [ ] **Step 4: Final status**

```bash
npx gitnexus status
```
