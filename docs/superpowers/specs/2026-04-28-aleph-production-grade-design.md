# Aleph-v2 — Production-Grade Design

**Data**: 2026-04-28
**Stato**: Design approvato, in attesa di implementation plan
**Reviewer**: Aleph (self-review via Chat API)

---

## Panoramica

Portare Aleph-v2 da prototipo funzionante a sistema production-grade (100%). Cinque fasi in ordine di priorità, non cronologico. Ogni fase produce un sistema deployabile e testabile prima di passare alla successiva.

Stack: Go 1.22+ backend (ConnectRPC), React 18.3 + Typescript + Tailwind frontend, DuckDB + PostgreSQL, Docker.

---

## FASE 0 — Hotfix Bloccanti (5 task)

Prima di qualsiasi modifica architetturale, fixare i bug che rompono funzionalità esistenti.

### 0.1 Fix responseWriter/Flusher (CRITICO — streaming LLM rotto)

- **Problema**: `responseWriter` in middleware/recovery o nel wrapping HTTP non implementa `http.Flusher`. Il server ConnectRPC fa streaming di token (Chat API), ma senza Flusher il client non riceve nulla fino alla fine della risposta.
- **Soluzione**: Aggiungere `Flush()` al wrapper `responseWriter` o usare `http.Flusher` interface check + flush su ogni write. Fix già applicato parzialmente durante la sessione.
- **File**: `internal/middleware/recovery.go` o equivalente.
- **Verifica**: Stream di Chat API deve tornare token in tempo reale (già testato — funziona).

### 0.2 Fix skill_ids NULL (migrazione + backfill)

- **Problema**: Dati esistenti con `skill_ids = NULL` causano errore `sql.Scan` perché Go non può fare scan di NULL in `string`. Già fixato con `sql.NullString`, ma i record vecchi restano NULL.
- **Soluzione**:
  - Migrazione PostgreSQL: `UPDATE system_agents SET skill_ids = '[]' WHERE skill_ids IS NULL;`
  - Aggiungere NOT NULL constraint se sicuri, altrimenti lasciare soft.
  - Nuove insert devono forzare `skill_ids = '[]'` nel Go code.

### 0.3 Fix sentiment sempre 0.0 (bug NLP sidecar)

- **Problema**: `AnalyzeSentiment` restituisce sempre score=0.0. Il NLP sidecar non è running su :8001. L'handler degrada a mock.
- **Soluzione**: Aggiungere NLP sidecar al Docker compose. Per ora, se sidecar non disponibile, restituire errore esplicito invece di 0.0 fittizio. Feature reale in FASE 3.
- **Impatto**: Decision Engine usa sentiment. Con 0.0, tutte le decisioni sono neutre — inutile.

### 0.4 Docker compose base

- **File**: `docker-compose.yml` nella root del progetto.
- **Servizi**:
  - `postgres` (immagine postgres:16-alpine, 5432, volume persistente, env POSTGRES_PASSWORD via .env)
  - `aleph` (build dal codice Go, :8080, dipende da postgres)
  - `nlp-sidecar` (opzionale, stub, per test sentimento reale) — se non disponibile, skip.
- **Goal**: `docker compose up` deve far partire tutto, con backend funzionante su :8080.
- **Nota**: `FRONTEND` non in Docker — si serve tramite `embed.FS` nel Go binary.

### 0.5 Test hotfixes

- Unit test per Flusher wrapper
- Integration test per skill_ids scan (NULL + valid)
- Health check: start Docker compose, curl /api/v1/healthz → OK
- Chat streaming test: inviare messaggio, ricevere token in streaming

---

## FASE 1 — Production Hardening (8 task)

Sicurezza, osservabilità, affidabilità — prerequisito per qualsiasi altra feature.

### 1.1 Rate limiting

