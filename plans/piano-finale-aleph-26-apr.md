# Piano Finale Aleph-v2 — 26 Aprile 2026

> **Sorgente**: Sintesi di 8 agenti paralleli (Oracle, Momus, Metis, Explore, Backend, Frontend, DB, Architecture) + Review finale 8-gap
> **Baseline**: a-v2.2-ultrabrain-audit-plan.md + reconciliation plan W0-W6
> **Audit scope**: Build, Tests, Correctness, Architecture, Features, Observability, Security, Performance, Deployment, Documentation, Accessibility, Error Handling
> **Build verificati**: `go build ./...` ✅ | `tsc --noEmit` ❌ (43 errori) | `vitest run` 117/117 ✅ ma runtime-only
> **Questa versione**: 71 task originali preservati + 5 nuove wave (27 task) + rinforzi in-task per Generalizzazione e Ingegneria.

---

## Principi Ordinanti

1. **Build fixes prima di tutto** — Se `tsc --noEmit` non passa, CI/CD è rotto
2. **Sicurezza prima delle feature** — SQL injection in scopeQuery è CRITICAL
3. **Infrastruttura prima della logica** — Hardening W1-W2 prima del Decision Loop
4. **Decision trace integrato, non separato** — Oracle: merge telemetry in W4-W5
5. **E2E test e build check in OGNI wave**
6. **Error handling sistematico** — Zero bare errors, wrapping obbligatorio, logging strutturato (Gap 6)
7. **Generalizzazione obbligatoria** — Ogni fix di bug include audit di tutti i casi simili (Gap 8)
8. **Interfacce precise** — Ogni componente critico ha contract esplicito con validation rules (Gap 5)
9. **Deployability prima del release** — CI/CD, container, health check, env config (Gap 1)
10. **Documentazione come codice** — README, ARCHITECTURE, API docs, changelog (Gap 2)

---

## FINDINGS SINTETIZZATI (da 8 agenti + 8-gap review)

### CRITICAL (blocco immediato)
| # | Finding | Fonte | Gravità |
|---|---------|-------|---------|
| C1 | SQL injection in `scopeQuery` — concatenazione stringa | DB Retry | CRITICAL |
| C2 | SQL injection in `info_schema` queries | Backend Retry | CRITICAL |
| C3 | `toolFailCount` variabile globale senza lock | Backend Retry | CRITICAL |
| C4 | `RWMutex` usato come `Mutex` (Lock/Unlock invece di RLock/RUnlock) | DB Retry | CRITICAL |
| C5 | DuckDB zero transazioni — ogni operazione è auto-commit | DB Retry | CRITICAL |
| C6 | `h.db.Cleanup()` chiamato in loop + in AdmitFailure — double cleanup | Momus | HIGH |
| C7 | `Chat()` god method ~440 righe — query.go 1147 righe totali | Architecture Retry | HIGH |
| C8 | `Config.Validate()` mancante — nessuna validazione config al boot | Architecture Retry | HIGH |
| C9 | Memory embedding hardcode `NewEmbedder("http://localhost:11434", "")` | Oracle | HIGH |
| C10 | Tool registry race — `RegisterAll()` senza lock | Oracle | HIGH |
| C11 | LLM zero retry/cache — ogni fallimento è definitivo | Backend Retry | HIGH |
| C12 | 43 errori TS (non 10) in `StateCreator<T>` | Metis | HIGH |

### Architecture
| # | Finding | Fonte |
|---|---------|-------|
| A1 | query.go (1147 righe) va estratto in `internal/decision/` PRIMA di W4-W5 | Oracle |
| A2 | W5 LLM-in-Observe aggiunge latenza inaccettabile → hybrid scorer raccomandato | Oracle |
| A3 | W8 A2A in conflitto con Chat() sincrono — Oracle: risolvere con Job queue | Oracle |
| A4 | W6 telemetry da fondere in W4-W5, non wave separata | Oracle |
| A5 | `plannedTool` deve validare contro ToolRegistry prima di eseguire | Oracle |
| A6 | NotificationService.Stop() non collegato a app.Close() | Momus/Explore |
| A7 | tool_suggest cleanup usa `r.Context()` invece di `app.ctx` — contesto morto su shutdown | Explore |
| A8 | Ingestion Engine senza `Stop()` metodo | Explore |
| A9 | DuckDB backup Lock() esclusivo blocca letture | Momus |
| A10 | MCP discovery + Health checker start senza graceful stop | Explore |
| A11 | SSE broker close prima di server shutdown → client EOF | Explore |
| A12 | Vitest.config.ts ESISTE già (non serve W1-02) | Momus |
| A13 | SlideOver null-safe è GIA' fatto (VIEW_REGISTRY) | Momus |

### Frontend
| # | Finding | Fonte |
|---|---------|-------|
| F1 | 43 errori TypeScript in store test — `StateCreator<T>` richiede 3 arg, test passano 1 | Metis |
| F2 | `any` type diffuso (40+ occorrenze) | Frontend Retry |
| F3 | Polling senza cleanup in hook → memory leak su unmount | Frontend Retry |
| F4 | 88% componenti senza test | Frontend Retry |
| F5 | ErrorBoundary mancante in punti critici | Frontend Retry |
| F6 | useInfiniteQueries usa offset, non cursor pagination | Explore |
| F7 | component tests: solo 4 form test esistenti | Explore |
| F8 | 15 hook file, zero test | Explore |

