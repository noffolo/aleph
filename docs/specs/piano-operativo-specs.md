# Specifiche Tecniche — Piano Integrato 120gg (v2.0 Production Grade)

> **Allineato a**: `piano-final-integrato.md` (1 Maggio 2026, 4 fasi, 48 task, 168.5gg-uomo)
> **Stato build attuale**: `go build` ✅ | `go test -race` ✅ | `npx tsc` ✅ | `npx vite build` ✅
> **Completion score**: 72% (funzionante ma non production-ready)
> **Divergenze risolte** rispetto a specs precedente (v1 90gg): gVisor differito, label corrette, multi-tenancy scope ridotto, 38 task aggiunti, ship gates, review findings, nuovi task B04.5/B08/A10

---

## FASE 1: Safety Net (G1-30, 49gg-uomo)

**Obiettivo**: 14 CRITICAL → 0. Sandbox isolata. Auth JWT. PAORA cablato.

### A01 — Sandbox Isolamento Reale (G1-10, 6gg)

```
SandboxConfig {
    IsolationMode: "container"                 // default, NO "gvisor" (differito backlog)
                                               // NO "none" — sempre container isolation
    RestrictedPath: "/usr/bin:/bin"            // fisso, non os.Getenv
    MaxInputSize: 1 * 1024 * 1024             // 1MB
    CPUQuota: 1.0                              // 1 core
    MemoryLimit: 512MB                         // 512MB
    ProcessLimit: 50                           // max processi
    NetworkBlocked: true                       // default: --network=none
    TimeoutSeconds: 30                         // context timeout su esecuzione
}
```

**Fix specifici**:
- Bloccare TUTTE le varianti di flag interattivi: `--interactive=always`, `--tty=yes`, `-it`, `--interactive`, `--tty`, `-i`, `-t`
- Validazione Go/Python via parser AST (non regex): `go/parser` e Python `ast` module
- Blocklist Python estesa: `imaplib`, `urllib`, `requests`, `socket`, `os`, `subprocess`, `shutil`, `sys`
- Docker SDK: `--network=none --read-only --cap-drop=ALL --security-opt=no-new-privileges`

**Test** (penetration test):
- `--interactive=always` → DEVE fallire con errore
- `--tty=yes` con pipe a shell → DEVE fallire
- STORED_XSS/EXFIL pattern TUTTI coperti
- OOM kill con input > MemoryLimit → verificato
- CPU throttle con fork bomb → verificato

**Nota**: Spike gVisor G1-3. Se non fattibile → commit container-only (decisione G3).

---

### A02 — Auth System Rewrite (G5-18, 5gg)

**Pre-requisito**: Migration runbook G5-7 (validato PRIMA di toccare codice). Deve coprire:
1. Migrazione API key esistenti senza rompere integrazioni
2. Periodo coesistenza API key header + JWT cookie
3. Rollback step-by-step
4. Test accettazione per ogni step

```
// Session token JWT
type SessionToken struct {
    UserID    string   `json:"sub"`
    ProjectID string   `json:"pid"`
    Role      string   `json:"role"`   // "admin" | "user" | "agent"
    ExpiresAt int64    `json:"exp"`
    Scopes    []string `json:"scopes"`
}

// Endpoints
POST /api/v1/auth/login    → Set-Cookie: session=<JWT>; HttpOnly; Secure; SameSite=Strict; Path=/
POST /api/v1/auth/logout   → Clear-Cookie: session
POST /api/v1/auth/session  → Valida sessione, restituisce dati utente (NO api key in chiaro)

// Agent API keys in list responses
{ "api_key": "ale_abc...1234" }  // ultimi 4 caratteri
POST /api/v1/agent/{id}/reveal-key → full key (richiede admin auth)

// skipAuth → exact path match table (NON strings.Contains)
var skipAuthPaths = map[string]bool{
    "/api/v1/auth/session": true,
    "/readyz":              true,
    "/livez":               true,
    "/metrics":             true,
}
```

**Backward Compatibility**:
- `X-Aleph-Api-Key` header: accettato con deprecation warning in response header
- Session token priority #1 (se presente, API key ignorata)
- Rimozione API key support pianificata per v2.1
- Confronto chiavi: constant-time (`subtle.ConstantTimeCompare`)

**Test**:
- JWT: creazione, validazione, expiry, refresh
- API key header backward compat (deprecation warning presente)
- Agent key masking: solo ultimi 4 char visibili in list
- Reveal-key: solo admin
- skipAuth: exact match, non substring
- Constant-time comparison verificato

---

### A03 — SQL Injection Fix (G1-5, 2gg)

**Sequencing**: Completato PRIMA di B01 (PAORA core). I test integration PAORA non devono operare su codice vulnerabile.

**Fix specifici**:
- `query.go`: TUTTE le query → prepared statements (`$1`, `$2`) al posto di `fmt.Sprintf`
- `scopeQuery` e `info_schema` query → parametrizzate
- `memory/store.go`: 9 siti string concat → parametrizzati
- DSL compiler → prepared statements
- `validName()` regex applicata a TUTTI i parametri stringa: `[a-zA-Z0-9_-]+`
- Filter objects: whitelist campi consentiti per sorting/filtering
- CI linter: bloccare `fmt.Sprintf` in file che contengono SQL