- **Globale**: 500 richieste/min per IP (o per API key se autenticato)
- **Differenziato**:
  - `/chat` (LLM-heavy) → 10 richieste/min per utente
  - `/health`, `/readyz`, `/livez` → 100 richieste/min
  - Tutti gli altri → 500 richieste/min
- **Implementazione**: Middleware Go con mappa `sync.Map` + sliding window (o token bucket con `golang.org/x/time/rate`).
- **Headers di risposta**: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- **Configurabile** via `config.RateLimit` con default.

### 1.2 CSP headers + Security Headers

- **CSP**: `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self' ws://localhost:*;`
  - Per frontend SPA embed: script/style inline controllati solo dal build.
- **Altri headers**: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Strict-Transport-Security` (se TLS), `Referrer-Policy: same-origin`.
- **File**: Middleware globale in `internal/middleware/security.go`.

### 1.3 Graceful shutdown

- **Problema attuale**: `AlephApp.Close()` esiste ma non è chiamata su SIGTERM/SIGINT. Il server si ferma e basta.
- **Soluzione**: Aggiungere signal handler in `main.go`:
  1. Catch SIGTERM/SIGINT
  2. Set health endpoint a "not ready" (return 503 da /readyz)
  3. Stop accepting new requests (server shutdown)
  4. Drain connessioni attive (context deadline esplicito, es. 30s)
  5. Chiamare `a.Close(ctx)` nell'ordine corretto
- **Verifica**: `kill <pid>` → log ordinato di shutdown, zero richieste perse.

### 1.4 Prometheus metrics + /readyz /livez

- **Endpoint**:
  - `GET /readyz` → 200 se pronto, 503 se draining
  - `GET /livez` → 200 se vivo (sempre, anche senza DB — health check leggero)
  - `GET /metrics` → formato Prometheus
- **Metriche**:
  - `aleph_requests_total{method,path,status}` — counter
  - `aleph_request_duration_seconds{method,path}` — histogram
  - `aleph_streaming_tokens_total` — counter
  - `aleph_db_connections_active` — gauge
  - `aleph_llm_duration_seconds` — histogram
  - `aleph_memory_store_operations_total` — counter (quando MemStore sarà attivo)
  - `aleph_subsystem_health{name}` — gauge (1=healthy, 0=degraded)
- **Implementazione**: Usare `prometheus/client_golang` con middleware automatico.

### 1.5 Structured logging con correlation ID

- **Problema**: log in formato misto (`log.Printf` misto a `slog`).
- **Soluzione**:
  - Tutto a `slog.JSONHandler` (già impostato, mancano solo alcune conversioni)
  - Aggiungere middleware che estrae/inietta `X-Request-ID` (o genera UUID se assente)
  - Ogni log nel ciclo di vita di una richiesta include `request_id`
  - Logging strutturato per errori: `slog.Error("db query failed", "err", err, "query", query, "request_id", rid)`
- **File**: `internal/middleware/requestid.go`, modifiche sparse.

### 1.6 Secret management

- **Problema attuale**: API key salvate in PostgreSQL, encryption key passata via config ma non validata.
- **Soluzione**:
  - `ALEPH_ENCRYPTION_KEY` come env var obbligatoria
  - Se mancante → log "FATAL: ALEPH_ENCRYPTION_KEY required" e exit con codice 1
  - Lunghezza minima 32 byte (AES-256)
  - Nessun fallback a default
  - Documentare in `.env.example`
- **File**: `internal/config/config.go`, `internal/app/app.go`

### 1.7 CORS configurabile

- **Problema attuale**: CORS hardcoded nel middleware (se presente).
- **Soluzione**:
  - `CORSAllowedOrigins` in config (default `["http://localhost:5173"]` per sviluppo)
  - In produzione, forzare a dominio specifico
  - Non usare mai `Access-Control-Allow-Origin: *` con credenziali

### 1.8 Test — ogni item di FASE 1

- Rate limiting: test con +10 richieste consecutive, verificare 429.
- Security headers: GET su / e verificare headers.
- Graceful shutdown: kill server, verificare log ordinato.
- Prometheus: GET /metrics, verificare formato.
- /readyz: 200 a boot, 503 dopo signal.
- Logging: X-Request-ID propagato.

---

## FASE 2 — Backend Wiring (7 task)

Cablaggio dei tre subsystem esistenti ma scollegati.

### 2.1 Wire MemStore (memory subsystem)

- **Stato attuale**: `memStore, _ := memory.NewMemoryStore(...)` + `_ = memStore` (linee 207-212 di app.go). Creato ma non usato.
- **Azioni**:
  1. Definire interfaccia `memory.Store` nel package memory
  2. `MemStore` implementa l'interfaccia (embedding storage + retrieval)
  3. Collegare a `QueryHandler` per salvare/recuperare vettori durante le query
  4. Collegare a `DecisionEngine` per trust score basato su memoria
- **Non serve persistenza al volo** — MemStore usa DuckDB già passato nel costruttore.

### 2.2 Wire GNN client (epistemic trust)

- **Stato attuale**: Package `internal/gnn/` esiste ma non è importato in `app.go`. DecisionEngine usa trust score via registry (AvgBrierScore) ma GNN potrebbe migliorare la stima.
- **Azioni**:
  1. Verificare interfaccia GNN esistente
  2. Creare wrapper/adapter se necessario
  3. Integrare in `DecisionEngine` come optional estimator (fallback al registry se GNN non disponibile)
  4. Se GNN non addestrato, skip silenzioso con log Warn
- **Approccio difensivo**: GNN è un enhancement, non un bloccante. Fallisce silenziosamente.

### 2.3 Wire Workflow Engine (base)

- **Stato attuale**: `internal/workflow/` esiste ma contiene solo `.gitkeep`.
- **Definizione**: Workflow Engine gestisce esecuzione di task multi-step (es. "analizza progetto, genera report, invia notifica"). Non è un orchestrator generico — è specifico per toolchain Aleph.
- **Interfaccia**:
  ```go
  type WorkflowEngine interface {
      RegisterStep(name string, fn StepFunc)
      Execute(ctx context.Context, w *Workflow) error
      GetStatus(workflowID string) (WorkflowStatus, error)
  }
  ```
- **Implementazione iniziale**: DuckDB-backed per persistenza stato. Valutazione in item 2.5.
- **Limiti**: Singolo nodo, no scheduling complesso. Multi-agent orchestration in FASE 3.

### 2.4 Circuit breaker per subsystem

- **Problema**: NLP sidecar down, GNN non risponde, MemStore crash → uno di questi non deve abbattere l'intero server.
- **Soluzione**: Circuit breaker pattern standard:
  - Stato: CLOSED (funziona) → OPEN (N fallimenti, skip) → HALF_OPEN (riprova dopo N secondi)
  - Soglia: 5 fallimenti consecutivi, timeout 30 secondi
  - Ogni subsystem ha il suo circuit breaker
- **Interfaccia**:
  ```go
  type CircuitBreaker struct { /* ... */ }
  func (c *CircuitBreaker) Execute(fn func() error) error // torna ErrCircuitOpen se aperto
  ```
- **File**: `internal/middleware/circuitbreaker.go` — da riutilizzare per ogni subsystem.

### 2.5 DuckDB evaluation per workflow

- **Valutazione**: DuckDB è OLAP, ma per workflow con ~10-100 write/secondo e stato semplice, va bene. Non abbiamo bisogno di transazioni complesse.
- **Decisione**: Iniziare con DuckDB. Se le performance diventano un problema (1000+ workflow attivi), migrare a PostgreSQL.
- **Metrica di monitoraggio**: `aleph_workflow_write_duration_seconds` (Prometheus item 1.4). Se p95 > 100ms, migrare.
- **Nessuna migrazione automatica**: Solo manuale, documentata.

### 2.6 Timeout definitivi

- **HTTP client generico**: 30 secondi (max)
- **DB queries**: 10 secondi
- **LLM calls**: 5 minuti (modelli grandi, streaming)
- **NLP sidecar**: 30 secondi
- **Workflow execution**: 15 minuti per workflow singolo
- **Configurabili** via `config.Timeout` struct

### 2.7 Integration tests per wiring

Per ogni subsystem cablato:
1. Test di creazione (MemStore, WorkflowEngine, GNN client)
2. Test di fallimento controllato (circuit breaker si apre)
3. Test di integrazione end-to-end per workflow semplice
4. Mockare PostgreSQL (DuckDB per test è sufficiente)

---

## FASE 3 — Advanced Backend (6 task)

Feature avanzate, dopo che security e wiring sono stabili.

### 3.1 Multi-agent orchestration

- **Pattern**: Un agent orchestrator (workflow step) che:
  1. Riceve un task complesso
  2. Lo decompone in sotto-task
  3. Assegna sotto-task a agents specializzati (es. "code-review", "research", "summarize")
  4. Raccoglie risultati e li assembla
- **Limite massimo**: 3 agents attivi contemporaneamente (configurabile). Con rate limiting già attivo, questo è safe.
- **DuckDB-backed**: Workflow state salvato in DuckDB (dalla FASE 2).

### 3.2 Export PDF/CSV/JSON

- **Endpoint**:
  - `GET /api/v1/export/{type}` dove type = pdf | csv | json
  - Prende parametri: `project_id`, `query`, `format`
- **CSV/JSON**: Streaming response (non caricare tutto in memoria)
- **PDF**: Libreria Go (es. go-pdf o wkhtmltopdf wrapper). MVP: HTML→PDF via headless browser o template PDF base.
- **Non**: Documenti complessi. Report semplici con dati strutturati.

### 3.3 NLP sidecar reale (fix sentimento + Docker compose)

- **Fix bug sentimento**: Il modello NLP non veniva caricato. Nel Docker compose, aggiungere servizio NLP (fastapi + modello transformers) sulla porta 8001.
- **Docker compose**: `nlp` service nell'esistente compose (creato in FASE 0.4).
- **Health check**: Aleph deve verificare che NLP sia vivo su /health prima di usarlo.

### 3.4 DSL compiler caching (deferred)

- **Stato**: Sospeso fino a quando Prometheus metrics mostrano che DSL è un percorso caldo.
- **Metrica**: `aleph_dsl_compilation_duration_seconds`. Se p95 > 500ms e chiamato > 100 volte/ora, implementare cache LRU.
- **Nessuna implementazione ora** — solo wiring del misuratore.

### 3.5 API pagination

- **Pattern**: `?page=1&per_page=50` su tutti gli endpoint che restituiscono liste (projects, agents, skills, tools, library).
- **Headers di risposta**: `X-Total-Count`, `X-Total-Pages`, `Link` (rel=next/prev).
- **Default**: `per_page=50`, `page=1`.
- **Backend**: Tutti i query handler devono supportare limit/offset.

### 3.6 Tests

- Multi-agent: test di orchestration con agenti mock
- Export: test di formato per CSV/JSON
- Pagination: test limit/offset su ogni endpoint lista
- NLP: test di fallback quando sidecar non disponibile

---

## FASE 4 — UI Redesign (8 task)

Ultima fase. Solo dopo che security, wiring e advanced backend sono stabili.

Priorità interna: 3 critici PRIMA, poi il resto.

### 4.1 Scroll continuo CopilotView (CRITICO)

- **Problema**: La vista chat interrompe lo scroll a ogni nuovo messaggio, UX frustrante.
- **Soluzione**: Lazy loading infinito — quando l'utente scrolla in alto, carica messaggi più vecchi. Scroll ancorato in basso per nuovi messaggi. Nessun "carica altri" button.
- **Componente**: `CopilotView.tsx` — modificare per scroll infinito con `IntersectionObserver`.

### 4.2 ErrorBoundary per view (CRITICO)

- **Problema**: Un errore in una view fa cadere tutta l'app.
- **Soluzione**: `AlephErrorBoundary` componente che wrappa ogni view lazy-loaded. Mostra errore specifico + pulsante "riprova". Non propaga l'errore alla root.
- **Già iniziato**: App.tsx ha React.lazy per 6 view. Aggiungere ErrorBoundary.

### 4.3 Tema scuro default (CRITICO)

- **Problema**: Tema chiaro fa flashbang all'avvio.
- **Soluzione**: 
  - Sfondo: `#0d0d0d`
  - Testo: `#e0e0e0`
  - Accenti: `#33ff33` (verde terminale classico) — configurabile
  - CSS: `:root { ... }` con custom properties, applicato PRIMA del render React (inline nel tag `<style>` o `<script>`)
