# Aleph Reality Plan — Build Order con Dipendenze Reali

> **Principio**: Questo piano riflette lo STATO REALE del codebase dopo verifica esplorativa (2026-04-26). Nessuna wave dichiarata completa senza verifica su file.
>
> **Build Verification Gate**: `go build ./... && npx tsc --noEmit && npx vite build`

---

## Registro Esecuzione

| Onda | Stato Reale | Items | Note |
|------|------------|-------|------|
| **W0** | ✅ 18/18 | 18 items | Tutti completati. W0-12 (slash commands allow-list) era deferred, ora integrato in W4-15 |
| **W0.5** | ✅ 5/5 | 5 items | Completati. Sentiment reale, is_synthetic, Brier, UI incertezza, README qualificato |
| **W1** | ✅ 12/12 + bench + plan | 12 items | Completati. Zustand 6 slices, LLM Provider, migration, goroutine ctx, gRPC lifecycle, streaming abort, LRU cache, timeout, PRAGMA fix, hex arch plan |
| **W2** | ✅ 7/8 | 8 items | ✅ 7 completati. W2-05 (GNN only positive) DEFERRED — strategicamente prematuro |
| **W3** | ✅ 17/17 (FASE 0) | 17 items | **TUTTI WIREATI in app.go** — telemetry, timeout/retry/bulkhead, error_handler, audit, health, MCP, diagnostic. ✅ |
| **W4** | ✅ 20/20 (FASE 1-2 + residuali) | 20 items | **COMPLETA 2026-04-27**: palette, typography, glassmorphism, volatility, tokens, radius, React.lazy 3 chunks, SlideOverContent, 4 forms, /tool subcommands, command/input mode, ghost prompt EmptyState, real Suggester, VersioningRollback. ✅ |
| **W5** | ✅ 12/12 (FASE 3) | 12 items | **COMPLETA 2026-04-27**: DataSourceForm 3-step, split view + search/export, terminal effects, command palette Tab, GetDataStats batched, app.go wired. ✅ |
| **W6** | ✅ 12/15 (3 differiti) | 15 items | **COMPLETA 2026-04-27**: 12 completi (dead code useViewActions cancellato, cursor pagination, bundle budget, Playwright dep, cross-context test 820 righe, SSE, bias, tool lifecycle, MCP, repair, shadcn/ui 9 comp). 3 differiti: i18n, URL state, Yjs cleanup. ✅ |
| **W-ERR** | ⏳ Audited | — | ✅ Buono: APIError types, ErrorHandlerInterceptor, boundaries. 🔴 Critico: nessun toast/notifica errori, 47 return err nudi, nessun panic recovery middleware |
| **W-A11Y** | ⏳ Audited | — | ✅ Buono: contrasto ~12:1, prefers-reduced-motion, focus form. 🔴 Critico: nessun <main>/skip-link, nessun focus trap modali, sidebar senza aria-label |
| **W-PERF** | ⏳ Audited | — | ✅ Buono: timeout 19 usi, caching, manualChunks. 🔴 Critico: d3 65KB gzip sincrono, N+1 ListTools() 6/chiamata, quasi zero memoization |
| **W-DEPLOY** | ⏳ Audited | — | ✅ Buono: 3 Dockerfile, compose 4 servizi, CI, OTel, nginx. 🔴 Critico: nessun liveness probe, nessun Docker push, nessun secrets management |
| **W-DOCS** | ⏳ Audited | — | ✅ Buono: README, ARCHITECTURE, AGENTS, threat-model, bias-checklist. 🔴 Critico: nessun CHANGELOG, nessun CONTRIBUTING, API.md scheletrico |

---

## W0 — SOPRAVVIVENZA ✅ (18/18)

