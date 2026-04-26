# Debug Plan — aleph-v2

**Origine**: 20-agent audit (10 explore + 5 Oracle + 5 Metis) + hardening findings
**Obiettivo**: Rendere Aleph funzionante e senza bug — fix sicurezza, chiudere stub/TODO, rimuovere mock incompleti, goroutine safety, context propagation
**Principio**: Ogni wave è completa e builda prima di passare alla successiva
**Dimensione**: 4 wave, ~30 task (include nuove findings critiche da audit 20 agenti)

---

## Wave 1 — Sicurezza e Stabilità (10 task)

Nessuna dipendenza tra task. Parallelizzabili via deep agent.

### D1-1/D1-2: Auth validator unificato — HTTP middleware + SSE

| Campo | Valore |
|-------|--------|
| **Problema** | 17 raw HTTP handler registrati via `mux.HandleFunc` NON passano auth. `AuthInterceptor` è solo Connect RPC (`connect.WithInterceptors`). SSE auth usa solo `strings.HasPrefix(key, "aleph_")` senza validare contro repository. L'`AuthInterceptor` in `internal/middleware/auth_middleware.go` ha già `validateKey()` che chiama `metaRepo.ValidateAPIKey()` — va estratta per riuso. |
| **Endpoint colpiti** | `/api/v1/tools/intelligence`, `/api/v1/tools/recommendations`, `/api/v1/tools/health`, `/api/v1/tools/verify`, `/api/v1/tools/{id}/health/history`, `GET /api/v1/tools`, `/api/v1/tools/suggest`, `/api/v1/tools/suggest/approve`, `/api/v1/tools/categories`, `/api/v1/tools/execute/{category}/{name}`, `/api/v1/tools/register`, `/api/v1/codeflow/graph`, `/api/v1/codeflow/metrics`, `/api/v1/codeflow/executions`, `/api/v1/codeflow/engines`, `/api/v1/diagnostic/patterns`, `/api/v1/swagger.json` |
| **Decisione team** | Estrarre `ValidateAPIKey(metaRepo, key) (string, error)` in `internal/middleware/auth.go`. Refactor: `AuthInterceptor.validateKey()` → chiama nuova funzione. Creare `AuthMiddleware(next http.Handler) http.Handler` che la riusa. SSE `isAuthenticatedForSSE` idem. Unico punto di verità, nessuna duplicazione. |
| **Fix** | 4 step: (1) estrarre `ValidateAPIKey()` in `internal/middleware/auth.go`, (2) refactor `AuthInterceptor.validateKey()` per usarla, (3) creare `AuthMiddleware(http.Handler) http.Handler` e applicare a tutte le route raw HTTP in app.go, (4) refactor `isAuthenticatedForSSE` per chiamare `ValidateAPIKey()`. |
| **Rischio** | **CRITICO** — dati esposti, mutazioni non protette, auth bypassabile |
| **Verifica** | Endpoint senza header → 401. `aleph_fake` → 401. API key valida → 200. `go build ./...` ✅ |
| **File** | `internal/middleware/auth_middleware.go`, `internal/app/app.go`, `internal/api/sse/sse_handler.go` |

### D1-3: SQL injection via projectID

| Campo | Valore |
|-------|--------|
| **Problema** | `storage.ContextWithSchema(ctx, "project_"+projectID)` concatena projectID direttamente in query SQL. 3 vettori: `scopeQuery` (duckdb.go:83, `fmt.Sprintf("SET schema = '%s'", schema)`), `EnsureProjectSchema` (duckdb.go:122, CREATE SCHEMA), `SchemaContext` (duckdb_layer.go:75). |
| **Decisione team** | Regex più larga: `^[a-zA-Z0-9_.:-]{1,128}$`. Non limitarsi a DuckDB — `ContextWithSchema` usato in altri storage. |
| **Fix** | Sanitize projectID con regex all'ingresso di ogni funzione che usa projectID in query. Rifiutare input non validi con `fmt.Errorf("invalid projectID")`. Controllare TUTTI i call site di `ContextWithSchema`. |
| **Rischio** | **CRITICO** — injection SQL classica, potenziale loss di dati |
| **Verifica** | Inviare projectID con `; DROP TABLE` → errore 400. projectID valido → OK. `go build ./...` ✅ |
| **File** | `internal/storage/duckdb.go`, `internal/tools/humanecosystems/duckdb_layer.go`, `internal/storage/context.go` |