- **Design tokens già esistono** in `design-tokens.json` e `index.css`. Verificare coerenza.

### 4.4 Code splitting d3 439KB

- **Problema**: `d3` importato interamente (~439KB non compresso, chunk unico).
- **Soluzione**: 
  - Importare solo moduli usati: `import { scaleLinear, select, line } from 'd3'` invece di `import * as d3 from 'd3'`
  - Se 439KB è compresso (gzip), probabilmente ~120KB — accettabile.
  - Verificare se d3 è un import statico o lazy. Se statico, non caricato nella vista iniziale.
- **Dimensione target**: chunk d3 < 150KB gzip.

### 4.5 TerminalPrompt Warp-style

- **Input multi-riga**: Shift+Enter per newline, Enter per submit. No limiti di riga.
- **Syntax highlight comandi**: `/tool`, `/agent`, `/query` in colore diverso. Niente parser complesso — regex basta.
- **Stato**: Mostrare modalità (CMD vs INPUT) con badge colorato. Già implementato come `TerminalPrompt` ma da rifinire.
- **Shortcuts**: `Ctrl+C` cancella richiesta corrente, `Tab` autocomplete comandi.

### 4.6 TerminalOutput zero-chat

- **Niente bubble UI**: I messaggi non sono incorniciati. Output LLM è testo puro, flusso continuo.
- **Separazione**: Prompt utente in un colore (es. `#33ff33`), risposta Aleph in `#e0e0e0`, errori in `#ff3333`.
- **Dimensione output**: Limitata a 1000 righe visibili, scroll indefinito.