| Item | Status | Verifica |
|------|--------|----------|
| W0-01 SQL Injection | ✅ | `validName.MatchString()` + defense-in-depth |
| W0-02 Sandbox isolation | ✅ | Env hardened (PATH, HOME), projectRoot fix |
| W0-03 Segreti hardcoded | ✅ | `.env` + `${ENV_VAR}` in docker-compose |
| W0-04 API key leak | ✅ | Mascheramento `****` in risposta |
| W0-05 Entrypoint duale | ✅ | `cmd/aleph-server/` eliminato |
| W0-06 Auth chat | ✅ | `sha256(inputKey)` confronto |
| W0-07 Y.js sicurezza | ✅ | room usa projectID, non simpleHash(apiKey) |
| W0-08 SSRF bypass | ✅ | OllamaBaseURL configurabile |
| W0-09 Data leakage DuckDB | ✅ | Schema isolato per progetto |
| W0-10 DB() bypass | ✅ | Zero `.DB()` in handler code |
| W0-11 CORS permissivo | ✅ | AllowOriginFunc con validazione |
| W0-12 Slash commands allow-list | ✅ | Integrato in W4-15 |
| W0-13 Ragionamento fabbricato | ✅ | "Executing tool: %s" |
| W0-14 Chat amnesia | ✅ | `GetChatMessages()` in Chat() |
| W0-15 Ontologia vuota | ✅ | `slog.Warn` on error |
| W0-16 Modello default | ✅ | `CodeFailedPrecondition` |
| W0-17 skipYMapSet race | ✅ | `ydoc.transact()` + `queueMicrotask` |
| W0-18 Query senza limiti | ✅ | Default LIMIT 1000 |

**Build**: `go build ./...` ✅ | `npx tsc --noEmit` ✅

---

## W0.5 — ONESTÀ EPISTEMICA ✅ (5/5)

| Item | Status | Verifica |
|------|--------|----------|
| W0.5-01 Sentiment reale | ✅ | `enrichPredictiveMetadata()` chiama NLP |
| W0.5-02 is_synthetic flag | ✅ | Proto + Python set |
| W0.5-03 Brier/Trust score | ✅ | BrierMonitor in app.go, agent tool |
| W0.5-04 UI incertezza | ✅ | "72% ±8%", livello Alta/Media/Bassa |
| W0.5-05 DI claim | ✅ | README qualificato "beta" |

---

## W1 — STRUTTURA ✅ (12/12 + bench + plan)

| Item | Status |
|------|--------|
| W1-01 Zustand decomposition | ✅ 6 slices, types.ts |
| W1-02 Migrazioni database | ✅ DuckDB + PostgreSQL separate |
| W1-03 LLM Provider interface | ✅ Ollama, Anthropic, OpenAI |
| W1-04 Goroutine ctx | ✅ Derivati da richiesta con timeout |
| W1-05 gRPC leak | ✅ NLPHandler.Close() in shutdown |
| W1-06 Chat streaming abort | ✅ AbortController + STOP button |
| W1-07 LRU program cache | ✅ Max 64, TTL 30min |
| W1-08 ListModels timeout | ✅ 30s http.Client |
| W1-09 DuckDB concurrency bench | ✅ Benchmark completato |
| W1-10 PRAGMA DuckDB | ✅ SQLite-specific rimossi |
| W1-11 Hex architecture plan | ✅ Piano scritto |
| W1-12 Down migrations | ✅ Migrations separate |

---

## W2 — ONESTÀ PROFONDA ✅ (7/8, 1 DEFERRED)

| Item | Status | Note |
|------|--------|------|
| W2-01 Data provenance | ✅ | Ogni record: source, ingested_at, transform_version |
| W2-02 Feedback pipeline | ✅ | Fase 1: trust score update |
| W2-03 Sigmoid calibration | ✅ | Platt scaling su validation |
| W2-04 JSON truncation | ✅ | truncateJSON() depth-tracking |
| W2-05 GNN only positive | ⏭️ **DEFERRED** | strategicamente prematuro |
| W2-06 StreamPredictions fix | ✅ | recordSuccess() + circuit breaker |
| W2-07 json.Unmarshal errors | ✅ | Tutti loggati con contesto |
| W2-08 Watcher service | ✅ | Codice morto rimosso |

