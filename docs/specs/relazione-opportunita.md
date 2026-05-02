# Relazione Opportunità/Idee — Aleph-v2

> **Prodotto da**: Sisyphus (analisi sistematica 5 esploratori + ricerche librarian)
> **Data**: 1 Maggio 2026
> **Stato build**: `go build` ✅ | `go test -race` ✅ | `npx tsc` ✅ | `npx vite build` ✅
> **Completamento globale**: ~72%

---

## Indice

1. [Premessa: Filosofia degli Interventi](#1-premessa-filosofia-degli-interventi)
2. [Opportunità Architetturali (Backend)](#2-opportunità-architetturali-backend)
3. [Opportunità Frontend](#3-opportunità-frontend)
4. [Opportunità Sicurezza & Infrastruttura](#4-opportunità-sicurezza--infrastruttura)
5. [Opportunità CI/CD & Testing](#5-opportunità-cicd--testing)
6. [Opportunità NLP Sidecar](#6-opportunità-nlp-sidecar)
7. [Opportunità UX/Prodotto](#7-opportunità-uxprodotto)
8. [Opportunità ML/AI](#8-opportunità-mlai)
9. [Priorità Strategica](#9-priorità-strategica)

---

## 1. Premessa: Filosofia degli Interventi

Aleph-v2 è un sistema **funzionante** ma non **robusto**. La differenza è cruciale:
- Funzionante = build passa, test passano, feature ci sono
- Robusto = edge case gestiti, failure mode previsti, scaling lineare, sicurezza difensiva

Tutte le opportunità elencate seguono questi principi:
- **Difesa in profondità**: ogni layer deve gestire il fallimento del layer sottostante
- **Fail closed**: in caso di dubbio, nega/blocca (non permetti)
- **Zero trust**: non fidarti di input, ambiente, utente, o subsystema vicino
- **Observability first**: se non puoi misurarlo, non puoi fixarlo
- **Minimal viable security**: non over-engineering, ma non buchi

---

## 2. Opportunità Architetturali (Backend)

### 2.1 Reflection Engine Unificato
**Stato attuale**: `Engine.Reflect` (semplice, usato in prod) vs `DefaultReflector` (completo, morto). L'implementazione completa con classificazione GapType (strategy_gap, execution_gap, knowledge_gap) esiste ma non è cablata.

**Opportunità**: Unificare su DefaultReflector. Aggiungere feedback loop automatico:
- Strategy Gap → rivedere piano
- Execution Gap → retry con parametri diversi
- Knowledge Gap → query ontology/knowledge base

**Impatto**: Trasforma il decision loop da passeggero a attivo. Da ~50 righe di stub a sistema reale.

### 2.2 Plan-Act Connector
**Stato attuale**: Plan genera step strutturati, Act chiama LLM raw — disconnected. Act non usa mai i piani generati da Plan.

**Opportunità**: Introdurre un TaskExecutor che:
1. Prende gli step da Plan
2. Li traduce in tool call specifici
3. Tracking execution per step
4. Se fallisce, mappa errore allo step e torna a Reflect

**Architettura**: `Plan → TaskQueue → TaskExecutor (per step) → risultato → Observe → Reflect`

### 2.3 Rate Limiter Memless
**Stato attuale**: Mappa in memoria senza cleanup — crescita illimitata sotto DDoS.

**Opportunità**: Implementare sliding window con Redis O(1) oppure:
- Periodica cleanup degli IP scaduti (scan every 10 min)
- Fixed-size LRU cache per IP attivi
- Rate limit per-user oltre che per-IP

### 2.4 DuckDB Concurrency Model Review
**Stato attuale**: RWMutex + semaphore.Weighted(5). QueryRowContext ritorna nil su semaphore exhaustion. Deadlock potenziale TX che tiene Lock mentre chiama RLock internamente.

**Opportunità**:
- QueryRowContextOrError dovrebbe essere il default (non QueryRowContext con nil panic)
- Separare read pool e write pool (due semafori)
- Timeout su acquisizione semaforo (context deadline)
- Pool di connessioni DuckDB separato (non :memory: condivisa)

### 2.5 Circuit Breaker Pattern Completo
**Stato attuale**: MCP health check ha thundering herd problem in half-open state — tutti i goroutine riprovano contemporaneamente.

**Opportunità**: Aggiungere:
- Randomized retry delay (jitter)
- Numero massimo di richieste in half-open (es. 1 su 10 passa)
- Exponential backoff con cap
- Circuit stato per-endpoint (non globale)

### 2.6 DuckDB VSS First-Class
**Stato attuale**: VSS extension gracefully skipped se non installata — fallback a sequential scan.

**Opportunità**:
- `goose up` migration per installare VSS extension su DuckDB
- Test con VSS attivo (ora skippato)
- Embedding dimension validation (768) all'avvio
- Index rebuild periodico

### 2.7 Structured Error Enrichment
**Stato attuale**: Error handler ha 3-tier wrapping ma errori interni spesso non hanno codice/sottosistema.

**Opportunità**: Aggiungere a ogni errore:
- `Subsystem` (storage/mcp/ingestion/tool/decision)
- `Operation` (query/insert/discover/execute)
- `Recoverable` flag
- `RetryAfter` suggerito

Questo permetterebbe al DiagnosticMonitor di correlare automaticamente, e al frontend di mostrare messaggi utente migliori.

### 2.8 HealthChecker Context Fix
**Stato attuale**: `Start()` overwrites cancel da `NewHealthChecker` — context leak. MCP discovery health loop senza WaitGroup.

**Opportunità**:
- Non overwriteare cancel; creare child context
- Aggiungere WaitGroup a tutti i goroutine lifecycle
- Graceful shutdown ordinato: Stop() segnala → Wait() conferma arresto

---

## 3. Opportunità Frontend

### 3.1 Type Safety Zero-Tolerance
**Stato attuale**: 14 file con `any` in produzione, 42 `as unknown as` cast. `assertType()` identity function bypassa Zod.

**Opportunità:**
- Lighthouse audit: non si può aggiungere `any` senza review
- `assertType` va rimpiazzata con `z.infer<typeof schema>` in ogni consumer
- `as unknown as` tolleranza zero — ogni cast deve avere un commento che spiega perché è sicuro
- 42 casts → 42 ticket → 0 casts rimanenti

### 3.2 AbortController Pattern
**Stato attuale**: Nemmeno un AbortController in tutta la codebase. Fetch di dati in App.tsx, chat history loading, tutti vulnerabili a race condition su view switch.

**Opportunità**: Hook custom `useFetch` o pattern standard:
```typescript
function useFetch<T>(url: string, deps: any[]) {
  useEffect(() => {
    const ac = new AbortController();
    fetch(url, { signal: ac.signal }).then(setData);
    return () => ac.abort();
  }, deps);
}
```

**Impatto**: Elimina tutte le race condition su navigazione rapida.

### 3.3 ConnectRPC Come Unico Transport
**Stato attuale**: ToolForm, SkillForm, DataSourceForm usano fetch() raw REST invece di ConnectRPC client. useOntologyActions usa fetch() senza nemmeno auth headers.

**Opportunità**: Unificare su ConnectRPC:
- Creare client ConnectRPC con auth interceptor automatico
- Deprecare tutti i fetch() raw
- Aggiungere tipo protobuf per ogni operazione CRUD

### 3.4 Error Handling Centralizzato
**Stato attuale**: handleError duplicato (module-level + hook) — doppio toast. Empty catch blocks in 3 posti. Errore non tipizzato.

**Opportunità**:
- Singleton `errorService` con subscriber pattern
- Categorie errore (network, auth, validation, server)
- Toasts automatici + log strutturato
- Empty catch blocks: mai `catch {}` — sempre `catch (e) { errorService.handle(e) }`

### 3.5 SSE Reconnection Logic
**Stato attuale**: SSE reconnect immediato su 401 — flooding il server. Mutable state module-level (`lastEventIdInternal`).

**Opportunità**:
- Exponential backoff su reconnect (500ms → 1s → 2s → 4s → max 30s)
- Reset backoff su connessione riuscita
- `lastEventId` in Zustand store (non module-level)
- Stop reconnect su 401/403 fino a refresh token

### 3.6 Frontend Cache & Pagination
**Stato attuale**: useCursorPagination ha stale closure. fetchTools non passa projectId.

**Opportunità**:
- useCallback/cache aggiornato correttamente
- React Query o SWR per caching e dedup
- Paginazione cursor-based ottimistica
- Prefetch view su hover sidebar

### 3.7 Bundle Splitting Perfetto
**Stato attuale**: codice splitting già presente (React.lazy + Vite chunks). Factory.ts ha dual import (40969e3a + 49abb05e in stesso chunk). Vendor chunk 295KB > target 150KB.

**Opportunità**:
- Splittare vendor chunk in vendor (React/Zustand) + ui (lucide-react + components)
- Factory chunk: unificare il dual import
- Chunk map viewer (d3) lazy loading
- Analisi bundle automatica in CI

### 3.8 CSS per Produzione
**Stato attuale**: Dynamic Tailwind classi non purge-safe. CSS volatility layers introdotti in W4 ma non coprono tutto.

**Opportunità**: Audit di tutte le classi dinamiche:
```typescript
// ❌ Non purge-safe
<div className={`text-${color}-500`}>
// ✅ Purge-safe  
<div className={colorVariants[color]}>
```

Tutte le varianti dinamiche devono usare lookup table.

---

## 4. Opportunità Sicurezza & Infrastruttura

### 4.1 Session Token Pattern
**Stato attuale**: API key in HttpOnly cookie plaintext — nessuna sessione, nessun refresh, nessun revoke.

**Opportunità**: Implementare session token pattern:
```
POST /api/v1/auth/session → create session → return session_id
API key non esce mai dal server
Session scade, si refresh, si revoca
Server mantiene mappa session_id → api_key con TTL
```
Vantaggio: revoca, rotazione, audit logging.

### 4.2 Sandbox Isolamento Reale
**Stato attuale**: Sandbox bloccato solo da allowlist — niente isolamento OS, niente limiti risorse, niente Docker/gVisor.

**Opportunità**: Tre livelli di sandbox:
1. **L1 (base)**: Blocchi allowlist + timeouts — per tool trusted interni
2. **L2 (standard)**: Docker container con resource limits — per tool non fidati
3. **L3 (massimo)**: gVisor o Firecracker VM — per esecuzione codice arbitrario

Raccomandazione: iniziare con L2 via Docker SDK.

### 4.3 SQL Injection Difesa Definitiva
**Stato attuale**: query.go ha 3 fmt.Sprintf con lone. DSL filter pure. 9 siti in memory/store.go.

**Opportunità**: Creare uno `statementBuilder` che:
1. Accetta solo parametri tipizzati (string, int, float, []string)
2. Usa sempre $1, $2, ... o DuckDB prepared statements
3. Vieta fmt.Sprintf per query SQL a livello CI (linter)

### 4.4 CSRF -> SameSite=Lax Default
**Stato attuale**: CSRF permette richieste senza Origin/Referer header — bypassabile.

**Opportunità**: 
- Cambiare default a `SameSite=Lax`
- Origin/Referer validation obbligatoria
- Bloccare richieste senza Origin che non sono GET/HEAD
- Aggiungere test specifici per Origin assente

### 4.5 CSP Hardening
**Stato attuale**: CSP include `ws://localhost:*` — bypassabile via WebSocket a localhost.

**Opportunità**:
- Sostituire con URL specifici (`ws://localhost:8080` se necessario)
- `'strict-dynamic'` per loading JS gerarchico
- `base-uri 'self'`
- `form-action 'self'`

### 4.6 Rate Limiting Anti-DDoS
**Stato attuale**: Mappa in memoria senza cleanup.

**Opportunità**:
- Implementare sliding window
- Per IP + per API key
- Header `X-RateLimit-Remaining` in risposta
- Redis-backed per multi-instanza
- Cleanup periodico (eviction)

### 4.7 Docker Secrets
**Stato attuale**: `KEY_ENCRYPTION_KEY` e `POSTGRES_PASSWORD` in env vars nel compose visibili via `docker inspect`.

**Opportunità**: Usare Docker Secrets o env_file con permessi 600. Al minimo, non esporre in docker-compose.yml per `docker compose config`.

---

## 5. Opportunità CI/CD & Testing

### 5.1 PAORA Decision Engine Test Suite
**Stato attuale**: Zero test su Plan, Act, Observe, Reflect, Admit. ChatSession 330 linee non testate.

**Opportunità**:
- Mock LLM Provider (return fixed plan)
- Unit test per ogni fase individualmente
- Integration test per ciclo PAORA completo
- Property-based test: "per qualsiasi piano valido, Reflect produce feedback"
- Test GNN predictor (mock training)

### 5.2 Pipeline CI a Prova di Fallimento
**Stato attuale**: `go test | tee` senza pipefail — test failure non ferma la build. Deploy parte senza test gate.

**Opportunità**:
- `set -o pipefail` in CI scripts
- Deploy step con `needs: [test, build]` esplicito
- Aggiungere `go vet` in CI
- Cache go modules layer
- Test matrix almeno per Go 1.24

### 5.3 Contract Test Connessi
**Stato attuale**: `-tags=contract` ma nessun `//go:build contract` nel sorgente.

**Opportunità**:
- Cablare build tags nei file handler_test.go
- Eseguire contract test in CI separatamente
- Verificare API response shape (ConnectRPC)
- Aggiungere OpenAPI spec validation

### 5.4 E2E Testing con Playwright
**Stato attuale**: Zero E2E test. Solo unit visivi.

**Opportunità**:
- Setup Playwright + docker-compose per integration
- Test flusso login → dashboard → tool creation
- Test SSE event delivery in UI
- Test error boundary rendering
- Visual regression su componenti core

### 5.5 Fuzzing & Property-Based Test
**Stato attuale**: Fuzzing assente.

**Opportunità**:
- Go fuzzing su query.go (SQL injection detection)
- DuckDB query fuzzing
- Sandbox input fuzzing (allowlist bypass detection)
- Frontend: vitest + faker per test a priori

### 5.6 Vulnerability Scanning Continuo
**Stato attuale**: gitleaks in CI, ma nessuno scan dipendenze.

**Opportunità**:
- `govulncheck` in CI
- `npm audit` in CI (con break su critical)
- Docker image scanning (trivy o snyk)
- Dependabot/Renovate per auto-update

---

## 6. Opportunità NLP Sidecar

### 6.1 Watchdog Riavvia su Panico
**Stato attuale**: `watchSidecar` in goroutine senza defer recover — panico silenzioso uccide il loop.

**Opportunità**:
- Defer recover in watchSidecar con restart
- Max restart count (es. 3) prima di arrendersi
- Logging su panic

### 6.2 gRPC Health Check Frequency
**Stato attuale**: Health check ogni 10s — OK per produzione ma lento per failure detection.

**Opportunità**: 
- Health check ogni 2s (leggero)
- 3 fallimenti consecutivi → segnala unhealthy
- Immediate retry dopo fix

### 6.3 NLP Model Caching
**Stato attuale**: Ogni chiamata NLP carica/elabora il modello? (da verificare).

**Opportunità**:
- Warm up models on startup
- Cache risultati per input identici (TTL)
- Rate limit per utente per chiamate NLP

---

## 7. Opportunità UX/Prodotto

### 7.1 Gestione Errore Human-Readable
**Stato attuale**: Errori mostrati con codici interni. Utente vede "ERR_INTERNAL".

**Opportunità**:
- Mappare ogni APIError.code a messaggio utente in italiano
- Error context: "Connessione al database fallita. Riprova tra qualche secondo."
- Azione suggerita: "Contatta l'amministratore" per errori server

### 7.2 Onboarding Zero-Config
**Stato attuale**: SetupWizard esiste ma incompleto.

**Opportunità**:
- First-run wizard: configura provider LLM, crea primo agente
- Demo data: 2-3 agenti preconfigurati, 1 datasource fittizio
- Tooltip tour guidato

### 7.3 Stato Sistema in Tempo Reale
**Stato attuale**: Stato sistema accessibile via `/api/v1/healthz` ma non in UI.

**Opportunità**:
- Health panel in StatusBar con indicatori verde/giallo/rosso
- Ultimo errore con timestamp
- Tool health in UI (MCP discovery status)
- Backend: health endpoint con deep check

### 7.4 Accessibility (a11y) Audit
**Stato attuale**: Focus visibile? Tastiera navigabile? Screen reader?

**Opportunità**: Audit base:
- Tab order su tutti i form
- Focus trap su modal/slideover
- ARIA labels su icone decorative
- Color contrast su testi piccoli (13px JetBrains Mono)
- Keyboard shortcut per comandi principali

### 7.5 Performance Perception
**Stato attuale**: Chiamate sincrone senza feedback.

**Opportunità**:
- Skeleton loader per ogni view (non "loading..." testuale)
- Ottimistic UI per create/update tool
- Prefetch dati view adiacenti su sidebar hover
- Debounce ricerca 300ms

---

## 8. Opportunità ML/AI

### 8.1 GNN Predictor Training Pipeline
**Stato attuale**: GNN LinkPredictor creato con 100 nodi, 64 dimensioni, learning rate 0.01 — MAI addestrato. IsTrained sempre false.

**Opportunità**:
- Addestramento offline su dati storici (tool usage, agent relationships)
- Online fine-tuning con feedback dell'utente
- Threshold predictions solo se confidence > 0.7
- Metrics: precision@k, recall@k

### 8.2 LLM Provider Fallback Chain
**Stato attuale**: Provider singolo. Se ollama muore, tutto fermo.

**Opportunità**:
- Fallback chain: ollama → anthropic → openai
- O punteggio weighted basato su disponibilità
- Health check per provider
- Configurazione in UI con drag-and-drop priority

### 8.3 Embedding Caching
**Stato attuale**: Ogni embed è una chiamata LLM.

**Opportunità**:
- LRU cache per embedding con TTL
- Content-hash come chiave
- Cache persistente su DuckDB
- Batch embedding requests

### 8.4 Tool Suggestion ML
**Stato attuale**: Tool suggestion fixed pattern-based.

**Opportunità**:
- Embedding similarity tra messaggio utente e descrizione tool
- Usage-based ranking (tools più usati pesano di più)
- Context-aware (quali tools già in sessione)
- Feedback loop: suggest → usato/non usato → impara

---

## 9. Priorità Strategica

### Quadrant Impacto/Effort

| Area | Impatto | Effort | Priorità |
|------|---------|--------|----------|
| Sandbox isolamento L2 (Docker) | 🔴 CRITICO | 3gg | **W0** |
| Session Token Pattern | 🔴 CRITICO | 2gg | **W0** |
| SQL injection fix all sites | 🔴 CRITICO | 1gg | **W0** |
| AbortController pattern frontend | 🔴 CRITICO | 0.5gg | **W0** |
| Reflection Engine unificato | 🟡 ALTO | 2gg | **W1** |
| Plan-Act Connector | 🟡 ALTO | 2gg | **W1** |
| Error centralizzato frontend | 🟡 ALTO | 1gg | **W1** |
| PAORA test suite | 🟡 ALTO | 3gg | **W1** |
| CI pipefail fix + test gate | 🟡 ALTO | 0.5gg | **W1** |
| Type safety audit (any -> never) | 🟡 ALTO | 4gg | **W2** |
| Rate limiter sliding window | 🟡 ALTO | 1gg | **W2** |
| DuckDB concurrency fix | 🟡 ALTO | 1gg | **W2** |
| ConnectRPC unico transport | 🟡 ALTO | 3gg | **W2** |
| GNN training pipeline | 🟢 MEDIO | 5gg | **W3** |
| E2E testing (Playwright) | 🟢 MEDIO | 3gg | **W3** |
| NLP watchdog restart | 🟢 MEDIO | 0.5gg | **W3** |
| CSP hardening | 🟢 MEDIO | 1gg | **W3** |
| CSS purge-safe audit | 🟢 MEDIO | 1gg | **W3** |
| Bundle splitting fine | 🟢 MEDIO | 1gg | **W3** |
| LLM fallback chain | 🔵 BASSO | 2gg | **W4** |
| Embedding caching | 🔵 BASSO | 1gg | **W4** |
| Tool suggestion ML | 🔵 BASSO | 3gg | **W4** |
| a11y audit | 🔵 BASSO | 2gg | **W4** |
| Fuzzing tests | 🔵 BASSO | 2gg | **W4** |
| Vulnerability scanning CI | 🔵 BASSO | 1gg | **W4** |
| Docker secrets | 🔵 BASSO | 0.5gg | **W4** |

### Roadmap Raccomandata (5 Wave)

**W0 — Critical Safety Net (impatto ~8gg)**
- Sandbox isolamento L2 (Docker)
- Session Token Pattern
- SQL injection fix (tutti i 12+ siti)
- AbortController pattern frontend
- CI pipefail fix + test gate

**W1 — Stability & Reliability (impatto ~9gg)**
- Reflection Engine unificato + Plan-Act Connector
- PAORA test suite
- Error centralizzato frontend
- Rate limiter + DuckDB concurrency fix

**W2 — Code Quality Hardening (impatto ~8gg)**
- Type safety audit (any → zero)
- ConnectRPC unico transport
- Empty catch block elimination
- SSE reconnection fix
- CSRF SameSite hardening

**W3 — Testing & Infrastructure (impatto ~9gg)**
- E2E testing (Playwright)
- GNN training pipeline
- NLP watchdog + health check
- CSP hardening
- CSS purge-safe audit
- Bundle splitting fine

**W4 — Polish & ML (impatto ~10gg)**
- LLM fallback chain
- Embedding caching
- Tool suggestion ML
- a11y audit
- Fuzzing tests + vulnerability scanning
- Docker secrets
- Performance perception (skeleton, optimistic UI)

### Stima Totale: ~44gg uomo

---

## Appendice: Metriche Chiave

| Metrica | Stato | Target |
|---------|-------|--------|
| `any` in produzione | 14+ file | 0 |
| `as unknown as` | 42 | 0 |
| Fetch senza auth headers | 1 (useOntologyActions) | 0 |
| AbortController usati | 0 | Tutti i fetch |
| Empty catch blocks | 3 | 0 |
| Sandbox isolation layers | 0 (solo allowlist) | 2 (Docker + gVisor) |
| Session token | No | Yes |
| PAORA test coverage | 0% | >80% per fase |
| E2E test count | 0 | >20 scenari |
| Rate limiter cleanup | Nessuna | Periodica/Redis |
| GNN trained | Mai | Sì, con metriche |
| CSP score | D+ | A |
| CI/CD security scans | 1 (gitleaks) | 4+ (govulncheck, npm audit, trivy, gitleaks) |