### 4.7 Sidebar tmux-style + SlideOverPanel vim-style

- **Sidebar**: Stato dei sistemi (health, metriche) in barra laterale stretta. Stile tmux: scuro, testo monospaziato, minimo.
- **SlideOverPanel**: Sostituisce i modali per AgentForm, Settings, etc. Scivola da destra, non blocca il resto dell'interfaccia. Già implementato come componente — forse da rifinire.

### 4.8 Empty states + Pagination frontend

- **Stati vuoti**: Ogni lista (agenti, skills, progetti, tools) deve mostrare un messaggio utile quando vuota. Non "no data" — "Nessun agente configurato. Crea il tuo primo agente con /agent create".
- **Pagination frontend**: Solo dopo che FASE 3.5 (API pagination) è implementata. Usare paginazione lato server, non client-side.

---

## Architettura

```
┌──────────────────────────────────────────────────────────┐
│                    Docker Compose                          │
│  ┌──────────┐   ┌──────────┐   ┌──────────────────────┐  │
│  │PostgreSQL │   │   NLP    │   │ Aleph Backend (Go)   │  │
│  │  :5432    │   │ Sidecar  │   │  :8080               │  │
│  │           │   │  :8001   │   │                      │  │
│  │System DB  │   │          │   │ ┌──────────────────┐ │  │
│  │(audit,    │   │Sentiment │   │ │ Middleware Stack │ │  │
│  │agents,    │   │Analysis  │   │ │ Rate Limiter    │ │  │
│  │api_keys)  │   │          │   │ │ Security Hdrs   │ │  │
│  └──────────┘   └──────────┘   │ │ Request ID      │ │  │
│                                 │ │ Timeout/Rety    │ │  │
│  ┌──────────────────────┐      │ │ Circuit Brkr    │ │  │
│  │  DuckDB               │      │ │ Prometheus      │ │  │
│  │  :file (aleph.duckdb) │      │ └──────────────────┘ │  │
│  │                       │      │                      │  │
│  │ Analytics + Workflow  │      │ ┌──────────────────┐ │  │
│  │ + MemStore (vectors)  │      │ │ Subsystems       │ │  │
│  │ + Tool Execution      │      │ │ MemStore         │ │  │
│  └──────────────────────┘      │ │ GNN Client       │ │  │
│                                 │ │ Workflow Engine  │ │  │
│  ┌──────────────────────┐      │ │ DecisionEngine   │ │  │
│  │ Frontend (embedded)   │      │ │ Health/MCP/Diag  │ │  │
│  │ React 18 + Tailwind  │      │ └──────────────────┘ │  │
│  │ Served via embed.FS  │      │                      │  │
│  │ (no separate server)  │      │ └──────────────────────┘  │
│  └──────────────────────┘      │                      │
│                                 │                      │
│                                 │ HTTP/2 + h2c         │
│                                 │ ConnectRPC API       │
│                                 └──────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### Flussi principali

```
User → Terminal → ConnectRPC QueryService/Chat → Middleware → QueryHandler
  → [Circuit Breaker] → NLP (sentiment)
  → [Circuit Breaker] → LLM Provider
  → [Circuit Breaker] → MemStore (save/retrieve context)
  → [Decision Engine] → Multi-Agent (se necessario)
  → [Workflow Engine] (se task multi-step)
  → Response streaming → TerminalOutput