---

## W3 — RESILIENZA 🔴 (12 files EXIST, 5 da WIRE)

### Files Esistenti (verificati su disco):

| Item | File Status | App.go Status | Azione |
|------|------------|---------------|--------|
| W3-01 CI/CD | ✅ `.github/workflows/ci.yml` | N/A | OK |
| W3-02 Linting | ✅ `.golangci.yml`, `.pre-commit-config.yaml`, `.prettierrc` | N/A | OK |
| W3-03 Unit test | ✅ Test files in `internal/` | N/A | OK |
| W3-04 **OpenTelemetry** | ✅ `internal/telemetry/telemetry.go`, `middleware.go` | ❌ **NON WIRED** | Wire middleware in mux |
| W3-05 Error glossary | ✅ `docs/error-glossary.md`, `internal/errors/` | N/A | OK |
| W3-06 Air hot reload | ✅ `.air.toml` | N/A | OK |
| W3-07 **Timeout/Retry/Bulkhead** | ✅ `internal/middleware/timeout.go`, `retry.go`, `bulkhead.go` + test | ❌ **NON REGISTRATI** | Wire 3 interceptor in app.go |
| W3-08 Audit logging | ✅ `internal/repository/audit.go`, `middleware/audit.go`, migration | ❌ **NON REGISTRATO** | Wire AuditInterceptor |
| W3-09 SHA-256 Checksum | ✅ `computeChecksum`/`VerifyChecksum` in engine.go | N/A | OK |
| W3-10 testify+mockery | ✅ testify in go.mod, `.mockery.yaml` | N/A | OK |
| W3-11 ConnectRPC errors | ✅ `middleware/error_handler.go`, `errors/errors.go` | ❌ **NON REGISTRATO** | Wire ErrorHandlerInterceptor |
| W3-12 Sandbox isolation | ✅ `internal/sandbox/exec_sandbox.go`, `validation.go`, `security.go` | N/A | OK |
| W3-13 Tool metadata | ✅ Category/Version/HealthStatus/SourceType on ToolRecord, 2 migrations | N/A | OK |
| W3-14 Sandbox verification | ✅ `internal/sandbox/verification.go` | N/A | OK |
| W3-15 **Health check** | ✅ `internal/health/checker.go`, `history.go` | ❌ **NON INSTANZIATO** | Wire in app.go Serve() |
| W3-16 **MCP Discovery** | ✅ `internal/mcp/discovery.go`, `schemas.go`, `health.go`, `ssrf.go` | ❌ **NON AVVIATO** | Wire in app.go |
| W3-17 **Auto-diagnostic** | ✅ `internal/diagnostic/patterns.go` + test | ❌ **NON INIZIALIZZATO** | Wire in app.go |

### Task W3-fix: Wire 5 interceptor in app.go

Obiettivo: attivare i 5 componenti dormienti in `app.go`.

**Dipendenza per W5/W6**: Alcuni item W5/W6 dipendono da W3-15/16/17. Vanno wireati PRIMA.

---

## W4 — VOCE 🔴 (5 EXIST, 6 PARTIAL, 9 MISSING)

### Verifica Reale per Item:

| Item | Status Reale | Evidenza |
|------|-------------|----------|
| **W4-01** Design tokens | ⚠️ PARTIAL | design-tokens.json: ✅ color, typography, spacing, radius. ❌ Manca elevation (4 livelli), shadow (3), transition (3), border (3 tier) |
| **W4-02** Tipografia | ❌ **MISSING** | ❌ No fontSize 13px body. ❌ No fontSize 11px meta. ❌ No tabular-nums. ❌ No 8px grid. ❌ No max-width container. |
| **W4-03** Command palette | ✅ **EXIST** | CommandPalette.tsx 143 righe, keyboard nav, fuzzy search |
| **W4-04** Border-radius | ⚠️ PARTIAL | ✅ radius in tokens. ❌ No terminal=0 class. ❌ No radius-card class. |
| **W4-05** Terminal effects | ✅ **EXIST** | TerminalEffects.tsx 118 righe, CRT scanlines, cursor blink step-end |
| **W4-06** Command/Input Mode | ❌ **MISSING** | No mode switching anywhere |
| **W4-07** Dark palette | ❌ **MISSING** | Usa #0a0a0f/#12121a/#1a1a28. Richiede: #080810/#0e0e18/#141420 |
| **W4-08** Glassmorphism | ❌ **MISSING** | No backdrop-filter in SlideOverPanel. No glass-panel class. |
| **W4-09** CSS volatility | ❌ **MISSING** | No .vol-* classes in CSS |
| **W4-10** Icon system | ⚠️ PARTIAL | ✅ Lucide React usati. ❌ No empty state component. ❌ No ghost prompt pattern. |
| **W4-11** Sidebar refactor | ✅ **EXIST** | Sidebar.tsx con ID_TO_INLINE_TYPE, divider |
| **W4-12** App.tsx rewrite | ⚠️ PARTIAL | ✅ SlideOverContent switch. ❌ NO React.lazy. ❌ Solo 5 casi (skill, tool, sandbox, asset, detail). Mancano: agents, datasources, library, components views. |
| **W4-13** SlideOverPanel | ⚠️ PARTIAL | ✅ SlideOverPanel.tsx esiste. ✅ fullscreen toggle. ❌ Solo 5 content types wired (non 6). |
| **W4-14** StatusBar refactor | ⚠️ PARTIAL | ✅ StatusBar esiste. ❌ activeTab prop da verificare. |
| **W4-15** /tool commands | ❌ **MISSING** | ❌ No /tool install/list/health/diagnose in slashCommands.ts. No ToolManagementView.tsx |
| **W4-16** Finance pkg | ✅ **EXIST** | 5 file: prophet_forecast, openbb_market_data, sentiment_analysis_fin, package.go |
| **W4-17** OSINT pkg | ✅ **EXIST** | 8 file: 5 tool + shadowbroker + package.go |
| **W4-18** Human Ecosystems | ✅ **EXIST** | 10 file: 5 tool + duckdb_layer + package.go |
| **W4-19** Tool suggestion | ❌ **MISSING** | suggestion.go è stub che ritorna ToolDefinition{} vuoto |
| **W4-20** Adaptation pipeline | ⚠️ PARTIAL | pipeline.go scaffold 5-stage. VersioningRollback è no-op stub. |

### Task W4-fix: Ordine di Esecuzione