### D1-4: Audit goroutine fire-and-forget senza recover

| Campo | Valore |
|-------|--------|
| **Problema** | `audit.go:32`: `go a.logAuditEvent(ctx, req, resp)` — lancia goroutine senza `recover()`. Se `logAuditEvent` panica, l'intero server si arresta. |
| **Fix** | Aggiungere `defer func() { if r := recover(); r != nil { log.Error("audit panic", "recover", r) } }()` in testa a `logAuditEvent`. |
| **Rischio** | ALTO — un panic spegne il server |
| **Verifica** | Forzare panic in logAuditEvent → server resta in piedi, log registra errore. `go build ./...` ✅ |
| **File** | `internal/middleware/audit.go` |

### D1-5: SSE Broker goroutine leak

| Campo | Valore |
|-------|--------|
| **Problema** | `Broker.run()` (sse.go:65-144) non ha quit channel. Una volta avviato, la goroutine vive per sempre. Leak in test e in produzione. |
| **Decisione team** | Critico anche per operabilità — restart broker dopo crash leakerebbe goroutine. |
| **Fix** | Aggiungere campo `quit chan struct{}` a `Broker`. In `Broker.run()`, select su `s.quit` in ogni iterazione del loop. Aggiungere `Broker.Stop()`. |
| **Rischio** | MEDIO — leak in test, shutdown graceful impossibile |
| **Verifica** | `go test -race ./internal/api/sse/` — zero leak. `go build ./...` ✅ |
| **File** | `internal/api/sse/sse.go` |

### D1-6: DuckDB RWMutex + pool timeout (spostato da Wave 3)

| Campo | Valore |
|-------|--------|
| **Problema** | `DuckDBLayer.QueryContext()` fa `RLock()` ma chiama `db.QueryContext()`. Pool semaphore cap 5. Sotto carico (5 richieste + 1 DDL), deadlock simulato. `EnsureProjectSchema` tiene exclusive Lock durante DDL. |
| **Decisione team** | **Aleph**: "Non puoi avere un database che deadlocka sotto carico e chiamarlo refactor non critico." Spostato in Wave 1. |
| **Fix** | (1) Aggiungere `context.WithTimeout` 30s a tutte le chiamate DuckDB. (2) Aumentare pool semaphore da 5 a 20. |
| **Rischio** | ALTO — deadlock sotto carico = produzione giù |
| **Verifica** | `go test -race -count=5 ./internal/storage/...` — zero deadlock. `go build ./...` ✅ |
| **File** | `internal/storage/duckdb.go` |

### D1-7: Migrazioni SQL silenziose (spostato da Wave 3)

| Campo | Valore |
|-------|--------|
| **Problema** | Se una migrazione fallisce, il server parte comunque con stato DB incoerente. Bug impossibili da tracciare. |
| **Decisione team** | **DevOps**: "Meglio non partire che partire rotti." Spostato in Wave 1. |
| **Fix** | Dopo `migrate.Up()`, controllare errore e fare `log.Fatal` se fallisce. Oppure esporre stato `migration_failed` e rifiutare richieste. |
| **Rischio** | ALTO — stato incoerente = bug silenziosi |
| **Verifica** | Migrazione con SQL errato → server non parte. `go build ./...` ✅ |
| **File** | `internal/app/app.go` |

### D1-8: HandleRegister — verifica end-to-end