**Test**:
- SQLMap su tutti gli endpoint query → 0 vulnerabilità
- Input malevoli: `'; DROP TABLE--`, `1 OR 1=1`, Unicode homoglyph → tutti bloccati
- CI gate: `gosec` senza esclusioni su query.go → 0 findings

---

### A04 — Hardcoded Secrets Removal (G5-10, 2gg)

- Rimuovere Postgres DSN default `postgres:postgres` → deve fallire con errore chiaro se non configurato
- `.env.example`: placeholder sostituiti con istruzioni esplicite (non valori fake)
- `KEY_ENCRYPTION_KEY`: da env var a file-based `/run/secrets/key_encryption_key`
- Docker secrets per `POSTGRES_PASSWORD` via `env_file` con permessi 600
- **Opportunità 4.7 integrata**: Docker Secrets pattern documentato

---

### A05 — Network & Auth Hardening (G10-18, 3gg)

- CSP: `ws://localhost:*` → `ws://localhost:8080` (specifico). Aggiungere `strict-dynamic`, `base-uri 'self'`, `form-action 'self'`
- CSRF: richiedere Origin/Referer validi. Bloccare richieste senza Origin non GET/HEAD. SameSite=Lax default.
- X-Forwarded-For: trusted proxy list (non accettare da qualsiasi IP)
- `skipAuth`: exact match table (già fissato in A02)
- **Opportunità 4.4**: CSRF → SameSite=Lax
- **Opportunità 4.5**: CSP hardening

---

### B01 — PAORA Core Fix (G3-18, 5gg)

**Reflect Unification (G3-7)**:
- `Engine.Reflect()` → chiama `DefaultReflector` internamente
- `DefaultReflector` classifica gap: `CONFIDENCE_GAP`, `EXECUTION_GAP`, `CONTEXT_GAP`, `TOOL_GAP`
- `TrustDelta` calcolato da: confidence drop + consecutive failures + tool errors
- Soglie escalation: >0.3 = repeat, >0.5 = revise, >0.7 = admit
- **Opportunità 2.1 integrata**: Reflection Engine unificato

**Plan-Act Connector (G7-12)**:
```
type PlanStep struct {
    Tool    string
    Input   map[string]any
    Depends []int  // indici step precedenti
}

type PlanResult struct {
    Steps     []PlanStep
    Rationale string
}
```
- `Act()` DEVE eseguire `PlanResult.Steps` in ordine (rispettando Depends)
- Se Executor assente → cerca in Registry → fallback a dispatch
- **Opportunità 2.2 integrata**: Plan-Act Connector

**nil Provider Fix (G12-14)**:
```
type EngineConfig struct {
    Provider    LLMProvider          // REQUIRED (no nil)
    MetaRepo    ToolRepository       // REQUIRED
    Executor    ToolExecutor         // OPTIONAL (fallback a dispatch)
    Registry    ToolRegistry         // OPTIONAL
    MaxAttempts    int               // default 5, min 1, max 20
    TrustThreshold float64           // default 0.7
    GapThreshold   int               // default 3
    PlanMaxSteps   int               // default 5
}
```
- `app.go` DEVE passare provider valido (es. `llm.NewProvider("ollama", ...)`)
- `Act()` gestisce `query_dispatch` come fallback
- `EngineConfig.Validate()`: Provider e MetaRepo REQUIRED, zero values = errore

**ChatSession Hardening (G14-18)**:
- `MaxAttempts` da EngineConfig (non hardcoded 5)
- Nil-check su plan in Reflect (evitare panic)
- Admit: retry su primo errore (non terminare immediatamente)

---

### B02 — Decision Engine Test Suite (G10-22, 4gg)

**Unit test per fase PAORA**:
```
TestPlan_GeneratesValidSteps    → input diversi producono []PlanStep validi
TestAct_ExecutesPlannedSteps    → esecuzione in ordine rispettando Depends
TestObserve_CalculatesTrustDelta → TrustDelta ≠ 0 con errori reali
TestReflect_ClassifiesGap       → classificazione CONFIDENCE vs EXECUTION gap
TestAdmit_Retries               → retry su errore; termina dopo MaxAttempts
```

**Integration test**:
```
TestPAORA_FullCycle         → Plan→Act→Observe→Reflect→Admit con mock provider
TestPAORA_DegradedMode      → nil Provider → fallback funzionante
TestPAORA_TrustEscalation   → 3+ gap consecutivi → Admit finale
```

**Property-based test**:
- "Per qualsiasi piano valido, Reflect produce feedback non-vuoto"
- "Act non restituisce mai panic con input malformato"

**Mock**: `MockLLMProvider` ritorna piani predefiniti per test deterministici.

---

### B03 — GNN Predictor Training (G15-25, 3gg)

- Training offline su dati storici (tool usage, agent relationships)
- `IsTrained()` → true dopo training
- `TrustDelta` dal GNN integrato in Engine.Observe
- Metrics: precision@k, recall@k
- Threshold: predictions solo se confidence > 0.7
- **Opportunità 8.1 integrata**: GNN training con dati reali

---

### C01 — AbortController Integration (G1-7, 2gg)

```
// Hook pattern
function useAbortableEffect(
    effect: (signal: AbortSignal) => Promise<void>,
    deps: unknown[]
) {
    useEffect(() => {
        const controller = new AbortController();
        effect(controller.signal);
        return () => controller.abort();
    }, deps);
}
```