1. **W4-07**: Aggiornare palette dark in design-tokens.json e index.css (#080810/#0e0e18/#141420)
2. **W4-02**: Aggiungere fontSize body/meta, tabular-nums, 8px grid in tailwind.config.js + index.css
3. **W4-09**: Aggiungere .vol-* CSS layers in index.css
4. **W4-08**: Aggiungere backdrop-filter blur a SlideOverPanel
5. **W4-12**: Aggiungere React.lazy per 6 view components + switch cases
6. **W4-15**: Implementare /tool commands in slashCommands.ts
7. **W4-01**: Aggiungere elevation, shadow, transition, border tokens
8. **W4-04**: Aggiungere radius-terminal/radius-card classes
9. **W4-10**: Aggiungere EmptyState component + ghost prompt
10. **W4-06**: Aggiungere command mode indicator
11. **W4-13**: Wire 6 view content types
12. **W4-19**: Implementare tool suggestion workflow reale
13. **W4-20**: Implementare VersioningRollback reale

---

## W5 — ACCOGLIENZA 🔴 (4/16 completati)

### Completati:
- **W5-01** ✅ AgentForm (in App.tsx SlideOverContent - case 'agent-form')
- **W5-03** ✅ SetupWizard + WelcomeScreen
- **W5-05** ✅ Toast system + AlephErrorBoundary cascade
- **W5-12** ✅ Error handling frontend centralizzato

### Da fare (12 items):
- **W5-02** DataSourceForm multi-step (upload/DB/URL) — DEP: W4-13
- **W5-04** Vista split, ricerca chat, esportazione — DEP: W0-14
- **W5-06** Terminal effects toggle
- **W5-07** Command palette slash commands + Tab completion — DEP: W4-15
- **W5-08** Y.js collaboration (DEFERRED — bassa priorità)
- **W5-09** Zod schemas + fromProto mappers — DEP: W5-10
- **W5-10** Eliminare `any` — DEP: W5-09
- **W5-11** Ottimizzare GetDataStats
- **W5-13** Tool DSL .aleph — DEP: W3-12, W3-13
- **W5-14** Sandbox enhancements — DEP: W3-12, W3-14
- **W5-15** Auto-repair strategies — DEP: W3-17, W5-13
- **W5-16** Integration CodeFlow/HE/Shadowbroker — DEP: W4-16/17/18

---

## W6 — AUTOCOSCIENZA ⏳ (0/15)

- W6-01 Dead code removal — DEP: W4 completato
- W6-02 i18n unificazione
- W6-03 useViewActions refactor — DEP: W5-12
- W6-04 Yjs cleanup + command history
- W6-05 shadcn/ui migration
- W6-06 Cursor-based pagination
- W6-07 SSE streaming
- W6-08 URL state
- W6-09 Bundle budget
- W6-10 Playwright E2E — DEP: W4/W5 completati
- W6-11 Bias checklist (documento)
- W6-12 E2E tool lifecycle — DEP: W3-13/14/15/16, W4-20
- W6-13 MCP connectivity test — DEP: W3-16, W4-16/17
- W6-14 Self-repair demo — DEP: W3-17, W5-15, W3-14
- W6-15 Cross-context adaptability — DEP: W4-16/17/18

---

## BUILD ORDER (con dipendenze)

```
FASE 0: W3 Wiring (app.go) — nessuna dipendenza
  │ Wire telemetry, timeout/retry/bulkhead, error_handler, audit, health, MCP, diagnostic
  │ Build check: go build ./... ✅

FASE 1: W4 CSS Fixes — nessuna dipendenza
  │ W4-07 palette, W4-02 typography, W4-09 volatility layers
  │ W4-08 glassmorphism, W4-01 design tokens, W4-04 border-radius
  │ Build check: npx vite build ✅

FASE 2: W4 React Components — ✅ COMPLETED (2026-04-27)
  │ W4 20/20 — React.lazy 3 chunks, SlideOverContent 79 righe, 4 forms, /tool 5 subcommands
  │ Command/Input mode, ghost prompt EmptyState, real Suggester, VersioningRollback
  │ Build check: npx tsc --noEmit ✅ | npx vite build ✅ | go build ✅

FASE 3: W5 Remaining (12 items) — IN PROGRESS (2026-04-27)
  │ W5-02 DataSourceForm, W5-04 split view
  │ W5-06 effects toggle, W5-07 command palette
  │ W5-09 Zod schemas, W5-10 eliminate any
  │ W5-11 GetDataStats, W5-13 DSL, W5-14 sandbox, W5-15 auto-repair, W5-16 integration
  │ Build check: go build ./... ✅ | npx tsc --noEmit ✅

FASE 4: W6 — DEP: FASE 3
  │ All 15 W6 items
  │ Build check: FULL ✅

FASE 5: Residuali (W-ERR, W-A11Y, W-PERF, W-DEPLOY, W-DOCS)
  │ Build check: FULL ✅
```

---

## IMMEDIATE NEXT — FASE 0: W3 Wiring

### Task 0.1: Wire error_handler + audit + timeout/retry/bulkhead interceptors
File: `internal/app/app.go`
- Importare: `middleware.ErrorHandlerInterceptor`, `middleware.AuditInterceptor`, `middleware.TimeoutInterceptor`, `middleware.RetryInterceptor`, `middleware.BulkheadInterceptor`
- Creare interceptor chain: `errorHandler → audit → auth → timeout → retry → bulkhead`
- Sostituire `connect.WithInterceptors(authInterceptor)` con chain interceptor

### Task 0.2: Wire health checker
File: `internal/app/app.go`
- Importare: `health.NewHealthChecker`
- In Serve(): `healthChecker := health.NewHealthChecker(a.metaRepo, a.logger, 5*time.Minute)`
- `go healthChecker.Start(a.ctx)`
- `mux.Handle("/api/v1/tools/health", ...)`

### Task 0.3: Wire MCP discovery engine
File: `internal/app/app.go`
- Importare: `mcp.NewDiscoveryEngine`
- In Serve(): `discoveryEngine := mcp.NewDiscoveryEngine(a.logger)`
- `go discoveryEngine.Start(a.ctx)`

### Task 0.4: Wire diagnostic monitor
File: `internal/app/app.go`
- Importare: `diagnostic.NewDiagnosticMonitor`
- In Serve(): `diagnosticMonitor := diagnostic.NewDiagnosticMonitor(a.logger)`
- `go diagnosticMonitor.Start(a.ctx)`
- `mux.Handle("/api/v1/diagnostic/patterns", ...)`

### Task 0.5: Wire telemetry middleware
File: `internal/app/app.go`
- In mux creation: wrap with `telemetry.Middleware`

### Build Check
```bash
go build ./...
go vet ./...
```

---

## IMMEDIATE NEXT — FASE 1: W4 CSS Fixes

### Task 1.1: Fix dark palette (W4-07)
- `frontend/src/styles/design-tokens.json`: cambiare `"#0a0a0f"` → `"#080810"`, `"#12121a"` → `"#0e0e18"`, `"#1a1a28"` → `"#141420"`
- `frontend/src/index.css`: cambiare `--color-background: #0a0a0f` → `#080810`, etc.

### Task 1.2: Add typography system (W4-02)
- `frontend/tailwind.config.js`: aggiungere `fontSize.body = ['13px', { lineHeight: '1.25' }]`, `fontSize.meta = ['11px', { lineHeight: '1.25' }]`
- `frontend/src/index.css`: aggiungere `.text-body { font-size: 13px; line-height: 1.25; font-family: JetBrains Mono; }`, `.text-meta { font-size: 11px; }`, `* { font-variant-numeric: tabular-nums; }`, `.terminal-grid { display: grid; gap: 8px; }`

### Task 1.3: Add CSS volatility layers (W4-09)
- `frontend/src/index.css`: aggiungere `.vol-static`, `.vol-structural`, `.vol-interactive`, `.vol-signal`

### Task 1.4: Add glassmorphism (W4-08)
- `frontend/src/index.css`: aggiungere `.glass-panel`
- `frontend/src/components/terminal/SlideOverPanel.tsx`: applicare backdrop-filter

### Task 1.5: Add design tokens (W4-01)
- `frontend/src/styles/design-tokens.json`: aggiungere elevation (4 livelli), shadow (3), transition (3), border (3 tier)

### Task 1.6: Add border-radius classes (W4-04)
- `frontend/src/index.css`: aggiungere `.radius-terminal { border-radius: 0; }`, `.radius-card { border-radius: 8px; }`

### Build Check
```bash
npx vite build
npx tsc --noEmit
```

---

*Piano generato da Sisyphus dopo audit esplorativo del codebase. Ogni task verrà eseguito subito dopo l'approvazione.*