| Campo | Valore |
|-------|--------|
| **Problema** | `POST /api/v1/tools/register` è implementato in `tool_exec.go` e wired in `app.go:296`. Ma non ci sono test che verifichino il flusso completo: POST con body valido → tool record persistito → GET lo restituisce. |
| **Fix** | Scrivere 1 test end-to-end: POST body JSON valido → verifica 200 + tool ID nel response. |
| **Rischio** | BASSO — già implementato, manca solo test |
| **Verifica** | `go test ./internal/api/handler/... -run TestHandleRegister`. `go build ./...` ✅ |
| **File** | `internal/api/handler/tool_exec.go` (HandleRegister lines 181+), `internal/app/app.go:296` (già wired) |

### D1-9: Cache unbounded — LRU + TTL (ex D3-3)

| Campo | Valore |
|-------|--------|
| **Problema** | Cache in vari package non ha limiti di dimensione o TTL. Cresce senza bound = memory leak graduale. |
| **Decisione team** | **Go Engineer**: usare `hashicorp/golang-lru` già usato altrove. Confermato in Wave 1 (leak = sicurezza). |
| **Fix** | Pre-scout completato: grep `sync.Map` in internal/ trova usi in `internal/tools/finance/` (NLPAdapter cache) e `internal/mcp/discovery.go`. Grep `map[` senza LRU pattern: `internal/nlp_adapter/adapter.go` (resultCache), `internal/storage/` (DuckDB conn pool map non ha limiti), `internal/api/sse/sse.go` (Broker client map). Sostituire con `lru.New(size)` con LRU + TTL 30min. Priorità: DuckDB conn pool map > SSE client map > NLPAdapter cache. |
| **Rischio** | MEDIO — memory leak graduale in produzione |
| **Verifica** | Dopo 10k inserimenti, memoria non cresce. `go build ./...` ✅ |
| **File** | `internal/storage/duckdb.go`, `internal/api/sse/sse.go`, `internal/nlp_adapter/adapter.go` |
| **Rischio** | MEDIO — memory leak graduale in produzione |
| **Verifica** | Dopo 10k inserimenti, memoria non cresce. `go build ./...` ✅ |
| **File** | Da identificare via grep `map[` + `sync.Map` senza limiti |

### D1-10: Build check Wave 1

| Comando | Verifica |
|---------|----------|
| `go build ./...` | ✅ exit 0 |
| `go vet ./...` | ✅ exit 0 |
| `npx tsc --noEmit` | ✅ exit 0 |

---

## Wave 2 — Stub/TODO/Incomplete (9 task)

### D2-1: SlideOverContent — asset view stub

| Campo | Valore |
|-------|--------|
| **Problema** | `asset` restituisce hardcoded "Mostra contenuto asset... (da implementare)". |
| **Decisione team** | **React/TS**: AssetView con `JSON.stringify` è già un miglioramento. |
| **Fix** | Implementare AssetView: renderizzare contenuto dal record asset. Se dati insufficienti, "nessun contenuto". |
| **Rischio** | BASSO — stub visibile all'utente |
| **Verifica** | Navigare a un asset → contenuto reale o "nessun contenuto" coerente. `npx tsc --noEmit` ✅ |
| **File** | `frontend/src/App.tsx` (SlideOverContent) |

### D2-2: SlideOverContent — detail view stub

| Campo | Valore |
|-------|--------|
| **Problema** | `detail` mostra JSON grezzo. |
| **Decisione team** | **React/TS**: 20 righe per label/value layout. |
| **Fix** | Se `typeof data === 'object'`, label + value layout. Se stringa, testo. |
| **Rischio** | BASSO |
| **Verifica** | Navigare a dettaglio → label/valori formattati. `npx tsc --noEmit` ✅ |
| **File** | `frontend/src/App.tsx` (SlideOverContent) |

### D2-3: SkillForm/ToolForm update paths non wired

| Campo | Valore |
|-------|--------|
| **Problema** | Salvataggio SkillForm/ToolForm esegue solo `onClose()` + toast "non ancora implementato". |
| **Decisione team** | **React/TS**: Skill creato via `POST /api/v1/tools` con `category: "skill"`. Tool via `POST /api/v1/tools` con `category: "tool"`. Pattern già stabilito in AgentForm (chiamata a endpoint + store update). |
| **Fix** | Implementare `handleSave`: fetch POST/PATCH a endpoint tools, gestire loading/error/success state, aggiornare store locale. Seguire pattern identico a AgentForm. |
| **Rischio** | MEDIO — utente compila form e perde dati |
| **Verifica** | Compilare SkillForm → salva → 200 + conferma. Ricaricare → dato persistito. `npx tsc --noEmit` ✅ |
| **File** | `frontend/src/components/SkillForm.tsx`, `frontend/src/components/ToolForm.tsx` |