**Fix specifici**:
- `App.tsx` data loading: AbortController + cleanup su unmount
- Chat history loading: AbortController
- `useAppActions`: cancellare richieste pending su navigazione
- SSE reconnect: non martellare su 401 (auth check prima). Max 5 retry, backoff esponenziale 1s-2s-4s-8s-16s.
- Ogni fetch/chiamata ConnectRPC: AbortController al mount, passare signal, AbortError gestito silenziosamente

---

### C02 — Auth Fix Frontend (G3-12, 2gg)

- `useOntologyActions`: aggiungere auth headers (non raw fetch senza auth)
- Sostituire `fetch()` raw con chiamate ConnectRPC dove possibile
- API key non visibile in Zustand DevTools (persist senza sensitive data)
- ToolForm, SkillForm: migrare a ConnectRPC client (da raw REST)
- **Opportunità 3.3 integrata**: ConnectRPC come unico transport

---

### C03 — Type Safety Sprint (G7-22, 10gg)

**Nota**: Stima ricalibrata da Oracle (20-25gg reali → 10gg, scope ridotto a file produzione).

Priority 1 (2gg): Sostituire `assertType()` con `ZodSchema.parse()` reale
```
// PRIMA: const project = assertType<Project>(data)   // identity function, no-op
// DOPO:  const project = ProjectSchema.parse(data)   // runtime validation
```

Priority 2 (2gg): Unificare tipi: `store/types.ts` → `schemas/index.ts`
```
// Eliminare store/types.ts, usare solo schemas/index.ts
export type Project = z.infer<typeof ProjectSchema>
```

Priority 3 (1gg): Zod schemas mancanti per Scenario, ToolAnomaly

Priority 4 (3gg): Rimuovere `as unknown as` cast in file produzione (42 → 0)
```
// pattern: data as unknown as SomeType → SomeSchema.parse(data)
// pattern: (e as unknown as AxiosError) → ErrorSchema.parse(e) o type guard
```

Priority 5 (2gg): `any` reduction nei 14 file produzione con commenti eslint
```
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- [ragione specifica]
```

---

### C04 — Error Handling Centralizzato (G10-20, 2gg)

- Empty catch blocks (x3): `catch {}` → `catch (e) { errorService.handle(e) }`
- `handleError` duplicato: unificare in singleton ErrorService con subscriber pattern
- DataSourceForm `JSON.parse` senza try-catch: aggiungere guard
- `alert()`/`confirm()` in SetupWizard/SettingsView → modal React
- **Opportunità 7.1 integrata**: Errori human-readable in italiano
```
ErrorService.handle(error, {
    subsystem: "chat" | "query" | "ingestion" | "auth",
    userMessage: "Messaggio in italiano comprensibile"
})
```

---

### C05 — State Management Fix (G12-22, 3gg)

- `setProjectContext`: atomic update (non resettare parzialmente)
- `cancelStream`: non mutare state in setter (side effect puro)
- Set serialization in copilotSlice: `Array` invece di `Set` (JSON-safe)
- `useCursorPagination`: chiudere stale closure (useRef per valori correnti)
- `fetchTools`: includere `projectId` come parametro
- SSE `lastEventId` in Zustand store (non module-level, evitare race tra istanze)

---

### Ship Gate 1 (G30)

**Check obbligatori**:
- [ ] `go test -race -count=1 ./...` ✅ (0 FAIL)
- [ ] `npx tsc --noEmit` ✅ (0 errori produzione)
- [ ] `npx vite build` ✅
- [ ] 14/14 CRITICAL risolte e verificate con test
- [ ] Sandbox: penetration test → 0 bypass
- [ ] Auth: JWT login/logout/session; API key header backward compat
- [ ] PAORA: Act esegue PlannedStep; Reflect usa DefaultReflector
- [ ] AbortController in tutte le chiamate di rete frontend
- [ ] Migration runbook auth validato
- [ ] **Security re-check**: OWASP ZAP scan → 0 CRITICAL/HIGH

**Rollback**:
1. Sandbox container non stabile → feature flag `ENABLE_SANDBOX_ISOLATION=false`, revert a L0 allowlist
2. Auth JWT non funzionante → spegnere JWT middleware, revert a solo API key header
3. PAORA regressioni → feature flag `PAORA_USE_LEGACY_REFLECT=true`

---

## FASE 2: Stability Engine (G20-55, 40.5gg-uomo)

**Obiettivo**: 25 HIGH → 0. Test suite. CI/CD blindato. DuckDB/LLM robusti. Load test base.

### A06 — CI/CD Blindatura (G20-24, 1.5gg)

```yaml
# CI: pipefail obbligatorio
- name: Test
  run: |
    set -o pipefail
    go test -race -count=1 ./... 2>&1 | tee test-output.log

# Deploy: test gate esplicito
- name: Deploy Gate
  needs: [test, build]
  run: |
    go test -race -count=1 ./...
    npx tsc --noEmit
    npx vitest run
    npx vite build

# go vet in CI
- name: Vet
  run: go vet ./...

# .dockerignore
node_modules/
.git/
*.md
test/
**/*_test.go
frontend/node_modules/
frontend/src/**/*.test.*
```

**Opportunità 5.2 integrata**: Pipeline CI a prova di fallimento.

