# W3: Circuit Breaker Wiring + Tool Smoke Tests

**Branch:** `remediation/w1-w2-type-safety-panic` (continuare sullo stesso branch)
**Stima:** 1-2gg
**Dipendenza:** W2 completata (ctx propagation)

---

## W3-01: Circuit Breaker Interceptor nella Middleware Chain

### Situazione attuale

`internal/middleware/circuitbreaker.go` (101L):

- `CircuitBreaker` struct con stato `Closed/Open/HalfOpen`
- Metodi: `NewCircuitBreaker(threshold, cooldown)`, `Execute(fn func() error)`, `State()`, `Reset()`
- **NON è un ConnectRPC interceptor** — è standalone, non ha `WrapUnary`/`WrapStreamingHandler`
- Ha un TODO: `// Wire into HTTP client chain`

`internal/middleware/timeout.go` (124L) — pattern di riferimento:

```go
type TimeoutInterceptor struct {
    config TimeoutConfig
}
func NewTimeoutInterceptor(config *TimeoutConfig) *TimeoutInterceptor { ... }
func (t *TimeoutInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc { ... }
func (t *TimeoutInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc { ... }
func (t *TimeoutInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc { ... }
```

Tutti e 7 gli interceptor esistenti seguono lo stesso pattern struct + WrapUnary + WrapStreamingHandler.

`internal/app/app.go:186-198` — interceptor chain attuale:

```go
interceptors := []connect.HandlerOption{
    connect.WithInterceptors(
        subsystemInterceptor,    // 188
        errorHandler,            // 189
        auditInterceptor,        // 190
        authInterceptor,         // 191
        authRateLimitInterceptor,// 192
        timeoutInterceptor,      // 193
        retryInterceptor,        // 194
        bulkheadInterceptor,     // 195
        trackingInterceptor,     // 196
    ),
}
```

**Circuit breaker MANCANTE** tra bulkhead (195) e tracking (196).

### Cosa fare

#### 1. Aggiungere `CircuitBreakerInterceptor` in `circuitbreaker.go`

Seguire esattamente il pattern di `TimeoutInterceptor`:

```go
// CircuitBreakerInterceptor wraps Connect RPC handlers with circuit breaker protection.
type CircuitBreakerInterceptor struct {
    breaker *CircuitBreaker
}

// NewCircuitBreakerInterceptor creates a new circuit breaker interceptor.
func NewCircuitBreakerInterceptor(threshold int, cooldown time.Duration) *CircuitBreakerInterceptor {
    return &CircuitBreakerInterceptor{
        breaker: NewCircuitBreaker(threshold, cooldown),
    }
}

func (c *CircuitBreakerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
    return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
        var resp connect.AnyResponse
        err := c.breaker.Execute(func() error {
            var innerErr error
            resp, innerErr = next(ctx, req)
            return innerErr
        })
        return resp, err
    }
}

func (c *CircuitBreakerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
    return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
        return c.breaker.Execute(func() error {
            return next(ctx, conn)
        })
    }
}

func (c *CircuitBreakerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
    return next
}
```

**Threshold/cooldown:** Usare 5 failure / 30s cooldown (default di `NewCircuitBreaker`).

**Importante:** Non modificare la struttura `CircuitBreaker` esistente — usare solo il suo metodo `Execute()`.

#### 2. Cablare in `app.go:186-198`

```go
circuitBreakerInterceptor := middleware.NewCircuitBreakerInterceptor(5, 30*time.Second)
```

Aggiungere `circuitBreakerInterceptor` tra `bulkheadInterceptor` (195) e `trackingInterceptor` (196).

### Verifica W3-01

- `go build ./...` ✅
- `go test -race -count=1 ./internal/middleware/...` ✅ (esistono già test per CircuitBreaker)
- `go vet ./...` ✅
- `npx tsc --noEmit` ✅ (nessun file TS modificato)

---

## W3-02: Tool Package Smoke Tests

### Situazione attuale

**Il piano originale diceva "zero test" ma è ERRATO.** Verifica indipendente (Oracle + GitNexus + conteggio diretto) ha confermato:

- `finance/`: 21 test function in 3 file
- `osint/`: 13 test function in 2 file
- `humanecosystems/`: 18 test function in 3 file
- `adaptation/`: 7 test function in 2 file
- `codeflow/`: 40 test function in 1 file
- `synthesis/`: 21 test function in 1 file
- `registry/`: 12 test function in 1 file
- **Totale: ~141 test function in 13 file, 7 package**

### Cosa fare

Verificare che i test esistenti coprano:

1. **Compilazione** — `go build ./internal/tools/...` ✅
2. **Chiamata base non panica** — ogni tool package ha almeno un test che istanzia e chiama il costruttore
3. **Error handling path** — test per casi di errore evidenti

Se i test esistenti sono sufficienti (come da verifica Oracle), **W3-02 è già COMPLETO**.

### Verifica W3-02

```bash
go test -race -count=1 ./internal/tools/...
```

---

## Verifica finale W3

```bash
go build ./...
go vet ./...
go test -race -count=1 ./internal/middleware/...  # CB interceptor tests
go test -race -count=1 ./internal/tools/...        # tool smoke tests
npx tsc --noEmit                                   # frontend non toccato
```