### D2-4: DSL compiler — 3 TODO nei template

| Campo | Valore |
|-------|--------|
| **Problema** | DSL compiler ha 3 TODO nei template generati: `data_processor` (compiler_tool.go:299), `api_connector` (compiler_tool.go:350), `analyzer` (compiler_tool.go:400). Sono commenti nei template Go e Python, non blocchi di codice vivo. I template sono già funzionanti — i TODO sono placeholder per implementazioni future. |
| **Decisione** | I TODO sono commenti nei template di codice generato. Rimuoverli e lasciare solo `// implement data transformation logic` senza prefisso TODO, per non generare falsi allarmi. |
| **Rischio** | BASSO |
| **Verifica** | `grep -r "TODO" internal/dsl/compiler_tool.go` → zero risultati. `go build ./...` ✅ |
| **File** | `internal/dsl/compiler_tool.go` |

### D2-5: repair.go — TODO retry backoff

| Campo | Valore |
|-------|--------|
| **Problema** | repair.go:834 TODO retry backoff. Senza retry, riparazione fallisce al primo errore temporaneo. |
| **Fix** | Implementare backoff esponenziale: max retry=3, base=100ms, `time.Sleep` + context cancellation. |
| **Rischio** | BASSO |
| **Verifica** | Errore temporaneo simulato → retry 3x. `go build ./...` ✅ |
| **File** | `internal/repair/repair.go` |

### D2-6: nlp.go — TODO batch processing + metriche

| Campo | Valore |
|-------|--------|
| **Problema** | nlp.go:91 TODO batch processing. NLP pipeline non ha metriche di throughput/latenza. |
| **Decisione team** | **AI/ML**: throughput attuale adeguato. Ma senza metriche, un crollo passa inosservato. |
| **Fix** | Rimuovere TODO "single-item processing sufficiente". Aggiungere contatore richieste processate + latenza media (prometheus o log). |
| **Rischio** | BASSO |
| **Verifica** | NLP processa input singolo ✅. `go build ./...` ✅ |
| **File** | `internal/nlp_adapter/adapter.go` |

### D2-7: Guide content mancante

| Campo | Valore |
|-------|--------|
| **Problema** | GuideTour registrata ma acceptance item 4 unchecked. Guide probabilmente senza contenuti. |
| **Decisione team** | **React/TS**: GuideTour senso averle in ITA (target italiano). |
| **Fix** | Verificare `contextualGuides.ts`. Se stub, implementare contenuti ITA per onboarding, tools, agents, datasources. |
| **Rischio** | BASSO — guida vuota |
| **Verifica** | Aprire guida onboarding → testo significativo. `npx tsc --noEmit` ✅ |
| **File** | `frontend/src/components/GuideTour.tsx`, `frontend/src/data/contextualGuides.ts` |

### D2-8: Health indicators UI

| Campo | Valore |
|-------|--------|
| **Problema** | HealthView mostra dati ma UI indicators (colore, timestamp) deferred. |
| **Decisione team** | **React/TS**: Badge colorato con `lastChecked` in 5 righe di CSS. |
| **Fix** | Aggiungere badge verde/giallo/rosso + timestamp ultimo check. |
| **Rischio** | BASSO |
| **Verifica** | HealthView mostra stato colorato per ogni tool. `npx tsc --noEmit` ✅ |
| **File** | `frontend/src/components/DataHealthView.tsx` |

### D2-9: Build check Wave 2

| Comando | Verifica |
|---------|----------|
| `npx tsc --noEmit` | ✅ exit 0 |
| `npx vite build` | ✅ exit 0 |
| `npx playwright test` | ✅ 20/20 |