---

### A07 — Docker Ottimizzazione (G22-28, 2gg)

```
# Multi-stage build
FROM golang:1.24-alpine AS builder       // non bullseye
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o aleph .

FROM alpine:3.20
COPY --from=builder /app/aleph /aleph
```

- `cache-from`/`cache-to` per layer caching in CI
- `docker-compose.yml`: healthcheck backend + `depends_on` con condition
- Target: image < 500MB (da 990MB)

---

### A08 — Alertmanager + Monitoring (G25-35, 3gg)

```
Receivers:
  - Slack webhook (canale #aleph-alerts)
  - Email SMTP (admin@org)

Alert rules:
  - Uptime < 99.9% per 5min
  - Error rate > 1% per 5min
  - NLP sidecar offline per 30s
  - DuckDB lock contention > 10s
  - LLM cost budget > 80%

Grafana dashboard:
  - Request latency (p50/p95/p99)
  - Error rate per endpoint
  - Tool execution count/success rate
  - DuckDB: query count, lock wait time, active connections
  - LLM: token count, call count, cost
```

---

### A09 — Deploy Pipeline (G35-50, 4gg)

**A09a — Docker Compose Production (G35-42, 3gg)**:
```bash
#!/bin/bash
# deploy.sh — set -euo pipefail
# 1. docker compose config validation
# 2. env file validation (required vars check)
# 3. secrets management (docker secret create)
# 4. docker compose up -d --wait
# 5. healthcheck verification loop (max 60s)
# 6. log rotation config (json-file, max-size 10m, max-file 3)
```

**A09b — Kubernetes Option (G42-50, 1gg)**: Documentazione Helm chart. NON implementazione. K8s deploy post-v2.0.

**A09c — Bare-metal Guide (G42-45)**: Documentazione setup manuale per ambienti senza container.

---

### A10 — Load Testing Anticipato (G40-55, 3gg)

**Nota**: Spostato da Fase 4 (G75) a Fase 2 per feedback Metis. Serve SUBITO per validare fix DuckDB e rate limiter.

```javascript
// k6 test script
export default function() {
    // Query test: 50% traffic
    http.post('/api/v1/query', JSON.stringify({query: 'SELECT COUNT(*) FROM tools'}))
    
    // Chat test: 20% traffic  
    http.post('/api/v1/chat', JSON.stringify({message: 'analyze system health'}))
    
    // Ingestion test: 20% traffic
    http.post('/api/v1/ingestion', JSON.stringify({source: 'test'}))
    
    // Tool execution: 10% traffic
    http.post('/api/v1/tool/execute', JSON.stringify({tool: 'echo', input: 'test'}))
}
```

- Target Fase 2: 500 req/s p95 < 1s
- Memory profiling con pprof durante load test
- Goroutine leak detection (`runtime.NumGoroutine()` trend)

---

### B04 — DuckDB Concurrency Fix (G22-30, 3gg)

```
type DuckDB struct {
    mu  sync.RWMutex
    sem *semaphore.Weighted  // max concurrent connections
    db  *sql.DB
}

func (d *DuckDB) BeginTx(ctx context.Context) (*sql.Tx, error) {
    if err := d.sem.Acquire(ctx, 1); err != nil {
        return nil, fmt.Errorf("semaphore acquire: %w", err)
    }
    d.mu.Lock()  // Lock, non RLock — transazione scrive
    tx, err := d.db.BeginTx(ctx, nil)
    if err != nil {
        d.mu.Unlock()
        d.sem.Release(1)
        return nil, err
    }
    return &txWrapper{Tx: tx, mu: d.mu, sem: d.sem}, nil
}

type txWrapper struct {
    *sql.Tx
    mu  sync.Locker
    sem *semaphore.Weighted
}

func (w *txWrapper) Commit() error {
    defer w.mu.Unlock()
    defer w.sem.Release(1)
    return w.Tx.Commit()
}
```

**Principi**:
- Lock ordering: `mu` preso PRIMA di operazioni DB, rilasciato DOPO
- Transazioni: `Lock` (scrittura), non `RLock`
- `QueryRowContext` → `QueryRowContextOrError` (torna errore, non nil)
- VSS INSERT: DELETE + INSERT in singola transazione
- txWrapper già implementato (Oracle conferma) → verificare, non ricostruire
- **Opportunità 2.4 integrata**: DuckDB concurrency model review

---

### B04.5 — LLM Budget & Cost Controls (G24-28, 2gg) — NUOVO

```
type LLMBudget struct {
    MaxCallsPerHour  int     // env: LLM_MAX_CALLS_HOUR (default 1000)
    MaxCostPerDay    float64 // env: LLM_MAX_COST_DAY (default 50.0)
    currentHourCalls atomic.Int64
    currentDayCost   atomic.Float64
}

func (b *LLMBudget) CanCall(provider string, estimatedTokens int) bool {
    if b.currentHourCalls.Load() >= int64(b.MaxCallsPerHour) {
        return false
    }
    cost := estimateCost(provider, estimatedTokens)
    if b.currentDayCost.Load() + cost > b.MaxCostPerDay {
        return false
    }
    return true
}

// Alert: superamento 80% soglia → notify admin
// Reset: hourly → cron ogni ora; daily → cron midnight UTC
```

---

