# Bugfix Cycle 3 — IMAP SSRF Validation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement.

**Goal:** Fix SSRF bypass in IMAP email ingestion — `fetchIMAP()` at `internal/ingestion/engine.go:1031` calls `tls.Dial("tcp", host, ...)` without validating the target host against SSRF rules, allowing SSRF attacks via IMAP source.

**Architecture:** The existing `ssrf.ValidateHostname(hostname, port)` function already validates hosts against private/reserved IP ranges. Insert this check before the `tls.Dial` call, similar to how every other HTTP client in the codebase uses `ssrf.NewClient()`.

**Tech Stack:** Go, ssrf package, IMAP TLS connections

---

### Task A1: Add SSRF validation to fetchIMAP

**Files:**
- Modify: `internal/ingestion/engine.go:1026-1031`
- Test: existing `internal/ingestion/engine_test.go` (add test case)

- [ ] **Step 1: Read existing code context**

The `fetchIMAP` function at engine.go:1026 currently:
```go
func fetchIMAP(host, user, pass, folder string, maxMessages int) ([]emailRow, error) {
    if !strings.Contains(host, ":") {
        host = host + ":993"
    }

    conn, err := tls.Dial("tcp", host, &tls.Config{MinVersion: tls.VersionTLS12})
    if err != nil {
        return nil, fmt.Errorf("IMAP TLS dial: %w", err)
    }
    defer conn.Close()
    // ... rest of IMAP logic
```

The `tls.Dial("tcp", host, ...)` directly dials any host:port without checking if the host is a private/reserved IP. An attacker who can configure an IMAP source could point it at internal services.

The `ssrf.ValidateHostname(hostname, port)` function exists and verifies hostnames against private IPs and reserved ranges. The import path is `github.com/ff3300/aleph-v2/internal/ssrf`.

- [ ] **Step 2: Write test in existing engine_test.go**

The test should verify that `fetchIMAP` with a loopback/private address returns an SSRF validation error before ever attempting a network dial:

```go
// In internal/ingestion/engine_test.go
func TestFetchIMAP_SSRFBlocked(t *testing.T) {
    // Use a loopback address — ssrf.ValidateHostname should reject it
    // even before tls.Dial is attempted.
    results, err := fetchIMAP("127.0.0.1:1993", "user", "pass", "INBOX", 1)
    if err == nil {
        t.Fatal("expected SSRF error for loopback address, got nil")
    }
    if results != nil {
        t.Fatal("expected nil results on SSRF rejection")
    }
    // Verify the error message mentions SSRF
    if !strings.Contains(err.Error(), "SSRF") && !strings.Contains(err.Error(), "private") {
        t.Errorf("expected error mentioning SSRF/private, got: %v", err)
    }
}
```

Run: `go test -count=1 -run TestFetchIMAP_SSRFBlocked ./internal/ingestion/`
Expected: PASS (because `fetchIMAP` will fail with SSRF validation error before dialing)

- [ ] **Step 3: Add SSRF validation to fetchIMAP**

Modify `internal/ingestion/engine.go` to add hostname validation before the TLS dial:

```go
func fetchIMAP(host, user, pass, folder string, maxMessages int) ([]emailRow, error) {
    if !strings.Contains(host, ":") {
        host = host + ":993"
    }

    hostname, port, err := net.SplitHostPort(host)
    if err != nil {
        return nil, fmt.Errorf("invalid IMAP host: %w", err)
    }
    if err := ssrf.ValidateHostname(hostname, port); err != nil {
        return nil, fmt.Errorf("IMAP SSRF: %w", err)
    }

    conn, err := tls.Dial("tcp", host, &tls.Config{MinVersion: tls.VersionTLS12})
```

Note: the imports list already includes `net` (used elsewhere), `fmt`, `strings`, `tls`. The `ssrf` import needs to be added if not present.

- [ ] **Step 4: Verify test passes**

Run: `go test -count=1 -run TestFetchIMAP_SSRFBlocked ./internal/ingestion/`
Expected: PASS (SSRF check aborts before dial attempt)

- [ ] **Step 5: Verify full suite**

```bash
go test -count=1 ./internal/ingestion/...
go vet ./internal/ingestion/...
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/ingestion/engine.go internal/ingestion/engine_test.go
git commit -m "fix: add SSRF validation to IMAP email ingestion

fetchIMAP() used raw tls.Dial without validating the target host
against SSRF rules, allowing SSRF attacks via IMAP source configuration.
Add ssrf.ValidateHostname check before dialing.

Fixes: an attacker with source-config access could target internal services."
```