### Backend
| # | Finding | Fonte |
|---|---------|-------|
| B1 | NLP dead code (`ensemble.py`, `calibration.py`, `predict.py`, `*.onnx`) | Explore |
| B2 | gRPC proto import fix (`nlp_pb2_grpc.py`) | Explore |
| B3 | Honest responses — `analyze_sentiment` può restituire score fittizio | Explore |
| B4 | ToolCodeWriter/Reader interfacce senza `context.Context` | Explore |
| B5 | DuckDB AutoBackup Lock() → RLock() | Momus/DB Retry |
| B6 | DuckDB manca supporto transazionale (Begin/Commit/Rollback) | DB Retry |
| B7 | Informant engine senza schema di configurazione centralizzato | DB Retry |
| B8 | 3 test Go falliscono (Metis discovered) | Metis |

### E2E / Testing
| # | Finding | Fonte |
|---|---------|-------|
| E1 | Vitest run 117/117 OK, ma nessun test di integrazione frontend-backend | Metis |
| E2 | Playwright test NON verificato (piano richiede 21/21) | Metis |
| E3 | Go test coverage backend sconosciuto | Metis |
| E4 | Nessun benchmark di latenza/throughput | Momus |
| E5 | W5 e W6 reconciliation segnano completi MA piani non documentati individualmente | Explore |

### Gap Review — 8 Domini Mancanti
| # | Finding | Fonte | Gravità |
|---|---------|-------|---------|
| G1 | **Zero deployment tasks** — no Docker prod, no CI/CD hardening, no health endpoint, no K8s manifests | Gap Review | HIGH |
| G2 | **Documentazione carente** — README superficiale, no API docs, no changelog, developer onboarding assente | Gap Review | HIGH |
| G3 | **Dependency non aggiornate** — go mod outdated, npm audit non eseguito, no version pinning, no Dependabot | Gap Review | MEDIUM |
| G4 | **W10 misura MA non ottimizza** — benchmark senza ciclo fix→re-benchmark | Gap Review | MEDIUM |
| G5 | **Interfacce vaghe** — DecisionEngine, plannedTool, Hybrid scorer, Job queue senza spec precise | Gap Review | HIGH |
| G6 | **Error handling non sistematico** — errori bare, no classificazione, no logging strutturato, no graceful degradation | Gap Review | HIGH |
| G7 | **Accessibilità assente** — nessun audit WCAG, keyboard nav, focus mgmt, screen reader, responsive test | Gap Review | MEDIUM |
| G8 | **Fix puntuali, non sistemici** — ogni bug fixato in un punto MA pattern non applicato ovunque | Gap Review | MEDIUM |

---

## PIANO D'AZIONE DETTAGLIATO

### WAVE 0: CRITICAL FIXES (Blocco Immediato)

> **Obiettivo**: Risolvere CRITICAL items che bloccano tutto il resto.
> **Durata**: ~1.5gg

| ID | Task | File | Effort | PP? |
|----|------|------|--------|-----|
| W0-01 | **Fix SQL injection in scopeQuery** — Sostituire concatenazione con query parametrizzate. **E AUDIT di TUTTE le query DuckDB per concatenazione stringa — fixarle tutte** (Gap 8) | `internal/storage/duckdb.go` (scopeQuery) + audit tutte le query | S | Sì |
| W0-02 | **Fix SQL injection in info_schema queries** — Parametrizzare tutte le query info_schema | `internal/storage/duckdb.go`, `internal/repository/*.go` | S | Sì |
| W0-03 | **Fix toolFailCount globale** — Aggiungere `sync.Mutex` o passare a struct thread-safe | `internal/api/handler/query.go` | S | Sì |
| W0-04 | **Fix RWMutex → Lock/Unlock sostituito con RLock/RUnlock. E AUDIT di TUTTI i sync.Mutex/RWMutex nel codebase per correttezza Lock/RLock** (Gap 8) | `internal/storage/duckdb.go` + audit globale | S | Sì |
| W0-05 | **Fix h.db.Cleanup() double call** — Rimuovere cleanup ridondante da AdmitFailure | `internal/api/handler/query.go` | S | Sì |
| W0-06 | **Fix tool registry race** — Aggiungere sync.Mutex a RegisterAll() | `internal/app/app.go` | S | Sì |
| W0-07 | **Fix Memory embedding hardcode** — Passare config da app.go invece di hardcode | `internal/api/handler/query.go`, `internal/app/app.go` | S | Sì |
| W0-08 | **Aggiungere Config.Validate()** — Validare tutta la configurazione al boot prima di avviare server | `internal/config/config.go` (nuovo metodo) | S | Sì |
| W0-09 | **Aggiungere LLM retry (3 tentativi) + cache TTL 5min** | `internal/llm/` | M | Sì |

**Build check W0**:
```bash
go build ./... && go test ./... 2>&1 | head -50
```

---

### WAVE 1: Build Fixes + Hardening Strutturale

> **Obiettivo**: `go build`, `go test`, `tsc --noEmit`, `vitest run` TUTTI puliti.
> **Durata**: ~1gg
> **Dipende da**: W0

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W1-01 | **Fix 43 errori TypeScript store test** — Creare tipo `SliceCreator<T> = (set: GetState, get: GetState, api: StoreApi<T>) => T` o fixare chiamate con 3 arg | `frontend/src/store/*Slice.ts`, `__tests__/*.test.ts` | M | Sì | W0 |
| W1-02 | **Verificare vitest.config.ts esiste e funziona** (Momus: già esiste) — Aggiungere coverage reporter, jsdom, timeout 10s se mancanti | `frontend/vitest.config.ts` | S | Sì | W0 |
| W1-03 | **Fix polling cleanup in hook** — Aggiungere cleanup su unmount in useEffect + AbortController. **E AUDIT di TUTTI gli hook per useEffect cleanup mancante** (Gap 8) | `frontend/src/hooks/*.ts` + audit globale | M | Sì | W0 |
| W1-04 | **NotificationService.Stop() wiring** — Chiamare in app.Close() PRIMA di server shutdown | `internal/app/app.go` | S | Sì | W0 |
| W1-05 | **tool_suggest cleanup ctx fix** — Sostituire `r.Context()` con `app.ctx` persistente | `internal/api/handler/tool_suggest.go` | S | Sì | W0 |
| W1-06 | **Ingestion Engine.Stop()** — Aggiungere metodo Stop() + chiamata da app.Close() | `internal/ingestion/ingestion.go`, `internal/app/app.go` | S | Sì | W0 |
| W1-07 | **DuckDB backup Lock→RLock** — Usare RLock + snapshot read per backup | `internal/storage/duckdb_backup.go` | S | Sì | W0 |
| W1-08 | **SSE broker graceful shutdown** — Chiudere SSE broker PRIMA di server.Shutdown() | `internal/app/app.go` | S | Sì | W0 |
| W1-09 | **Verifica build frontend** — `npx tsc --noEmit` deve passare, `npx vitest run` con coverage | — | S | No | W1-01,02,03 |

