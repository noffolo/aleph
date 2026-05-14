# W2: Panic & Context.TODO — Eliminate Production Risks

> **Branch:** `remediation/w1-w2-type-safety-panic`
> **Target:** Fix 1 real panic + propagate context to 8 sites
> **Stima:** 1gg

## Regole Generali

1. **NON modificare** i 2 panic init-time (giustificati): `config/secrets.go:49`, `storage/context.go:32`
2. **Ogni commit deve mantenere:** `go build ./...` clean, `go vet ./...` clean, `go test -race -count=1 ./...` pass
3. **Per context.TODO()**: propagare il `ctx` dal chiamante — non creare nuovi `context.Background()` o `context.TODO()`
4. **Test esistenti** devono continuare a passare dopo ogni modifica

## Task W2-01: Fix `panic()` in SSRF validator

**File:** `internal/ssrf/validator.go:179`

**Situazione attuale:**
```go
panic("invalid hardcoded CIDR: " + s + ": " + err.Error())
```

**Fix:**
```go
// PRIMA: panic in production path
// DOPO: return error
func validateCIDR(s string) error {
    // ...esistente...
    if err != nil {
        return fmt.Errorf("internal/ssrf/validator: invalid hardcoded CIDR %q: %w", s, err)
    }
    return nil
}
```

**Verifica:**
- `go build ./...` clean
- `go vet ./...` clean
- I test esistenti in `internal/ssrf/` devono ancora passare (se esistono)

## Task W2-02: Propagate context to `context.TODO()` sites

### Sito 1: `internal/tools/registry.go:154`

**Attuale:**
```go
ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
```

**Fix:** Aggiungere `ctx context.Context` al costruttore/setter del registry. Il chiamante in `app.go` che crea il registry deve passare il context di app.

### Sito 2: `internal/health/checker.go:61`

**Attuale:**
```go
ctx, cancel := context.WithCancel(context.TODO())
```

**Fix:** Aggiungere `ctx` al `Start()` o al costruttore di HealthChecker. Il chiamante in `app.go` passa il context di app.

### Sito 3: `internal/storage/duckdb_backup.go:39`

**Attuale:**
```go
tx, err := d.BeginReadTX(context.TODO())
```

**Fix:** Aggiungere `ctx context.Context` come parametro della funzione che chiama BeginReadTX. Propagare dal chiamante più in alto.

### Siti 4-5: `internal/storage/context.go:82` e `96`

**Attuale:**
```go
d.db.ExecContext(context.TODO(), ...)
```

**Fix:** Aggiungere `ctx context.Context` come parametro ai metodi `EnsureProjectSchema()` e `DropProjectSchema()`. I chiamanti (in `internal/storage/` e `internal/handler/`) devono propagare dal loro context.

### Sito 6: `internal/storage/duckdb.go:128`

**Attuale:**
```go
res, err := d.db.ExecContext(context.TODO(), query, args...)
```

**Fix:** Propagare `ctx` dal chiamante. Questa funzione è chiamata da `Execute()` su `*DuckDBStore`. Assicurarsi che `Execute(ctx, ...)` sia firmato con `ctx` e che il chiamante (da query handler) passi il context della richiesta.

### Sito 7: `internal/sandbox/namespace_isolated.go:38`

**Attuale:**
```go
ExecuteIsolated(context.TODO(), "", cmd)
```

**Fix:** Aggiungere `ctx context.Context` come primo parametro di `ExecuteIsolated`. Il chiamante in sandbox handler deve propagare dalla richiesta gRPC.

### Sito 8: `internal/sandbox/container_sandbox.go:78`

**Attuale:**
```go
ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
```

**Fix:** Usare `context.Background()` se è un timer di default per container execution, oppure propagare dalla richiesta gRPC. Verificare se c'è un context disponibile dal chiamante.

## Verifica W2

```bash
# Nessun context.TODO() residuo
grep -rn 'context\.TODO()' internal/ --include='*.go' | grep -v '_test.go' | grep -v vendor
# Output: solo commenti, 0 nel codice produzione

# Build check
go build ./... && go vet ./...

# Test
go test -race -count=1 ./... 2>&1 | tail -10
```