```

### Gestione errori per subsystem

Ogni subsystem (NLP, GNN, MemStore, LLM, Workflow) segue lo stesso pattern:

```
Call → CircuitBreaker.Execute → 
  Se CLOSED: esegui, conta fallimenti
  Se OPEN: return ErrCircuitOpen (degradato)
  Se HALF_OPEN: riprova, se ok → CLOSED, se ancora fallisce → OPEN
```

Se subsystem non disponibile → log Warn + return fallback (nessun crash).

---

## Dipendenze tra fasi

```
FASE 0 (nessuna dipendenza)
  ↓
FASE 1 (nessuna dipendenza esterna)
  ↓
FASE 2 (dipende dal Docker compose di FASE 0 per test di integrazione)
  ↓
FASE 3 (dipende da rate limiting di FASE 1 + wiring di FASE 2)
  ↓
FASE 4 (dipende da API pagination di FASE 3 + backend stabile)
```

Ogni fase può essere sviluppata in isolamento. Le dipendenze sono solo per test end-to-end.

---

## Criteri di completamento

Una fase è completa quando:
1. Tutti i task sono implementati e testati
2. `go build ./...` → zero errori
3. `go test ./...` → tutti passano
4. `npx tsc --noEmit` → zero errori
5. `npx vite build` → zero errori, chunk sizes verificati
6. Docker compose su → sistema funzionante
7. Nessun log WARN/ERROR inaspettato all'avvio

---

## Metriche di successo

- **Tempo di boot**: < 3 secondi (con PostgreSQL)
- **Richieste chat**: p50 < 2s, p95 < 10s
- **Memory**: < 500MB RSS a riposo
- **Disk DuckDB**: < 1GB (sotto carico normale)
- **Chunk frontend**: < 150KB gzip per chunk principale