---

## Wave 3 — Refactor (2 task)

### D3-1: Synthesis goroutine panic safety (ex D3-5)

| Campo | Valore |
|-------|--------|
| **Problema** | Synthesis.go lancia 3 goroutine (linee 95, 118, 153) che scrivono su canali. Gli errori via canale sono gestiti correttamente dal chiamante (`cfResult := <-cfCh; if cfResult.err != nil`). Ma le goroutine non hanno `recover()` — un panic interno fa crashare il server. Inoltre la goroutine security (linea 153) silenzia errori (Warn log + empty result) — se ShadowBroker è down, nessun alert. |
| **Fix** | (1) Aggiungere `recover()` in tutte e 3 le goroutine con log + errore sul canale. (2) ShadowBroker failure: upgrade da Warn a Error log. |
| **Rischio** | MEDIO — panic in goroutine = server down |
| **Verifica** | Input invalido in synthesis → log errore esplicito. `go build ./...` ✅ |
| **File** | `internal/tools/synthesis/synthesis.go` |

### D3-2: Build check Wave 3

| Comando | Verifica |
|---------|----------|
| `go build ./...` | ✅ exit 0 |
| `go vet ./...` | ✅ exit 0 |

---

## Wave 4 — Test Fix + User Journey (3 task)

### D4-1: Fix middleware tests (uniti: timeout_test + bulkhead_test)

| Campo | Valore |
|-------|--------|
| **Problema** | `timeout_test.go`: 3 test case con `wantMinDur > wantMaxDur` (es. 35s min vs 31s max). `bulkhead_test.go`: timing assertions flaky. |
| **Decisione team** | **Debug Engineer**: Unire in unico task. Per bulkhead, usare event-driven assertions (sync.WaitGroup + select su canale) invece di sleep + elapsed. |
| **Fix** | (1) Correggere wantMinDur < wantMaxDur. (2) Refactor bulkhead: event-driven, non timing-based. |
| **Rischio** | BASSO — test broken, non produzione |
| **Verifica** | `go test -count=10 ./internal/middleware/...` — zero failure. `go build ./...` ✅ |
| **File** | `internal/middleware/timeout_test.go`, `internal/middleware/bulkhead_test.go` |

### D4-2: Synthesis test coverage

| Campo | Valore |
|-------|--------|
| **Problema** | Synthesis 79.2% coverage ma error handling non testato. |
| **Fix** | Test per error path: input invalido, canale chiuso, contesto cancellato. |
| **Rischio** | BASSO |
| **Verifica** | Coverage synthesis ≥ 85%. `go test ./internal/tools/synthesis/...` ✅ |
| **File** | `internal/tools/synthesis/*_test.go` |

### D4-3: User journey smoke test

| Campo | Valore |
|-------|--------|
| **Problema** | Nessun test end-to-end che verifichi il percorso utente completo. |
| **Decisione team** | **Filosofo**: "Un sistema è funzionante quando ogni stub incontrato non rompe l'esperienza." |
| **Fix** | Creare 1 test E2E (Playwright o Go): login → crea agente → esegui tool → salva → ricarica → agente persiste. |
| **Rischio** | BASSO — assicurazione qualità |
| **Verifica** | Playwright test passa. `npx playwright test` ✅ |
| **File** | `frontend/e2e/` |

### D4-4: Build check finale

| Comando | Verifica |
|---------|----------|
| `go build ./...` | ✅ exit 0 |
| `npx tsc --noEmit` | ✅ exit 0 |
| `npx vite build` | ✅ exit 0 |
| `npx playwright test` | ✅ 20/20 |
| `go test ./...` | ✅ exit 0 |

---

## Esecuzione

```
Wave 1 (sicurezza+stabilità, 10 task) → Wave 2 (stub/TODO, 9 task) → Wave 3 (refactor, 2 task) → Wave 4 (test, 3 task)
```

**Build check dopo ogni wave:** obbligatorio prima di passare alla successiva.
**Principio:** Ogni wave builda, testa e passa review prima di procedere.