**Build check W1**:
```bash
npx tsc --noEmit && npx vitest run --coverage && go build ./... && go vet ./...
```

---

### WAVE 2: God Method Extraction + DuckDB Transactions ✅

> **Obiettivo**: query.go estratto in internal/decision/, Chat() decomposto, DuckDB transazionale.
> **Durata**: ~1.5gg
> **Dipende da**: W1

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W2-01 | **Estrarre internal/decision/** — Spostare logica decisionale da query.go (1147 righe). Creare: `planner.go`, `observer.go`, `reflector.go`, `admitter.go`, `decision.go`. **SPECIFICA DecisionEngine interface** (Gap 5): `Plan(ctx, intent)`, `Act(ctx, step)`, `Observe(ctx, result, expected)`, `Reflect(ctx, plan, obs)`, `Admit(ctx, results, maxAttempts)`. Intent, PlanResult, PlannedStep, ActResult, ExpectedOutput, Observation, AdmitResult sono struct con TUTTI i campi documentati (tipo, required/optional, validation). | `internal/decision/` (nuovo package) | M | No | W1 |
| W2-02 | **Chat() decomposto** — Refactor metodo ~440 righe in fasi chiamabili: Plan(), Act(), Observe(), Reflect(), Admit() su DecisionEngine | `internal/api/handler/query.go` → `internal/decision/engine.go` | L | No | W2-01 |
| W2-03 | **Aggiungere DuckDB transazioni** — Implementare Begin/Commit/Rollback su DuckDBStore | `internal/storage/duckdb.go` | M | Sì | W1 |
| W2-04 | **Transaction-aware query scopeQuery** — W0-01 già parametrizzato; ora usare transazioni BeginTX/Commit | `internal/storage/duckdb.go` (scopeQuery) | S | Sì | W2-03 |
| W2-05 | **ToolRegistry validazione in plannedTool** — plannedTool deve validare toolName contro registry prima di Act | `internal/decision/planner.go` | S | Sì | W2-01 |
| W2-06 | **Informant engine config centralizzata** — Estrarre config in internal/config/ | `internal/storage/duckdb.go` (informant) | M | Sì | W1 |

**Build check W2**:
```bash
go build ./... && go test ./... 2>&1 | grep -E "(FAIL|PASS|---)" | head -30
```

---

### WAVE 3: Hardening W2 — Test + NLP Cleanup

> **Obiettivo**: Onestà profonda, test hooks e componenti, NLP dead code rimosso.
> **Durata**: ~2gg
> **Dipende da**: W2

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W3-01 | **DELETE NLP dead code** — Eliminare ensemble.py, calibration.py, predict.py, convert_onnx.py, *.onnx | `nlp/` | S | Sì | W2 |
| W3-02 | **Fix gRPC proto import** — Check-in .proto, rigenerare pb2, fixare import path | `nlp/*.proto`, `nlp_pb2_grpc.py` | S | Sì | W2 |
| W3-03 | **Honest responses** — Se NLP non restituisce confidence, rispondere "NLP confidence unavailable" invece di 0.5 fittizio | `internal/api/handler/query.go` (analyze_sentiment) | S | Sì | W2 |
| W3-04 | **ToolCodeWriter/Reader + context.Context** — Aggiungere ctx a tutte le interfacce codeflow/. **E AUDIT di TUTTE le interfacce per ctx.Context mancante** (Gap 8) | `internal/tools/codeflow/*.go` + audit interfacce | S | Sì | W2 |
| W3-05 | **Hook unit tests (15 file)** — Testare useAppActions, useChat, useSSE, useToolActions, useViewActions, useSlideOver | `frontend/src/hooks/__tests__/*.test.ts` | M | Sì | W1 |
| W3-06 | **Store slice tests completi** — Coverage >= 90% per ogni slice | `frontend/src/store/__tests__/` | S | Sì | W3-05 |
| W3-07 | **Fix 3 Go test fallimenti** — Diagnosticare e fixare i test falliti scoperti da Metis | `internal/*/**_test.go` | M | No | W2 |

**Build check W3**:
```bash
npx vitest run --coverage && go test ./... 2>&1 | grep -E "^(ok|FAIL)" && npx tsc --noEmit
```

---

### WAVE 4: Decision Loop — Planner (F3-01)

> **Obiettivo**: Decision Loop con vero Planner: intent parsing, rationale, expected, fallback.
> **Durata**: ~2gg
> **Dipende da**: W3
> **Nota Oracle**: Telemetry integrata, non separata. I decision spans nascono qui.

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W4-01 | **Redefinire plannedTool struct** — Aggiungere: `Rationale`, `ExpectedOutput`, `FallbackTool`, `FallbackParams`, `ToolValidation`. **REGOLA DI VALIDAZIONE (Gap 5)**: Rationale è REQUIRED (non-empty string). ExpectedOutput è REQUIRED (usato da Observe). FallbackTool è OPTIONAL ma se assente, tool execution failure → Admit immediato. ToolValidation: toolName deve esistere in ToolRegistry prima di Act. Se Rationale vuoto → planner re-invocato con richiesta esplicita. | `internal/decision/planner.go` | S | Sì | W3 |
| W4-02 | **Intent parsing ricorsivo in Plan** — LLM call per decomporre intent. Prompt richiede JSON `steps[].{tool,params,expected,rationale,fallback}` | `internal/decision/planner.go` | M | Sì | W4-01 |
| W4-03 | **AdmitFailure precoce** — Se planner restituisce `[]` (nessun tool), rispondere subito con motivo | `internal/decision/planner.go` | S | Sì | W4-02 |
| W4-04 | **StartDecisionSpan + AddToolCandidate** — Creare span builder in telemetry per tracciare candidate tool, score, selected | `internal/telemetry/traces.go` (nuovo) | M | Sì | W4-02 |
| W4-05 | **Instrumentare Plan() con decision spans** — Tool candidates + rationale tracciati in OTEL span | `internal/decision/planner.go` | S | Sì | W4-04 |
| W4-06 | **Test Planner** — Mock LLM, test intent parsing, fallback selection, AdmitFailure precoce | `internal/decision/planner_test.go` | M | No | W4-03,05 |

**Build check W4**:
```bash
go build ./... && go test ./internal/decision/ -run TestPlanner -v && npx tsc --noEmit
```

---

### WAVE 5: Decision Loop — Observe + Reflect + Re-plan (F3-02)

> **Obiettivo**: Observe valuta output con hybrid scorer (LLM + euristico). Reflect ricalibra. Re-plan strutturato.
> **Durata**: ~2gg
> **Dipende da**: W4
> **Nota Oracle**: Hybrid scorer raccomandato (non solo LLM) per latenza.

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W5-01 | **Hybrid Observe scorer** — LLM eval + euristiche (error pattern match, empty result detect). Confronta output con ExpectedOutput. Restituisce `{diverges, reason, confidence}`. **ALGORITMO PRECISO (Gap 5)**: Euristiche weight: 0.3 (empty output = fail, error return = fail, exact match = pass). LLM weight: 0.7 (semantic eval output vs ExpectedOutput). Divergence threshold: >0.6 → trigger re-plan. Confidence scoring: combina entrambi in `{diverges: bool, reason: string, confidence: float64}`. | `internal/decision/observer.go` | M | Sì | W4 |
| W5-02 | **Reflect + Re-plan** — Se divergenza, LLM call per ricalibrare piano con motivazione. Max 2 re-plan | `internal/decision/reflector.go` | M | Sì | W5-01 |
| W5-03 | **AdmitFailure strutturato** — Dopo max 2 re-plan falliti, rispondere con motivo specifico + trace. Niente successo inventato | `internal/decision/admitter.go` | S | Sì | W5-02 |
| W5-04 | **AddDivergence + AddReplanAttempt spans** — Tracciare divergenza, re-plan, decisioni in telemetry | `internal/telemetry/traces.go` | S | Sì | W5-01 |
| W5-05 | **Instrumentare Observe/Reflect/Admit con spans** — Tutte le fasi tracciate in OTEL | `internal/decision/observer.go`, `reflector.go`, `admitter.go` | M | Sì | W5-04 |
| W5-06 | **Test Observe/Reflect/Re-plan** — Mock LLM simula output inaspettato → ricalibra → secondo tentativo → admit | `internal/decision/observer_test.go` | M | No | W5-03,05 |

**Build check W5**:
```bash
go build ./... && go test ./internal/decision/ -run TestObserveReflect -v
```

---

### WAVE 6: Tool Interfaces + Residuali Hardening

> **Obiettivo**: Completare item residuali, ctx propagation, MCP discovery graceful stop.
> **Durata**: ~1.5gg
> **Dipende da**: W5

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W6-01 | **MCP discovery stop wiring** — Aggiungere wait/channel per graceful stop del discovery loop | `internal/mcp/discovery.go`, `internal/app/app.go` | S | Sì | W5 |
| W6-02 | **Health checker stop wiring** — Attendere stop completto in app.Close() | `internal/health/checker.go`, `internal/app/app.go` | S | Sì | W5 |
| W6-03 | **Component tests — 10 componenti critici** — Testare TerminalOutput, CopilotView, SlideOverPanel, ToolSuggest, SkillForm, ToolForm | `frontend/src/components/__tests__/*.test.tsx` | M | Sì | W3 |
| W6-04 | **Hook integration tests** — useChat + useSSE + useToolActions integration | `frontend/src/hooks/__tests__/*.integration.test.ts` | M | Sì | W6-03 |
| W6-05 | **Fix any types in frontend** — Sostituire 40+ `any` con tipi specifici o unknown | `frontend/src/**/*.ts` (40+ occorrenze) | M | No | W3 |

**Build check W6**:
```bash
go build ./... && npx tsc --noEmit && npx vitest run --coverage | tail -20
```

---

### WAVE 7: Frontend Coverage + Playwright E2E

> **Obiettivo**: Frontend coverage >= 50%. Playwright 21/21 test passano.
> **Durata**: ~1.5gg
> **Dipende da**: W6

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W7-01 | **Aggiungere ErrorBoundary mancanti** — Coprire punti critici senza ErrorBoundary | `frontend/src/App.tsx`, views | S | Sì | W6 |
| W7-02 | **Fix cursor pagination** — useInfiniteQueries da offset a cursor | `frontend/src/hooks/useDataQuery.ts` | S | Sì | W6 |
| W7-03 | **Playwright E2E test suite** — 21 test: login, chat, tool exec, slideover, copilot, settings, error states | `frontend/e2e/` | M | Sì | W6 |
| W7-04 | **Frontend coverage gate** — Portare coverage a >= 50% (attuale ~12% basato su 4 form test) | `frontend/src/` | M | No | W7-01,02,03 |

**Build check W7**:
```bash
npx vitest run --coverage && npx playwright test && npx vite build
```

---

### WAVE 8: Multi-Agent A2A Protocol (F4)

> **Obiettivo**: Comunicazione agente-agente via A2A. Solo DOPO single-agent LAM funzionante (W4-W5).
> **Durata**: ~4gg
> **Dipende da**: W7

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W8-01 | **A2A RFC + spec** — identity, message envelope, task lifecycle, capability discovery | `docs/a2a-protocol.md` | M | Sì | W7 |
| W8-02 | **A2A transport** — HTTP/JSON envelope, routing `/a2a/`, timeout 10s per hop | `internal/a2a/transport.go` | M | Sì | W8-01 |
| W8-03 | **Agent capability registry** — Registrazione capabilities al boot, routing | `internal/a2a/registry.go` | S | Sì | W8-02 |
| W8-04 | **Job queue per A2A** — Oracle: Chat() sincrono confligge con A2A. Creare Job Queue per dispatch asincrono. **PROTOCOLLO PRECISO (Gap 5)**: REST polling con intervallo configurabile (default 1s) + max wait (30s). O WebSocket per real-time (preferito A2A). Timeout per job configurabile (default 60s). Max retries per job (3). Dead letter queue dopo max retries. Job status: pending, running, completed, failed, timed_out. | `internal/a2a/queue.go`, `internal/decision/engine.go` | L | No | W8-03 |
| W8-05 | **AgentOrchestrator** — Refactor DecisionEngine per routing multi-agente: intent → route a sub-agenti → merge output → conflitto detection | `internal/orchestrator/orchestrator.go` | L | No | W8-04 |
| W8-06 | **Multi-agent test suite** — 2 agent collaborativi, 2 con conflitto, 3 agent merge | `internal/orchestrator/orchestrator_test.go` | M | No | W8-05 |

**Build check W8**:
```bash
go build ./... && go test ./internal/a2a/... ./internal/orchestrator/... -v | grep -E "(PASS|FAIL|---)"
```

---

### WAVE 9: Sicurezza (F5)

> **Obiettivo**: Policy engine + Judge advisory + defense in depth.
> **Durata**: ~3gg
> **Dipende da**: W8

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W9-01 | **Policy engine** — YAML rules → compiled policy. Per-tool constraints (allowed params, rate limit, max calls) | `internal/policy/engine.go`, `internal/policy/rules.go` | M | Sì | W8 |
| W9-02 | **Judge Model advisory** — LLM call separata che reviewa tool selection (warn, NON block). Traccia in OTEL span | `internal/judge/judge.go` | M | Sì | W9-01 |
| W9-03 | **Defense in depth chain** — Policy check → Judge review → Execution → Output sanitization | `internal/decision/engine.go` | M | Sì | W9-02 |
| W9-04 | **Sandbox Verifier nel decision loop** — Valida tool code PRIMA del dispatch. Se malevolo → policy block | `internal/sandbox/verification.go`, `internal/decision/` | S | Sì | W9-01 |
| W9-05 | **Policy compliance test suite** — Input avversari contro policy rules | `internal/policy/policy_test.go` | M | Sì | W9-04 |
| W9-06 | **Security audit finale** — OWASP Top 10 review di tutto il codice modificato | `docs/security/audit.md` | M | No | W9-05 |

**Build check W9**:
```bash
go build ./... && go test ./internal/policy/... ./internal/judge/... -v && go vet ./...
```

---

### WAVE 10: Valutazione + E2E Benchmark (F7)

> **Obiettivo**: Prova quantitativa che funziona. Quality gates per ship.
> **Durata**: ~3gg
> **Dipende da**: W9

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| W10-01 | **E2E benchmark framework** — Task suite con scoring automatico: RAG query, tool exec, multi-step plan, multi-agent, adversarial | `benchmark/suite/` | L | Sì | W9 |
| W10-02 | **Decision trace audit tool (backend)** — Replay decisioni con filtri per agente/tool/outcome | `internal/telemetry/replay.go` | S | Sì | W9 |
| W10-03 | **Decision trace dashboard (frontend)** — Albero decisionale visuale, export JSON | `frontend/src/views/DecisionTraceView.tsx` | M | Sì | W9 |
| W10-04 | **Load test multi-agente** — 10+ sessioni concorrenti. Metrics: goroutine leak, latency P50/P95, throughput, mem/session | `benchmark/load/` | M | Sì | W10-01 |
| W10-05 | **Frontend coverage target >= 60%** | `frontend/src/` | M | Sì | W10-03 |
| W10-06 | **Rollback + migration plan v2→v3** — Schema DB changes, API backward compat, rollback procedure, feature flags | `docs/plans/v2-to-v3-migration.md` | S | Sì | W10-04 |
| W10-07 | **Full system verification** — Tutti i build check + test + benchmark | — | S | No | W10-05,06 |

**Build check W10 (Ship Gate)**:
```bash
# Backend
go build ./... && go test -race ./... && go vet ./...
# Frontend
npx tsc --noEmit && npx vitest run --coverage && npx vite build
# E2E
npx playwright test
# Benchmark
go test ./benchmark/... -bench=. -benchmem
```

---

### WAVE W-ERR: Error Handling Standards + Dependency Audit (Gap 3 + Gap 6)

> **Obiettivo**: Error handling sistematico in tutto il codebase. Dipendenze aggiornate e pinnate.
> **Durata**: ~2gg
> **Dipende da**: W0, W1 (build deve passare, pattern devono essere applicati su tutto il codebase)

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| ERR-01 | **Error wrapping standard** — Audit di TUTTI gli errori: ogni `return err` NON wrappato → `fmt.Errorf("context: %w", err)`. Zero bare errors. | `internal/**/*.go` | M | Sì | W0 |
| ERR-02 | **Sentinel errors + classification** — Definire error types: `ErrTemporary`, `ErrPermanent`, `ErrClient`, `ErrServer`. Applicare in error returns. | `internal/errors/errors.go` (nuovo) + `internal/**/*.go` | M | Sì | ERR-01 |
| ERR-03 | **Structured logging** — Adottare `slog` (o zerolog). Ogni log deve avere campi: error, component, operation, duration, request_id. Sostituire tutti i `log.Printf` sparsi. | `internal/logging/` (nuovo) + `internal/**/*.go` | M | Sì | ERR-02 |
| ERR-04 | **Graceful degradation** — Se DuckDB down: log + return 503. Se LLM fail: log + return "LLM unavailable" (no crash). Se tool fail: log + Admit con motivo. | `internal/api/handler/*.go`, `internal/decision/engine.go` | M | Sì | ERR-03 |
| ERR-05 | **Standard error response JSON** — Tutti gli errori HTTP rispondono con `{error: {code, message, details, request_id}}`. Creare helper `RespondError(w, err, requestID)`. | `internal/api/middleware/` (nuovo), tutti gli handler | M | Sì | ERR-04 |
| ERR-06 | **Dependency audit Go** — `go list -u -m all`, aggiornare deps outdated, pinnare versioni in go.mod. | `go.mod` | S | No | W1 |
| ERR-07 | **Dependency audit frontend** — `npm audit` + `npx npm-check-updates`, fixare vulnerabilità, pinnare versioni in package.json. Aggiungere Dependabot/renovate config. | `frontend/package.json`, `.github/dependabot.yml` (nuovo) | S | No | W1 |

**Build check W-ERR**:
```bash
go build ./... && go vet ./... && go test ./... && npx tsc --noEmit && npm audit --audit-level=high
```

---

### WAVE W-A11Y: Accessibility + UX Audit (Gap 7)

> **Obiettivo**: WCAG 2.1 AA compliance. Keyboard navigation, screen reader, responsive.
> **Durata**: ~2gg
> **Dipende da**: W7 (frontend deve essere stabile con coverage >= 50%)

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| A11Y-01 | **WCAG 2.1 AA audit** — Verificare CopilotView, Terminal, SlideOver, Settings per AA compliance. Generare report con violazioni. | `frontend/src/views/*.tsx`, `frontend/src/components/*.tsx` | M | Sì | W7 |
| A11Y-02 | **Keyboard navigation** — Tab order logico, Escape chiude modali, Enter submit forms. Focus trap in modali, focus restoration su close. | `frontend/src/components/*.tsx`, `frontend/src/App.tsx` | M | Sì | A11Y-01 |
| A11Y-03 | **Color contrast** — Verificare tutti i testi contro WCAG AA (4.5:1 normal, 3:1 large). Fixare contrast violations in tema e componenti. | `frontend/src/**/*.css`, `frontend/src/theme/` | S | Sì | A11Y-01 |
| A11Y-04 | **Screen reader support** — Aggiungere aria-labels su icon buttons, aria-live regions per contenuti dinamici (chat stream, notifiche), ruolo corretto su elementi interattivi. | `frontend/src/components/*.tsx`, `frontend/src/views/*.tsx` | M | Sì | A11Y-02 |
| A11Y-05 | **Responsive design test** — Testare e fixare layout a 320px, 768px, 1024px, 1440px. Breakpoint CSS, sidebar collapse, terminal resize. | `frontend/src/**/*.css`, `frontend/src/components/layout/` | M | Sì | A11Y-03 |

**Build check W-A11Y**:
```bash
npx tsc --noEmit && npx vitest run && npx playwright test --project=chromium
```

---

### WAVE W-PERF: Performance Optimization Cycle (Gap 4)

> **Obiettivo**: Dopo W10 benchmark, identificare e fixare top 3 bottleneck, ri-benchmark.
> **Durata**: ~2.5gg
> **Dipende da**: W10 (ha prodotto benchmark baseline)

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| PERF-01 | **Analyze W10 benchmarks** — Identificare top 3 bottleneck da load test (latenza, memoria, goroutine). Produrre report con root cause hypothesis. | `docs/performance/bottleneck-analysis.md` | S | No | W10 |
| PERF-02 | **Fix bottleneck #1: DuckDB query optimization** — Analizzare EXPLAIN per query lente, aggiungere indici, ottimizzare join. Target: -30% latenza query. | `internal/storage/duckdb.go`, `internal/repository/*.go` | M | Sì | PERF-01 |
| PERF-03 | **Fix bottleneck #2: Bundle code splitting** — Analizzare vite bundle, implementare React.lazy + Suspense per views, code splitting per route. Target: -40% initial load. | `frontend/src/App.tsx`, `frontend/vite.config.ts` | M | Sì | PERF-01 |
| PERF-04 | **Fix bottleneck #3: LLM call caching** — Implementare cache layer per chiamate LLM identiche (TTL configurabile, cache key = prompt hash + model). Target: -50% LLM calls in benchmark suite. | `internal/llm/cache.go` (nuovo), `internal/decision/` | M | Sì | PERF-01 |
| PERF-05 | **Re-benchmark post-optimization** — Eseguire W10-01 e W10-04 con le ottimizzazioni. Confrontare P50/P95/mem pre e post. Produrre comparison report. | `benchmark/suite/`, `benchmark/load/` | M | No | PERF-02,03,04 |
| PERF-06 | **Performance regression gate** — Aggiungere benchmark al CI: se P50 > 2x baseline → fail. Documentare performance budget in docs/performance/. | `.github/workflows/ci.yml`, `docs/performance/budget.md` | S | Sì | PERF-05 |

**Build check W-PERF**:
```bash
go test ./benchmark/... -bench=. -benchmem -count=3 && npx vite build --mode production
```

---

### WAVE W-DEPLOY: Deployment + DevOps (Gap 1)

> **Obiettivo**: Container production-ready, CI/CD hardening, health endpoints, deploy guide.
> **Durata**: ~2gg
> **Dipende da**: W10 (sistema deve passare ship gate), può parallelizzare con W-PERF, W-DOCS

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| DEP-01 | **Dockerfile multi-stage audit + optimization** — Verificare Dockerfile esistente: multi-stage build, layer caching, minimal base image (distroless/alpine), no secrets in image. Fix se carente. | `Dockerfile`, `frontend/Dockerfile`, `nlp/Dockerfile` | M | Sì | W10 |
| DEP-02 | **docker-compose.yml hardening** — Assicurare tutti i servizi (DuckDB, Ollama, backend, frontend) con healthcheck, restart policy, volume mounts, network isolation. | `docker-compose.yml` | M | Sì | DEP-01 |
| DEP-03 | **CI/CD pipeline hardening** — GitHub Actions: build cache (Go + npm), test paralleli (go test + vitest), coverage gate (>= 60%), deploy stage (build + push image). | `.github/workflows/ci.yml` | M | Sì | DEP-02 |
| DEP-04 | **Environment config audit** — Verificare `.env.example` ha TUTTE le variabili richieste con default per dev. Documentare ogni var: scopo, tipo, default, required. | `.env.example`, `docs/deployment/env-vars.md` | S | Sì | W10 |
| DEP-05 | **Health endpoint `/readyz`** — Aggiungere endpoint che verifica: DuckDB connectivity (ping), LLM connectivity (probe), memoria disponibile, uptime. Health JSON standard. | `internal/api/handler/health.go` (nuovo), `internal/app/app.go` | S | Sì | W10 |
| DEP-06 | **Deployment guide** — Istruzioni per: local dev (docker compose up), production (docker + env vars), K8s manifests (o link a Helm chart). | `docs/deployment/README.md` | S | Sì | DEP-02,05 |

**Build check W-DEPLOY**:
```bash
docker compose build && docker compose up -d && sleep 5 && curl -sf http://localhost:8080/readyz && docker compose down
```

---

### WAVE W-DOCS: Documentation + Quality (Gap 2)

> **Obiettivo**: Documentazione completa: setup, architettura, API, changelog, onboarding.
> **Durata**: ~2gg
> **Dipende da**: W10 (API e architettura sono stabili), può parallelizzare con W-PERF, W-DEPLOY

| ID | Task | File | Effort | PP? | Dipende da |
|----|------|------|--------|-----|------------|
| DOC-01 | **README.md overhaul** — Setup rapido, architettura overview, configurazione, come eseguire, come contribuire. Link a docs/ esistenti. | `README.md` | S | No | W10 |
| DOC-02 | **ARCHITECTURE.md update** — Documentare nuova struttura moduli dopo refactor (internal/decision/, internal/a2a/, internal/orchestrator/). Diagramma componenti Mermaid. | `ARCHITECTURE.md` | M | No | W10 |
| DOC-03 | **API documentation** — OpenAPI/Swagger per HTTP routes. gRPC proto docs. Endpoint list, request/response examples, error codes. | `docs/API.md` (update), `docs/api/swagger.yaml` (nuovo) | M | No | W10 |
| DOC-04 | **Changelog v2→v3** — Breaking changes, nuove feature, bug fixes, migration notes. Collegamento a W10-06 rollback plan. | `CHANGELOG.md` (nuovo) | S | No | DOC-03 |
| DOC-05 | **Developer onboarding guide** — Prerequisiti (Go 1.22+, Node 20+, DuckDB, Ollama), primo run, test suite, struttura directory, convenzioni codice. | `docs/contributing/onboarding.md` | M | No | DOC-01 |
| DOC-06 | **Error handling standards doc** — Documentare pattern da W-ERR: wrapping, classification, logging, error response format, graceful degradation. | `docs/architecture/error-handling.md` | S | No | W-ERR |

**Build check W-DOCS**:
```bash
# Verify all docs exist and are valid markdown
ls -la README.md ARCHITECTURE.md CHANGELOG.md docs/API.md docs/deployment/README.md docs/contributing/onboarding.md docs/architecture/error-handling.md
```

---

## RIEPILOGO

| Wave | Tema | Task | S | M | L | Stima |
|------|------|------|---|---|---|-------|
| W0 | Critical Fixes | 9 | 8 | 1 | 0 | ~1.5gg |
| W1 | Build Fixes + Hardening | 9 | 7 | 2 | 0 | ~1gg |
| W2 | God Method Extraction + Transactions | 6 | 3 | 2 | 1 | ~1.5gg |
| W3 | Hardening W2 — Test + NLP | 7 | 4 | 3 | 0 | ~2gg |
| W4 | Decision Loop — Planner (F3-01) | 6 | 3 | 2 | 0 | ~2gg |
| W5 | Decision Loop — Observe/Reflect (F3-02) | 6 | 2 | 3 | 0 | ~2gg |
| W6 | Tool Interfaces + Residuali | 5 | 2 | 3 | 0 | ~1.5gg |
| W7 | Frontend Coverage + Playwright | 4 | 1 | 3 | 0 | ~1.5gg |
| W8 | Multi-Agent A2A (F4) | 6 | 1 | 2 | 2 | ~4gg |
| W9 | Security (F5) | 6 | 1 | 4 | 0 | ~3gg |
| W10 | Evaluation + Benchmark (F7) | 7 | 3 | 2 | 1 | ~3gg |
| W-ERR | Error Handling + Dependency Audit (Gap 3+6) | 7 | 2 | 5 | 0 | ~2gg |
| W-A11Y | Accessibility + UX (Gap 7) | 5 | 1 | 4 | 0 | ~2gg |
| W-PERF | Performance Optimization Cycle (Gap 4) | 6 | 2 | 3 | 0 | ~2.5gg |
| W-DEPLOY | Deployment + DevOps (Gap 1) | 6 | 3 | 3 | 0 | ~2gg |
| W-DOCS | Documentation + Quality (Gap 2) | 6 | 2 | 4 | 0 | ~2gg |
| **Totale** | | **101** | **45** | **46** | **4** | **~34gg** |

> **Stima conservativa**: ~34 giorni lavorativi per 1 sviluppatore full-time.
> **Con parallelizzazione (sub-agenti)**: ~18-22 giorni (W-ERR ∥ W2, W-A11Y ∥ W8, W-PERF ∥ W-DEPLOY ∥ W-DOCS).
> **Multimodal (F6) escluso** — budget opzionale ~5gg.

---

## GRAFO DIPENDENZE

```
W0 (Critical Fixes)
  |
  +-- W-ERR (Error Handling + Deps, Gap 3+6) [∥ W2]
  |
W1 (Build Fixes + Hardening Strutturale)
  |
W2 (God Method + DuckDB Transactions) [∥ W-ERR]
  |
W3 (Test + NLP Cleanup)
  |
  +-- W4 (Planner F3-01) ---+
  |                          |
  +-- W5 (Observe/Reflect) --+--- W6 (Tool Interfaces Residuali)
                                    |
                                    W7 (Frontend Coverage + E2E)
                                    |       |
                                    |       W-A11Y (Accessibility, Gap 7) [∥ W8]
                                    |
                                    W8 (Multi-Agent A2A)
                                      |
                                      W9 (Security F5)
                                        |
                                        W10 (Evaluation F7)
                                          |
                              +-----------+-----------+
                              |           |           |
                              W-PERF    W-DEPLOY    W-DOCS
                            (Gap 4)    (Gap 1)     (Gap 2)
                           [paralleli, indipendenti tra loro]
```

---

## SHIP GATE CHECKLIST

- [ ] `go build ./...` pulito
- [ ] `go test -race ./...` tutti passano
- [ ] `go vet ./...` pulito
- [ ] `npx tsc --noEmit` pulito (zero errori)
- [ ] `npx vitest run --coverage` >= 60%
- [ ] `npx vite build` passa
- [ ] `npx playwright test` 21/21
- [ ] SQL injection check: zero concatenazione in query
- [ ] Race condition check: zero variabili globali senza lock
- [ ] Graceful shutdown: tutti i servizi Stop() chiamati in ordine corretto
- [ ] Decision trace: ogni decisione LAM tracciata in OTEL span
- [ ] Benchmark: P50 latenza < 5s, P95 < 15s, zero goroutine leak
- [ ] Security audit: OWASP Top 10 review documentato
- [ ] Rollback plan: migration v2→v3 documentato
- [ ] **Error handling**: zero bare errors, wrapping obbligatorio, logging strutturato (Gap 6)
- [ ] **Dependency audit**: `go list -u -m all` pulito, `npm audit` zero HIGH/CRITICAL (Gap 3)
- [ ] **Deploy**: `docker compose up` funzionante, `/readyz` 200 OK, CI/CD green (Gap 1)
- [ ] **Docs**: README, ARCHITECTURE, API docs, CHANGELOG, onboarding presenti e aggiornati (Gap 2)
- [ ] **Accessibility**: WCAG 2.1 AA verificato, keyboard nav funzionante, screen reader testato (Gap 7)
- [ ] **Performance**: re-benchmark post-ottimizzazione migliore della baseline (Gap 4)

---

## FONTI

| Agente | Task ID | Findings |
|--------|---------|----------|
| Momus | bg_d1e44d12 | 2 redundant tasks, DuckDB lock, AgentOrchestrator underspecified, build check mancanti |
| Oracle | bg_59e0d573 | query.go extraction, hybrid scorer, A2A vs Chat sync, telemetry merge, plannedTool validation |
| Metis | bg_31e647b0 | 43 TS errors, 3 Go test failures, 10-dim audit framework |
| Explore | bg_e911c93b | 8-domain gap audit, NLP dead code, SSE close order, tool_suggest ctx bug |
| Architecture Retry | bg_968da9e8 | Chat 440 lines, Config.Validate missing |
| Backend Retry | bg_afde22fd | SQL injection info_schema, toolFailCount global, LLM no retry/cache |
| Frontend Retry | bg_f4dd9384 | Polling leak, any types (40+), 88% untested components |
| DB Retry | bg_e4d0c6c4 | SQL injection scopeQuery, RWMutex wrong, no transactions |
| Gap Review (Sisyphus) | — | 8 gaps: Deploy, Docs, Deps, Perf Cycle, Interfaces, Error Handling, A11y, Generalization |

---

*Generato da Sisyphus orchestrator su aleph-v2. 101 task in 16 wave (71 originali + 30 nuovi/rinforzati). Stima ~34 giorni / 18-22 giorni con parallelizzazione.*