### B05 — Rate Limiter Memory Safety (G28-34, 2gg)

```
type RateLimitStore struct {
    mu      sync.RWMutex
    entries map[string]*rateEntry
    maxSize int           // default 100000
    ttl     time.Duration // default 1h
}

// Goroutine cleanup ogni 10 minuti
// Double-check locking pattern per evitare race
// LRU eviction quando maxSize raggiunto
// Response header: X-RateLimit-Remaining
// X-Forwarded-For: trusted proxy list validation
```

**Opportunità 2.3 integrata**: Rate Limiter sliding window.

---

### B06 — LLM Provider Robustezza (G32-42, 3gg)

```
// Timeout configurabile
http.Client{Timeout: 30 * time.Second}

// Retry con exponential backoff
for attempt := 0; attempt < 3; attempt++ {
    resp, err := client.Do(req)
    if err == nil { return resp, nil }
    time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
}

// Circuit breaker
type CircuitBreaker struct {
    state       State  // CLOSED, OPEN, HALF_OPEN
    failures    atomic.Int64
    lastFailure atomic.Time
}
// HALF_OPEN → jitter, max 1 richiesta (no thundering herd)
// OPEN → fail fast con errore

// Provider registry: nomi sconosciuti → errore esplicito (non nil)
// EngineConfig.Validate() su startup
```

**Opportunità 2.5 integrata**: Circuit breaker pattern completo.

---

### B07 — MCP Discovery Reliability (G35-45, 3gg)

```
// healthLoop: WaitGroup per graceful shutdown
func (d *Discovery) Start(ctx context.Context) {
    d.wg.Add(1)
    go func() {
        defer d.wg.Done()
        d.healthLoop(ctx)
    }()
}

func (d *Discovery) Stop() {
    d.cancel()
    d.wg.Wait()  // aspetta completamento graceful
}

// Retry su discovery fallita con backoff esponenziale
// SSRF validation centralizzata (HTTP client unico, non creare per ogni richiesta)
// URIs validation su input (IsURL schema check)
```

**Opportunità 2.8 integrata**: HealthChecker context fix — non sovrascrivere cancel.

---

### B08 — NLP Sidecar Watchdog (G40-48, 2gg) — NUOVO

```
func (n *NLPSidecar) watchSidecar(ctx context.Context) {
    defer recover()  // recupera da panic nel sidecar
    restartCount := 0
    
    for {
        select {
        case <-ctx.Done():
            return
        default:
            if !n.isHealthy() {
                restartCount++
                if restartCount > 3 {
                    alertManager.Fire("nlp_sidecar_failed", "3+ restarts in 5 min")
                    return
                }
                n.restart()
            }
            time.Sleep(2 * time.Second)  // check ogni 2s (non 10s)
        }
    }
}
```

**Opportunità 6.1 integrata**: NLP Watchdog con auto-restart.

---

### B09 — Python/NLP Validation Fix (G42-50, 2gg)

- Regex validation → parser AST (`go/parser` per Go, Python `ast` per NLP)
- Blocklist import: `eval`, `exec`, `compile`, `__import__`, `open`
- Bloccare eval/exec di import malevoli
- Subprocess PATH: `/usr/bin:/bin` (non `os.Getenv("PATH")`)
- Sandbox verification: timeout per ogni tool execution (da A01)

---

### C06 — Backend Test Coverage (G25-40, 4gg)

- ChatSession: unit test completi (PAORA cycle, errori, degrade mode)
- Sandbox: test per allowlist bypass tentativi
- MCP discovery: test per retry, SSRF validation, shutdown graceful
- HealthChecker: test per context lifecycle
- Ingestion engine: test per runDynamic

---

### C07 — Contract Tests Revival (G30-36, 2gg)

- `//go:build integration` nei file di test (non 'contract' build tag inesistente)
- Test integrazione: connettere a PostgreSQL + DuckDB + NLP reali
- CI: eseguire contract test su trigger manuale (non in ogni push)

---

### C08 — Frontend Test Suite (G35-50, 4gg)

- Vitest: store slices (navigation, auth, copilot, workspace)
- Componenti critici: InlineRenderer, SlideOverPanel, Terminal
- Hooks: test per useStreamSSE, useCursorPagination, useDebounce
- E2E Playwright base: journey auth → query → strumenti → settings

---

### Ship Gate 2 (G55)

**Check obbligatori**:
- [ ] `go test -race -count=1 ./...` ✅ (0 FAIL)
- [ ] `npx tsc --noEmit` ✅ (0 errori produzione)
- [ ] `npx vite build` ✅
- [ ] 25/25 HIGH bugs risolti e verificati
- [ ] CI/CD: pipefail, deploy con test gate
- [ ] Docker: build < 5 min, image < 500MB
- [ ] DuckDB: 0 deadlock in stress test (1000 query parallele per 10 min)
- [ ] LLM: timeout 30s, retry, circuit breaker
- [ ] Rate limiter: 100k IP test, cleanup verificato
- [ ] Backend test coverage > 60%
- [ ] Load test: 500 req/s p95 < 1s
- [ ] **Security re-check**: OWASP ZAP + gosec → 0 HIGH

---

## FASE 3: Feature Completion (G45-85, 45gg-uomo)

**Obiettivo**: PAORA V2 multi-step. Tool package 3/6. UI completa. Integration test > 50.

