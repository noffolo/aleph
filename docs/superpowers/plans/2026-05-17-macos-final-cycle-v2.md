# Final macOS Cycle v2 — Residual Coverage Push

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this task-by-task.

**Goal:** Push remaining macOS-testable coverage in internal/app (Close 45.9%→~70%, newTLSClient 41.7%→~60%, helpers ~35%→~50%).

**Architecture:** Close() shutdown ordering tests with nil fields, TLS client cert error, sentinel helpers with edge cases. All pure unit tests — no Postgres, no DuckDB file needed.

---

### Task A2: Push Close() coverage to ~70%

**Files:**
- Modify: `internal/app/app_test.go` — add TestClose_ShutdownOrder and TestClose_PartialInit

**Current Close at app.go:421 — 45.9%:** Cleanup path for 10 fields (db, pg, eng, metaRepo, nlpHandler, ctx, cancel, healthChecker, discoveryEngine, sseBroker). Only nil-guard paths tested. Need to test partial-initialization scenarios.

- [ ] **Step 1: Add partial init Close tests**

```go
func TestClose_PartialInit(t *testing.T) {
    app := &AlephApp{
        ctx: context.Background(),
        cancel: func() {},
        logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
    }
    err := app.Close(context.Background())
    assert.NoError(t, err)
}

func TestClose_ShutdownOrder(t *testing.T) {
    cancelCalled := false
    app := &AlephApp{
        ctx: context.Background(),
        cancel: func() { cancelCalled = true },
        logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
        metaRepo: nil, // nil — hit early return
    }
    assert.NoError(t, app.Close(context.Background()))
    assert.True(t, cancelCalled)
}
```

- [ ] **Step 2: Run tests**

```bash
go test -count=1 -coverprofile=/tmp/app2.cov ./internal/app/...
go tool cover -func=/tmp/app2.cov | grep Close
# Expected: Close around 60-70%
```

- [ ] **Step 3: Commit**

```bash
git add internal/app/app_test.go
git commit -m "test: extend Close coverage with partial init and shutdown order tests"
```

---

### Task B2: Push newTLSClient + helpers coverage

- [ ] **Step 1: Add TLS error scenario and helper edge cases**

```go
func TestNewTLSClient_CertError(t *testing.T) {
    client := newTLSClient("/nonexistent/cert.pem", "/nonexistent/key.pem")
    // Should return a functional http.Client even if cert files don't exist
    // (cert loading happens lazily or returns a default client)
    assert.NotNil(t, client)
    assert.NotNil(t, client.Transport)
}

func TestMakeSentimentHelper_NilHandler(t *testing.T) {
    app := &AlephApp{nlpHandler: nil, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
    fn := app.makeSentimentHelper()
    result, err := fn(context.Background(), "test")
    assert.Error(t, err)
    assert.Equal(t, "NLP sidecar not configured", result)
}

func TestMakeTrustScoreHelper_NilRepo(t *testing.T) {
    app := &AlephApp{metaRepo: nil, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
    fn := app.makeTrustScoreHelper()
    result, err := fn(context.Background(), "tool-1")
    assert.Error(t, err)
    assert.Equal(t, "", result)
}
```

- [ ] **Step 2: Run tests**

```bash
go test -count=1 -cover ./internal/app/...
# Expected: Close ~70%, newTLSClient ~55%, helpers ~50%
```

- [ ] **Step 3: Commit**

```bash
git add internal/app/app_test.go
git commit -m "test: add TLS cert error and helper edge case tests"
```

---

### Task C2: Full verification + coverage report + push

- [ ] **Step 1: Full suite**

```bash
go build ./...
go vet ./...
go test -count=1 -coverprofile=/tmp/final.cov ./internal/app/... ./internal/sandbox/... ./internal/...
npx tsc --noEmit
npx vitest run
```

- [ ] **Step 2: GitNexus reindex**

```bash
npx gitnexus analyze
```

- [ ] **Step 3: Push**

```bash
git push origin main
```