### A11 — Multi-tenancy Foundations (G45-65, 6gg) — SCOPE RIDOTTO

```
// PostgreSQL: schema per PROGETTO (non tenant)
CREATE SCHEMA IF NOT EXISTS project_<project_id>;
SET search_path TO project_<project_id>, public;

// DuckDB: database condiviso, namespace via prefisso tabella
// NO database per tenant — tabella: project_<id>_tools, project_<id>_memory, etc.

// Rate limiting: per API key (non per tenant)
type APIKeyLimiter struct {
    key     string
    limiter *rate.Limiter
}

// Resource quotas: SOLO soft limits configurabili (enforcement differito Fase 5)
MaxProjects: 10
MaxAgents:   20

// Validazione in creazione risorsa (non quota enforcement complesso)
func (s *Service) CreateProject(req CreateProjectRequest) error {
    count, _ := s.db.CountProjects(req.UserID)
    if count >= s.config.MaxProjects {
        return ErrQuotaExceeded
    }
}
```

---

### A12 — EU Compliance Base (G55-70, 4gg)

- GDPR: data retention policies (TTL configurabile per dati utente)
- Delete cascade: eliminare progetto → rimuovere schema PostgreSQL + tabelle DuckDB
- Audit logging per operazioni admin (`slog.Info("admin_action", ...)`)
- Privacy impact assessment documentato (`docs/privacy-impact-assessment.md`)
- Data residency: documentazione opzioni (non implementazione)

---

### B10 — Multi-Step Tool Execution (G45-60, 5gg)

```
// Plan produce PlanResult con più PlanStep
plan := engine.Plan(ctx, observation)
// plan.Steps = [{Tool: "search", Input: {q: "..."}, Depends: []},
//               {Tool: "analyze", Input: {data: "$step0"}, Depends: [0]},
//               {Tool: "report", Input: {findings: "$step1"}, Depends: [1]}]

// Act esegue in ordine, rispettando Depends
for _, step := range sortedByDeps(plan.Steps) {
    if step.HasUnresolvedDeps() {
        step.ResolveFromPreviousResults()
    }
    result := engine.Act(ctx, step)
    results[stepIndex] = result
}

// Confirmation flow per tool auto-esecuzione
if tool.RequiresConfirmation && result.Confidence < trustThreshold {
    return AskUserConfirmation(tool, step)
}

// Tool result feedback loop → observed state aggiornato
observation = engine.Observe(results)
```

---

### B11 — Tool Package Completamento (G50-70, 8gg) — 3/6

**Da completare**:
1. **Finance (3gg)**: API integration (Alpha Vantage free tier). Funzioni: stock price, historical data, moving averages, RSI.
```go
func (f *FinanceTool) Execute(ctx context.Context, input FinanceInput) (*FinanceOutput, error) {
    url := fmt.Sprintf("https://www.alphavantage.co/query?function=%s&symbol=%s&apikey=%s",
        input.Function, input.Symbol, f.apiKey)
    // response parsing, error handling, rate limiting
}
```

2. **OSINT (3gg)**: API integration (Shodan free tier). Funzioni: IP lookup, domain info, port scan results.
```go
func (o *OSINTTool) Execute(ctx context.Context, input OSINTInput) (*OSINTOutput, error) {
    client := shodan.NewClient(o.apiKey)
    return client.Host(ctx, input.IP)
}
```

3. **HumanEcosystems (2gg)**: Dataset statico o API pubblica. Demographic data, basic indicators.
```go
func (h *HETool) Execute(ctx context.Context, input HEInput) (*HEOutput, error) {
    return h.db.QueryDemographics(ctx, input.Region, input.Indicators)
}
```

**Differiti a backlog (3 stub)**:
- Adaptation pipeline (documentato, non implementato)
- Code generation tools
- Advanced analytics tools

---

### B12 — Memory & VSS Enhancement (G55-70, 3gg)

- `MemoryStore`: `sync.Once` con retry (non fallire permanentemente su primo errore)
- DuckDB VSS: upsert corretto: `DELETE + INSERT` in transazione
- Embedding dimension validation: `len(embedding) == 768` all'avvio
- Goose migration per VSS extension enabling
- **Opportunità 2.6 integrata**: VSS First-Class

---

### B13 — Structured Error Enrichment (G60-68, 2gg)

```
type EnrichedError struct {
    Err        error
    Subsystem  string   // "query" | "chat" | "ingestion" | "discovery" | "nlp"
    Operation  string   // "execute" | "validate" | "connect"
    Recoverable bool    // può ritentare?
    RetryAfter time.Duration
    UserMessage string  // human-readable (IT)
}

// DiagnosticMonitor: correlazione automatica errori per subsystem
// Frontend: messaggi utente basati su categoria errore
// IT: "Il servizio NLP non risponde. Riprovo tra 5 secondi."
// EN: "NLP service is unresponsive. Retrying in 5 seconds."
```

**Opportunità 2.7 integrata**: Structured Error Enrichment.

---

### C09 — Terminal & Chat UI (G48-60, 3gg)

- InlineRenderer: fix JSX error preesistenti (tag non chiusi)
- SSE reconnect: UI indicator (stato connessione: connected/connecting/disconnected)
- Streaming token: rendering fluido (batch update ogni 50ms, non ogni token)
- Command palette: completamento slash commands con tab

---

### C10 — Tool UI Completion (G52-65, 2gg)

- Tool execution result display: formattato per tipo (JSON tree, table, mini-chart)
- Tool configuration forms: campi validati con feedback inline
- Tool card component: finance/OSINT/HE con icona + stato
- Stato tool in UI: MCP discovery status, health indicator (green/yellow/red)

---

### C11 — Dashboard & Analytics (G55-70, 4gg)

- Usage statistics: chiamate API, tool usati, LLM costi (da B04.5)
- Health dashboard: stato backend, NLP, DuckDB, MCP (da A08)
- Query history: lista con performance metrics (tempo, righe)
- LLM cost tracking: per provider, per giorno, trend
- **Opportunità 7.3 integrata**: Stato sistema in tempo reale

---

### C12 — Integration Test Suite (G60-78, 5gg)

```
API integration scenarios (minimo 50 test):
  1. Auth flow: login → session validation → guarded endpoint → logout
  2. Project CRUD: create → read → update → delete (+ cascade verification)
  3. Query workflow: simple query → filtered query → paginated → error case
  4. Ingestion pipeline: create task → run → verify results in DuckDB
  5. Tool execution: echo tool → finance tool → multi-step PAORA
  6. DuckDB+Postgres dual write: verify both stores after ingestion
  7. SSE event flow: subscribe → send message → verify events received
  8. NLP sidecar: entity extraction → sentiment → verify results
```

---

### C13 — Frontend Polish (G65-80, 3gg)

- **Opportunità 3.6**: React Query/SWR per caching e dedup fetch
- **Opportunità 3.7**: Bundle splitting: vendor chunk 295KB → 150KB; factory chunk dual import fix
- **Opportunità 3.8**: CSS purge-safe audit: classi dinamiche via lookup table
- **Opportunità 7.5**: Performance perception: skeleton loader, optimistic UI, debounce 300ms

---

### Ship Gate 3 (G85)

**Check obbligatori**:
- [ ] `go test -race -count=1 ./...` ✅ (0 FAIL)
- [ ] `npx tsc --noEmit` ✅
- [ ] `npx vite build` ✅ (< 3s)
- [ ] Tool packages (finance/osint/he): funzionanti con test
- [ ] PAORA V2 multi-step: test passano (3+ step)
- [ ] UI: tutte le view renderizzano senza errori
- [ ] Integration test: > 50 test
- [ ] E2E Playwright: > 20 scenari
- [ ] Backend test coverage > 70%
- [ ] Frontend test coverage > 50%
- [ ] **Security re-check**: OWASP ZAP full scan → 0 CRITICAL, ≤ 3 HIGH

---

## FASE 4: Production Ready (G75-120, 34gg-uomo)

**Obiettivo**: v2.0 release. Security audit. Performance target. Documentazione completa.

### A13 — Security Audit Completo (G75-95, 6gg)

- Penetration testing OWASP Top 10 completo
- ZAP automated scan → remediation di tutti i finding
- `gosec` senza esclusioni
- Dependency audit: `govulncheck` + `npm audit` + `trivy` Docker scan
- `SECURITY.md` con responsible disclosure policy
- **Opportunità 5.6 integrata**: Vulnerability scanning continuo in CI
- Bug bounty program documentation (HackerOne/GitHub private reporting)

---

### A14 — Performance Optimization (G80-100, 5gg)

- DuckDB: EXPLAIN ANALYZE su query hotspot → indici mancanti
- Query optimization: eliminare query ridondanti, caching risultati frequenti
- Connection pooling: `sql.DB.SetMaxOpenConns()`, `SetMaxIdleConns()`
- Memory profiling con pprof: `import _ "net/http/pprof"`
- Goroutine leak detection: `runtime.NumGoroutine()` trend monitoring
- Target: 500 req/s p95 < 500ms, p99 chat < 2s, p99 query < 500ms

---

### A15 — Onboarding Zero-Config (G85-98, 3gg)

- First-run wizard: configura provider LLM, crea primo agente
- Demo data: 2-3 agenti preconfigurati, 1 datasource fittizio
- Tooltip tour guidato: 5 step per nuovi utenti
- Progress indicator setup (3/5 steps completed)
- **Opportunità 7.2 integrata**: Onboarding Zero-Config

---

### B14 — DuckDB Backup & Recovery (G78-88, 2gg)

```
func (d *DuckDB) Backup(ctx context.Context, destPath string) error {
    // fsync garantito dopo backup
    _, err := d.db.ExecContext(ctx, fmt.Sprintf("EXPORT DATABASE '%s'", destPath))
    if err != nil {
        return fmt.Errorf("backup: %w", err)
    }
    return fsyncDir(destPath)
}

// Backup schedulato: cron interno o esterno (systemd timer, k8s CronJob)
// Recovery procedure documentata in runbook
// Test restore da backup verificato
```

---

### B15 — Error Recovery & Resilience (G80-92, 3gg)

- Act: propagare errori reali (non `return nil, nil`)
- Observe: soglia troncamento configurabile `ObserveMaxChars` (non hardcoded 1900)
- TrustDelta da Engine.Observe: valori reali (non sempre 0.0)
- `validateToolName`: restringere pattern matching (no wildcard accettate)
- Retry policy documentata per ogni subsystem

---

### B16 — Accessibility Audit (G85-95, 2gg)

- Tab order: tutti i form (login → fields → submit con ordine logico)
- Focus trap: modal/slideover (non tab-escape dietro)
- ARIA labels: icone decorative (`aria-hidden="true"`), interattive (`aria-label`)
- Color contrast: testi 13px JetBrains Mono → ratio ≥ 4.5:1
- Keyboard shortcuts: `?` per help, `Ctrl+K` per command palette, `Esc` per chiudere
- WCAG 2.1 AA compliance documentata

---

### C14 — Documentazione Completa (G80-105, 6gg)

```
docs/
├── api/
│   └── reference.md          # OpenAPI/Swagger da protobuf + ConnectRPC
├── guides/
│   ├── user-guide-it.md      # Workflow principali in italiano
│   ├── user-guide-en.md      # User guide in inglese
│   └── deployment.md         # Docker Compose (primario), K8s (reference)
├── development/
│   ├── architecture.md       # System design, data flow, components
│   ├── setup.md              # Dev environment setup, prerequisites
│   └── contributing.md       # PR process, code style, testing requirements
└── operations/
    ├── runbook.md            # Startup/shutdown/recovery/monitoring per subsystem
    └── troubleshooting.md    # Common issues, diagnostic commands
```

---

### C15 — Pre-Release Verification (G95-115, 4gg)

- TypeScript: abilitare `strict: true` in tsconfig.json, fix errori residui
- Bundle size: < 500KB total, chunk splitting ottimizzato
- Performance audit: Lighthouse > 90 (Performance, Accessibility)
- React DevTools profiler: no unnecessary re-renders in production build
- Cross-browser: Chrome, Firefox, Safari tutti funzionanti
- Responsive: mobile (< 768px), tablet, desktop tutti layout corretti

---

### C16 — v2.0 Release (G105-120, 3gg)

```
# Release checklist
[ ] CHANGELOG.md da commit history (conventional commits grouping)
[ ] Version bump: v2.0.0 (go.mod + package.json)
[ ] Git tag: git tag -a v2.0.0 -m "Aleph-v2 Production Grade"
[ ] GitHub Release con release notes
[ ] Docker image: docker build -t aleph-v2:2.0.0 . && docker push
[ ] Migration guide v1.x → v2.0 (breaking changes, upgrade steps)
[ ] Release announcement: blog post, social
```

---

### Ship Gate 4 (G120) — v2.0

**Check obbligatori**:
- [ ] `go test -race -count=1 ./...` ✅ (0 FAIL)
- [ ] `npx tsc --noEmit` ✅ (0 errori)
- [ ] `npx vite build` ✅
- [ ] `npx vitest run` ✅
- [ ] `npx playwright test` ✅ (> 20 scenari)
- [ ] Security: 0 CRITICAL, ≤ 3 HIGH (ZAP + gosec + trivy + npm audit)
- [ ] Load test: 500 req/s p95 < 500ms
- [ ] Performance: p99 chat < 2s, p99 query < 500ms
- [ ] Test coverage: Go > 70%, Frontend > 50%
- [ ] Documentazione: API ref + user guide (IT/EN) + runbook
- [ ] Docker image < 500MB
- [ ] CI/CD: build < 10 min
- [ ] Multi-tenancy base funzionante
- [ ] GDPR compliance base verificata
- [ ] `v2.0.0` tagged su GitHub

---

## Riepilogo Effort Totale

| Fase | Giorni-uomo | Giorni calendario | Track |
|------|-------------|-------------------|-------|
| **Fase 1**: Safety Net | 49 | 30 | A+B+C |
| **Fase 2**: Stability Engine | 40.5 | 35 | A+B+C |
| **Fase 3**: Feature Completion | 45 | 40 | A+B+C |
| **Fase 4**: Production Ready | 34 | 45 | A+B+C |
| **Totale** | **168.5gg-uomo** | **~120gg calendario** | |

---

## Metriche Finali (OKR v2.0)

| KPI | Target v2.0 | Metodo |
|-----|------------|--------|
| Vulnerabilità CRITICAL | 0 | OWASP ZAP + gosec + audit manuale |
| Vulnerabilità HIGH | ≤ 3 | OWASP ZAP + gosec |
| Test coverage Go | > 70% | `go test -cover ./...` |
| Test coverage Frontend | > 50% | `vitest run --coverage` |
| TS strict errors | 0 | `npx tsc --noEmit` con strict:true |
| Build time CI | < 10 min | GitHub Actions |
| p99 latency chat | < 2s | k6 load test |
| p99 latency query | < 500ms | k6 load test |
| Requests/sec | > 500 | k6 load test |
| Sandbox escapes | 0 | Penetration test |
| Uptime | 99.9% | Prometheus/Alertmanager |
| Docker image size | < 500MB | `docker images` |
| Integration tests | > 50 | `go test ./...` |
| E2E scenarios | > 20 | Playwright |
| PAORA tests | > 50 | `go test ./decision/...` |
| Lighthouse score | > 90 | Chrome DevTools |
| WCAG compliance | 2.1 AA | axe-core audit |

---

*Specs generate il 1 Maggio 2026. Allineate a `piano-final-integrato.md` v1.0 con 3 review (Oracle, Momus, Metis). Sostituisce `piano-operativo-specs.md` v1 (90gg).*
