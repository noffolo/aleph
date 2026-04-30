# Manuale Tecnico — Aleph-v2

> **Versione:** 2.0.0 · **Ultimo aggiornamento:** Aprile 2026 · **Stato:** Produzione

---

## Indice

1. [Panoramica dell'Architettura](#1-panoramica-dellarchitettura)
2. [Stack Tecnologico](#2-stack-tecnologico)
3. [Backend Go — Architettura Interna](#3-backend-go--architettura-interna)
4. [Motore Decisionale — Ciclo PAORA](#4-motore-decisionale--ciclo-paora)
5. [Frontend React — Architettura UI](#5-frontend-react--architettura-ui)
6. [Sidecar NLP Python](#6-sidecar-nlp-python)
7. [API e Protocolli di Comunicazione](#7-api-e-protocolli-di-comunicazione)
8. [Sicurezza](#8-sicurezza)
9. [Osservabilità e Diagnostica](#9-osservabilità-e-diagnostica)
10. [Deployment e Infrastruttura](#10-deployment-e-infrastruttura)
11. [Testing](#11-testing)
12. [Glossario dei Codici Errore](#12-glossario-dei-codici-errore)

---

## 1. Panoramica dell'Architettura

Aleph-v2 è una piattaforma di intelligenza predittiva multi-agente con architettura a tre livelli:

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React/TS)                       │
│  TerminalView · CopilotView · SlideOver · Cmd+K Palette     │
│  Zustand Composite Store · SSE Streaming · ConnectRPC        │
└────────────────────────┬────────────────────────────────────┘
                         │ ConnectRPC (HTTP/2) + SSE + REST
┌────────────────────────┴────────────────────────────────────┐
│                    Backend Go                                 │
│  QueryHandler · ChatSession · DecisionEngine (PAORA)        │
│  13 ConnectRPC Services · Sandbox · Health · Diagnostic      │
│  7 Middleware · Genesis · Tools Registry · Audit             │
└──────────┬────────────────────────────────┬────────────────┘
           │ gRPC (HTTP/2 cleartext)        │ DuckDB (read-only)
┌──────────┴──────────┐     ┌───────────────┴────────────────┐
│  Python NLP Sidecar  │     │         PostgreSQL 16          │
│  Sentiment · ONNX    │     │    API Keys · Audit · Chat     │
│  Ensemble Prophet/GBM│     └────────────────────────────────┘
│  DuckDB read-only    │
└──────────────────────┘
```

**Principi di design:**

- **Terminal-first**: l'interfaccia primaria è un terminale interattivo con comandi slash, palette Cmd+K, effetti scanline/glow opzionali
- **PAORA loop**: ogni azione passa attraverso Plan → Act → Observe → Reflect → Admit con degradazione graceful
- **Sicurezza defense-in-depth**: ogni input è validato, ogni tool è sandboxed, ogni API key è cifrata AES-256-GCM
- **Osservabilità nativa**: OpenTelemetry + Prometheus + slog strutturato sin dal primo giorno

---

## 2. Stack Tecnologico

### Backend

| Componente | Tecnologia | Versione |
|-----------|-----------|----------|
| Linguaggio | Go | 1.25.0 (go.mod) |
| RPC | ConnectRPC | v1.19.1 |
| gRPC | google.golang.org/grpc | v1.80.0 |
| Database relazionale | PostgreSQL 16 | via pgx/v5 |
| Database analitico | DuckDB | via go-duckdb v1.8.5 |
| Telemetria | OpenTelemetry | v1.43.0 |
| Metriche | Prometheus client_golang | v1.23.2 |
| Configurazione | Viper | v1.20.1 |
| PDF generation | gofpdf | v1.16.2 |
| DSL parsing | participle/v2 | v2.1.4 |
| Cache LRU | golang-lru/v2 | v2.0.7 |
| UUID | google/uuid | v1.6.0 |
| Rate limiting | golang.org/x/time | v0.15.0 |
| Testing | testify | v1.11.1 |

### Frontend

| Componente | Tecnologia | Versione |
|-----------|-----------|----------|
| Framework | React | 18.3.1 |
| Linguaggio | TypeScript | 5.x |
| Build | Vite | 5.x |
| Stili | Tailwind CSS | 3.x |
| State management | Zustand | 4.5.2 |
| RPC client | ConnectRPC (connect-web) | — |
| Validazione | Zod | — |
| Icone | lucide-react | 0.300.0 |
| Componenti UI | @base-ui/react | — |
| Testing unità | Vitest | 4.1.5 |
| Testing E2E | Playwright | — |
| Testing componenti | @testing-library/react | 16.3.2 |

### Sidecar NLP

| Componente | Tecnologia |
|-----------|-----------|
| Linguaggio | Python 3.12 |
| gRPC server | grpcio, grpcio-testing |
| Embeddings | ONNX Runtime + all-MiniLM-L6-v2 |
| Previsione | Prophet + GBM Monte Carlo (100 paths, 252 giorni) |
| Analisi sentiment | Keyword-based ITA/EN (30+ keyword) |
| Database | DuckDB (read-only, injection guard) |
| Testing | pytest, pytest-asyncio |

---

## 3. Backend Go — Architettura Interna

### 3.1 Struttura dei Package

Il backend è organizzato in 35+ package sotto `internal/`, seguendo il principio di separazione delle responsabilità:

```
internal/
├── api/
│   ├── handler/       # 33 handler — QueryHandler, ChatSession, NLPHandler...
│   ├── proto/          # Protobuf definitions (ConnectRPC + gRPC)
│   ├── sse/            # Server-Sent Events broker
│   └── routes/         # RegisterConfig + RegisterRoutes
├── decision/           # Motore decisionale PAORA (Engine, Planner)
├── diagnostic/          # ErrorPattern classification (severità, root cause ITA)
├── errors/              # APIError con codici errore italiano
├── genesis/             # Suggester → Sandbox → VetoRegistry pipeline
├── gnn/                 # Graph Neural Network per link prediction
├── health/              # HealthChecker + HistoryStore (ring buffer)
├── ingestion/           # Engine + sources (RSS, GitHub, CSV, JSON, sitemap, sheets, email)
├── llm/                 # Provider interface per LLM (Ollama, OpenAI)
├── mcp/                 # DiscoveryEngine, MCPService, SSRF protection
├── memory/              # VSS MemoryStore (DuckDB array_cosine_similarity)
├── middleware/           # 8 HTTP middleware + 6 ConnectRPC interceptor
├── repository/          # MetadataRepository (30+ CRUD), AuditRepository, ToolCache
├── sandbox/              # ExecSandbox, Verifier, SecurityScanner, CommandAllowlist
├── service/
│   ├── watcher/         # File Watcher (fsnotify) con auto-ingestion
│   └── notification/    # Servizio notifiche
├── telemetry/            # OTel + Prometheus dual instrumentation
└── tools/                # Registry + 5 subpackage (finance, osint, humanecosystems, codeflow, adaptation)
```

### 3.2 Middleware Stack

Ogni richiesta HTTP attraversa 8 middleware nell'ordine:

```
CORSHandler → CSRFProtection → AuthMiddleware → SecurityHeaders → AuditMiddleware
    → RateLimitMiddleware → TimeoutMiddleware → BulkheadMiddleware
```

Per ConnectRPC, 7 intercettori si aggiungono:

```
RecoveryInterceptor → AuthInterceptor → SecurityHeaders → AuditInterceptor
    → RateLimitInterceptor → TimeoutInterceptor → BulkheadInterceptor
```

**AuthMiddleware/Interceptor**: legge l'header `X-Aleph-Api-Key`, calcola SHA-256, confronta con il database (cifrato AES-256-GCM a riposo). Restituisce `projectID` nel context. Gli endpoint `/readyz`, `/livez`, `/api/v1/healthz` e `/metrics` sono esclusi dall'autenticazione.

**ErrorMiddleware (handler)**: converte errori interni in `APIError` con codici ITA localizzati (vedi sez. 12). Risposte JSON strutturate con `code`, `message`, `details`.

**SecurityHeaders**: applica Content-Security-Policy (`default-src 'self'`, `script-src 'self'`, `style-src 'self'`, niente `unsafe-inline`), X-Content-Type-Options, X-Frame-Options, Referrer-Policy.

**CSRFProtection**: middleware per richieste non-GET. Valida header Origin/Referer contro la lista origins consentite. Le richieste senza Origin/Referer (CLI, script) sono permesse.

**RateLimitMiddleware**: limita richieste per IP, usando `X-Forwarded-For` se presente, con fallback a `X-Real-IP` e infine `RemoteAddr`. Configurabile via environment.

**BulkheadMiddleware**: limita concorrenza per endpoint. Default: 100 connessioni simultanee per `/api/v1/events` (SSE).

### 3.3 Handler Principali

**QueryHandler** (`internal/api/handler/query.go`):
- Entry point per la chat
- `Chat()` delega a `NewChatSession()` + `session.Run()`
- `resolveAgent()` estrae la configurazione agente con validazione

**ChatSession** (`internal/api/handler/chat_session.go`):
- Struct con campi: ctx, stream, handler, chatMessages, tools, agent, engine
- Loop fino a 5 iterazioni
- Fasi: Plan → Act → Observe → Reflect → Admit
- `callLLM()`, `streamResponse()`, `executeAndStreamTool()`, `appendToolCallToMessages()`, `appendToolResult()`
- Degradazione graceful: se `engine == nil`, usa planning euristico senza LLM

**SSEHandler** (`internal/api/handler/sse_handler.go`):
- Broker con fan-out a tutti i client connessi
- `Stream()` gestisce connessioni long-lived
- Fail-closed: disconnessione su errore di autenticazione

**NLPHandler** (`internal/api/handler/nlp_handler.go`):
- Bridge gRPC verso sidecar Python su `ALEPH_NLP_ADDR`
- Circuit breaker con fallback di risposta sintetica
- `AnalyzeSentiment`, `StreamPredictions`, `RecordFeedback`

### 3.4 Repository Layer

**MetadataRepository** (`internal/repository/metadata.go`):
30+ metodi CRUD per tools, agents, skills, projects. Tutte le query usano parametri posizionali (`$1`, `$2`) con validazione `validName()` regex. Cache LRU per lookup frequenti. `ToolRecord` esteso con Category, Version, HealthStatus, LastCheckedAt, SourceType.

**AuditRepository** (`internal/repository/audit.go`):
Registra ogni operazione mutations (create, update, delete) con timestamp, projectID, azione, dettagli JSON.

### 3.5 Sandbox e Sicurezza Tool Execution

**ExecSandbox** (`internal/sandbox/`):
- Esecution dei tool con timeout e resource limits
- `SecurityScanner` analizza il codice del tool prima dell'esecuzione
- `CommandAllowlist` con 14 comandi permessi e 5 flag bloccati (`-rf`, `--force`, `--no-dry-run`, `-exec`, `--allow-root`)
- Regex per metacaratteri shell: `;|&$\`<>`
- `CodeMetrics` calcola complessità ciclomatica e linee di codice
- `ScaffoldGenerator` genera template per nuovi tool

**Network isolation**: `network_mode: none` nel container Docker per il sandbox, `read_only: true` per il filesystem.

### 3.6 Health Check System

**HealthChecker** (`internal/health/checker.go`):
- Scheduler periodico (5 minuti di default)
- `BuiltinChecker` implementa controlli base (connectivity, disk, memory)
- `MCPHealthProvider` monitora tool MCP STDIO
- `HistoryStore` con ring buffer per storico health per tool
- Endpoint `/api/v1/tools/health` e `/api/v1/tools/{id}/health/history`

### 3.7 Genesis (Tool Suggestion Pipeline)

**GenesisEngine** (`internal/genesis/`):
- `Suggest()`: Suggester.Analyze() → Sandbox.Validate() → VetoRegistry.Register()
- `Approve()`: delega a VetoRegistry.Approve()
- `Suggester`: V1 stub che ritorna lista vuota (futuro: NLP-based suggestions)
- `Sandbox.Validate()`: controlla 9 dangerous patterns (os/exec, syscall, unsafe, reflect, os.Remove, os.RemoveAll, os.Chmod, net.Listen, net.Dial) con ctx cancellation
- `VetoRegistry`: Register/Approve/Reject/ListPending con TTL, cleanup goroutine, `Shutdown()` per context cancellation

---

## 4. Motore Decisionale — Ciclo PAORA

Il motore decisionale implementa il ciclo **Plan → Act → Observe → Reflect → Admit** come decision loop autonomo per ogni sessione di chat.

### 4.1 Architettura

```
                    ┌─────────────────┐
                    │   ChatSession    │
                    │   Run() loop     │
                    │   max 5 iters    │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         ┌────▼────┐  ┌─────▼─────┐  ┌─────▼──────┐
         │  Plan    │  │    Act    │  │  Observe    │
         │ Planning │  │ Execution │  │  Feedback   │
         └────┬─────┘  └─────┬─────┘  └─────┬──────┘
              │              │              │
         ┌────▼────┐  ┌─────▼─────┐  ┌─────▼──────┐
         │Reflect  │  │  Admit    │  │  Degraded  │
         │Evaluate │  │  Finish   │  │  Fallback  │
         └─────────┘  └───────────┘  └────────────┘
```

### 4.2 Engine Interface

```go
type DecisionEngine interface {
    Plan(ctx, msg, projectID, agentID, ontContent, agent) (*PlanResult, error)
    PlanWithProvider(ctx, msg, projectID, agentID, ontContent, agent, provider) (*PlanResult, error)
    Act(ctx, step, projectID) (*ActResult, error)
    Observe(ctx, step, result) (*Observation, error)
    Reflect(ctx, plan, observations) (*PlanResult, error)
    Admit(ctx, results, maxAttempts) (bool, error)
    BuildToolsMap(ctx) []map[string]interface{}
}
```

### 4.3 Flusso Operativo

1. **ChatSession.Run()** entra nel loop (max 5 iterazioni)
2. **Prima iterazione**: chiama `engine.PlanWithProvider()` con il provider LLM per-request
   - Se provider nil o errore LLM → degrada a `Plan()` euristico (keyword matching)
   - Keyword mapping: search/find/query/show/data/object → `search_data`, sentiment/feeling/opinion → `analyze_sentiment`, trust/score/brier/prediction → `get_trust_score`
3. **Per ogni tool nel piano**: `engine.Observe()` dopo l'esecuzione
4. **Dopo tutte le osservazioni**: `engine.Reflect()` valuta se continuare
   - Se ultima osservazione fallisce → `CanProceed = false` → stop
5. **Fine loop**: `engine.Admit()` verifica se sufficiente (result count ≥ maxAttempts o ultimo risultato ha errore)

### 4.4 Degradazione Graceful

Quando il provider LLM non è disponibile:
- `PlanWithProvider()` cade su `Plan()` che usa keyword matching
- Il `PlanResult` contiene `Reason: "degraded mode: heuristic planning (no LLM provider)"`
- Il sistema continua a funzionare con capacità ridotta ma garantita

---

## 5. Frontend React — Architettura UI

### 5.1 Architettura dello Store

Zustand composite store con 6 slice:

```
useStore()
├── authSlice      — projectID, apiKey, isAuthenticated
├── navigationSlice — activeView, selectedAgent, selectedSkill
├── copilotSlice   — messages, isLoading, streamingContent
├── workspaceSlice — agents, skills, tools, datasources, library
├── healthSlice    — toolHealth, systemHealth
└── uiSlice        — toasts, slideOverPanel, theme, editingState
```

**Data flow**:

```
TerminalPrompt → parseCommand/executeCommand
    ├── /slash commands (16 built-in)
    ├── Cmd+K palette → CommandPalette
    └── free text → queryClient.chat() stream SSE
        → InlineRenderer / SlideOverContent
```

### 5.2 Componenti Principali

| Componente | Path | Responsabilità |
|-----------|------|----------------|
| App.tsx | `src/App.tsx` | Router principale, 6 React.lazy imports, TerminalView di default |
| TerminalView | `src/components/terminal/TerminalView.tsx` | Layout terminale con header agente, CopilotView wrapper |
| CopilotView | `src/components/copilot/CopilotView.tsx` | Chat interattiva con streaming SSE |
| CommandPalette | `src/components/terminal/CommandPalette.tsx` | Palette Cmd+K con fuzzy search |
| SlideOverPanel | `src/components/SlideOverPanel.tsx` | Pannello laterale per form complessi (11 form) |
| InlineRenderer | `src/components/terminal/InlineRenderer.tsx` | Rendering markdown/tool inline nel chat |
| AlephErrorBoundary | `src/components/errors/AlephErrorBoundary.tsx` | Error boundary per-View con recovery |
| TerminalEffects | `src/components/terminal/TerminalEffects.tsx` | Effetti scanline/flicker/glow via Zustand toggle |

### 5.3 Slash Commands (16)

Definiti in `src/commands/slashCommands.ts`:

| Comando | Descrizione |
|---------|-------------|
| `/help` | Mostra tutti i comandi disponibili |
| `/clear` | Pulisce la sessione di chat |
| `/model` | Cambia modello LLM |
| `/agent` | Elenca/switcha agenti |
| `/tool` | Gestione tool (install/list/health/diagnose) |
| `/skills` | Elenca competenze agente |
| `/status` | Stato della connessione |
| `/export` | Esporta conversazione |
| `/diagnose` | Esegue diagnostica |
| `/theme` | Cambia tema chiaro/scuro |
| `/debug` | Modalità debug |

Gli altri 6 sono alias o estensioni dei precedenti.

### 5.4 Slide-Over Forms (11)

| Form | View | Responsabilità |
|------|------|----------------|
| AgentForm | agents-view | Crea/modifica configurazione agente |
| SkillForm | skills-view | Crea/modifica competenza |
| ToolForm | tools-view | Registra/configura tool |
| DataSourceForm | datasources-view | Connetti fonti dati (upload/DB/URL) |
| LibraryForm | library-view | Gestisci risorse libreria |
| ComponentForm | components-view | Gestisci componenti UI |
| ConfirmDialog | globale | Conferma azione distruttiva |
| SettingsForm | settings-view | Configurazioni globali |
| HealthDetailView | health-view | Dettaglio health tool |
| PredictView | predict-view | Visualizzazione predizioni |
| ExploreView | explore-view | Esplorazione ontologia |

### 5.5 Hook Personalizzati (9)

| Hook | Responsabilità |
|------|----------------|
| `useAgentActions` | CRUD agent |
| `useSkillActions` | CRUD competenze |
| `useToolActions` | CRUD e esecuzione tool |
| `useDataSourceActions` | Gestione fonti dati |
| `useLibraryActions` | Gestione libreria |
| `useAppActions` | Azioni globali (onSend, onConfirmAction, onNavigate) |
| `useChat` | Chat con streaming SSE |
| `useInfiniteQueries` | Paginazione infinita con Zod validation |
| `useSSE` | Connessione SSE con riconnessione automatica |

### 5.6 Client API

12 client generati da `api/client/factory.ts` usando ConnectRPC. Ogni client mappa 1:1 con i 12 servizi ConnectRPC backend. Validazione response con Zod schemas (22 definiti in `api/schemas/`).

### 5.7 Design System

- **Design tokens**: `src/styles/design-tokens.json` — palette (#080810 base scuro), spacing 8px grid, borderRadius (terminal=0, card=8px), elevation, shadow, transition
- **Font**: JetBrains Mono 13px body / 11px meta, `tabular-nums`, `no-ligatures`
- **CSS volatility layers**: `.vol-static` → `.vol-structural` → `.vol-interactive` → `.vol-signal`
- **Glassmorphism**: `.glass-panel` con backdrop-blur
- **Toast**: `ToastContainer` + `ToastBar` renderizzati in App.tsx, hook `useToast()` da uiSlice

---

## 6. Sidecar NLP Python

### 6.1 Servizi gRPC

Il sidecar espone 3 RPC su porta 8001:

| RPC | Input | Output | Descrizione |
|-----|-------|--------|-------------|
| `AnalyzeSentiment` | `AnalyzeSentimentRequest{text}` | `AnalyzeSentimentResponse{score, label}` | Analisi sentiment keyword-based ITA/EN |
| `StreamPredictions` | `StreamPredictionsRequest{...}` | stream `PredictionResponse` | Ensemble Prophet+GBM Monte Carlo |
| `RecordFeedback` | `RecordFeedbackRequest{...}` | `RecordFeedbackResponse{}` | Registra feedback per calibrazione Brier |

### 6.2 Sentiment Analysis

Implementazione `analyze_sentiment_simple()`:
- **Dizionari**: 15+ keyword positive italiane + 15+ keyword positive inglesi, altrettante negative
- **Algoritmo**: tokenizzazione del testo, lookup nei dizionari, punteggio normalizzato [-1, +1]
- **Soglie**: score ≥ 0.05 → "positive", score ≤ -0.05 → "negative", altrimenti "neutral"
- **Fallback integrato**: nessuna dipendenza esterna per il sentiment

### 6.3 Embeddings e Ensemble

- **ONNX BERT**: all-MiniLM-L6-v2 caricato da `/app/onnx_model/`
- **Ensemble Prophet+GBM**:
  - Prophet: trend + stagionalità
  - GBM Monte Carlo: 100 path randomizzate, 252 giorni orizzonte
  - Calibrazione mercati esterni: Polymarket (peso 0.4) + Metaculus (peso 0.6)
- **Fallback sintetico**: 20 giorni di dati casuali quando il DB è vuoto

### 6.4 DuckDB

Connessione read-only con injection guard. Path configurabile via `ALEPH_DUCKDB_PATH`.

### 6.5 Docker

```dockerfile
FROM python:3.12-slim
# gcc/g++ per dipendenze native
# pip install -r requirements.txt
# Non-root user: aleph
# HEALTHCHECK gRPC su localhost:50051
EXPOSE 8001
ENTRYPOINT ["python", "main.py"]
```

**requirements.txt** (17 dipendenze + test):
- onnxruntime, grpcio>=1.80.0, prophet, scikit-learn, duckdb
- pytest>=8.0.0, grpcio-testing>=1.80.0, pytest-asyncio>=0.25.0 (test)

**Test**: 11 test in `nlp/tests/` — 5 sentiment (test_sentiment.py), 6+ gRPC (test_grpc.py con FakeNLPServicer)

**Graceful shutdown**: Handler SIGTERM/SIGINT con grace period 5s

---

## 7. API e Protocolli di Comunicazione

### 7.1 Endpoint REST

| Metodo | Path | Autenticazione | Descrizione |
|--------|------|---------------|-------------|
| GET | `/readyz` | Nessuna | Readiness probe (503 durante drain) |
| GET | `/livez` | Nessuna | Liveness probe |
| GET | `/api/v1/healthz` | Nessuna | Health check per Docker/load balancer |
| GET | `/metrics` | Nessuna | Metriche Prometheus |
| GET | `/api/v1/tools` | X-Aleph-Api-Key | Lista tool (CRUD) |
| POST | `/api/v1/tools` | X-Aleph-Api-Key | Crea tool |
| GET | `/api/v1/tools/categories` | X-Aleph-Api-Key | Lista categorie tool |
| POST | `/api/v1/tools/execute/{category}/{name}` | X-Aleph-Api-Key | Esegui tool in sandbox |
| POST | `/api/v1/tools/call` | X-Aleph-Api-Key | Chiama tool per nome |
| POST | `/api/v1/tools/register` | X-Aleph-Api-Key | Registra nuovo tool |
| GET | `/api/v1/tools/health` | X-Aleph-Api-Key | Health di tutti i tool |
| GET | `/api/v1/tools/verify` | X-Aleph-Api-Key | Verifica integrità tool |
| GET | `/api/v1/tools/{id}/health/history` | X-Aleph-Api-Key | Storico health per tool |
| POST | `/api/v1/tools/suggest` | X-Aleph-Api-Key | Suggerisci tool (Genesis pipeline) |
| POST | `/api/v1/tools/suggest/approve` | X-Aleph-Api-Key | Approva suggerimento |
| GET | `/api/v1/codeflow/graph` | X-Aleph-Api-Key | Grafo CodeFlow |
| GET | `/api/v1/codeflow/metrics` | X-Aleph-Api-Key | Metriche CodeFlow |
| GET | `/api/v1/codeflow/executions` | X-Aleph-Api-Key | Lista esecuzioni CodeFlow |
| GET | `/api/v1/codeflow/engines` | X-Aleph-Api-Key | Lista motori CodeFlow |
| GET | `/api/v1/diagnostic/patterns` | X-Aleph-Api-Key | Pattern diagnostici |
| GET | `/api/v1/events` | X-Aleph-Api-Key (query) | SSE stream |
| GET | `/swagger.json` | Nessuna | OpenAPI specification |

### 7.2 Servizi ConnectRPC (13)

Ogni servizio usa interceptors ConnectRPC per auth, audit, rate-limit, timeout, bulkhead, recovery:

| Servizio | RPC | Descrizione |
|----------|-----|-------------|
| QueryService | Chat, GetHistory, StreamHistory, ... | Chat con streaming |
| AgentService | List, Get, Create, Update, Delete, GetDefault | CRUD agent |
| SkillService | List, Get, Create | Gestione competenze |
| ToolService | List, Get, Create | Gestione tool |
| LibraryService | List, Get, Create, Update, Delete | Gestione libreria |
| ProjectService | List, Get, Create, Update, Delete, GetDefault | Gestione progetti |
| NotificationService | List, MarkRead | Notifiche |
| AuthService | Validate, Refresh, Revoke | Autenticazione |
| IngestionService | Start, Status, Cancel, List, Get, Retry | Ingestione dati |
| SandboxService | Verify, Execute | Esecuzione sandboxed |
| RegistryService | Register, List, Get, Health | Registro tool esterni |
| NLPService | AnalyzeSentiment, StreamPredictions, RecordFeedback | Bridge sidecar |

### 7.3 SSE (Server-Sent Events)

Endpoint `/api/v1/events` con:
- Autenticazione query parameter `api_key` (SHA-256 hashed)
- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`
- Riabilitazione automatica lato client con exponential backoff

### 7.4 Protobuf

La definizione dei messaggi è in `internal/api/proto/aleph/v1/`. Generazione Go con `buf.gen.yaml`:
- `protoc-gen-go` v5.29.2
- `protoc-gen-connect-go` v1.19.1

Generazione Python: `make proto-python`

---

## 8. Sicurezza

### 8.1 Autenticazione

- **Header**: `X-Aleph-Api-Key` richiesto per tutti gli endpoint autenticati
- **Validazione**: SHA-256 hash della chiave → confronto con database
- **Cifratura a riposo**: API key cifrate con AES-256-GCM, chiave `KEY_ENCRYPTION_KEY`
- **Environment**:
  - `ALEPH_API_KEY_SECRET_BACKEND` — chiave per cifratura backend
  - `ALEPH_API_KEY_SECRET` — chiave per sidecar Python
  - `KEY_ENCRYPTION_KEY` — chiave master AES-256-GCM

### 8.2 Sandbox

- **Esecuzione tool**: `ExecSandbox` con timeout e resource limits
- **Codice tool**: `SecurityScanner` blocca 9 pattern pericolosi e `CommandAllowlist` con 14 comandi permessi
- **Network**: `network_mode: none` nel container Docker
- **Filesystem**: `read_only: true` nel container sandbox
- **Validazione input**: `validName()` regex su tutti i nomi tool, parametri posizionali ($1, $2) anti-SQLi

### 8.3 CORS

Configurabile via `CORS_ALLOWED_ORIGINS`. Default: `http://localhost:5173`, `http://localhost:3000`. Validazione rigorosa: deve iniziare con `http://` o `https://`. Origin non valide vengono saltate con warning slog.

### 8.4 Audit Logging

- Ogni mutation (create, update, delete) è registrata da `AuditRepository`
- Include: timestamp, projectID, azione, dettagli JSON
- Middleware `AuditInterceptor` logga ogni richiesta RPC

### 8.5 Protezione SSRF

`internal/mcp/ssrf.go`:
- Validazione URL per connections MCP
- Blocklist di indirizzi interni (127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
- Validazione schema (solo http/https)

---

## 9. Osservabilità e Diagnostica

### 9.1 Telemetria

**OpenTelemetry** (`internal/telemetry/`):
- Tracing: `otlptracegrpc` exporter
- Metrics: `otlpmetricgrpc` exporter
- Span per decisione PAORA (Plan, Act, Observe, Reflect, Admit)
- Span per tool execution, NLP call, SSE connection

**Prometheus**:
- Endpoint `/metrics` con metriche custom
- Metriche per tool health, SSE connections, request latency, error rate
- `telemetry.MetricsHandler()` wrappa il registry Prometheus

### 9.2 Diagnostica

**DiagnosticMonitor** (`internal/diagnostic/patterns.go`):
- 6 tipi di pattern: ErrorPattern con classificazione, severità, root cause in italiano
- `GetPatterns()` restituito via `/api/v1/diagnostic/patterns`
- Pattern: connessione fallita, timeout, auth fallita, tool non trovato, limite rate, loop di errore

### 9.3 Health Check

**HealthChecker** (`internal/health/`):
- Schedule periodico (5 minuti default)
- Controlli: connectivity, disk, memory
- Ring buffer `HistoryStore` per storico per-tool
- Endpoint: `/api/v1/tools/health` (tutti), `/api/v1/tools/{id}/health/history` (singolo)

### 9.4 Logging

`slog` strutturato贯穿整个 codebase:
- Livelli: DEBUG, INFO, WARN, ERROR
- Campi contestuali: projectID, toolID, requestID
- Nessun log di API key o dati sensibili

---

## 10. Deployment e Infrastruttura

### 10.1 Docker Compose

4 servizi:

```yaml
aleph-backend:    # Go backend, porta 8080
aleph-nlp-sidecar:  # Python NLP, porta 8001, user: aleph
aleph-frontend:   # React/Vite → nginx, porta 5174→80
aleph-db:        # PostgreSQL 16, porta 5432
```

**Volume**:
- `aleph-pgdata`: dati PostgreSQL persistenti
- `aleph-data`: dati applicazione (DuckDB, progetti)
- `aleph-duckdb`: database DuckDB
- `./aleph_tools`: tool configurati
- `./nlp/models`: modello ONNX

**Rete**: `aleph-network` (bridge default)

### 10.2 Health Check Container

| Servizio | Healthcheck |
|----------|-------------|
| aleph-backend | `/readyz` (HTTP 200/503), `/livez` (HTTP 200) |
| aleph-nlp-sidecar | gRPC channel check su `localhost:50051` (interval 30s, timeout 10s, start_period 10s, retries 3) |
| aleph-db | `pg_isready -U postgres -d aleph` (interval 5s, timeout 5s, retries 5) |

### 10.3 Environment Variables

```bash
# Obbligatorio
POSTGRES_DSN=postgres://postgres:PASSWORD@aleph-db:5432/aleph?sslmode=disable
KEY_ENCRYPTION_KEY=<32-byte AES key>
ALEPH_API_KEY_SECRET_BACKEND=<backend encryption key>
ALEPH_API_KEY_SECRET=<sidecar encryption key>
POSTGRES_PASSWORD=<postgres password>
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com

# Opzionale
SERVER_ADDRESS=:8080
DUCKDB_PATH=/app/aleph_registry.duckdb
NLP_ADDR=aleph-nlp-sidecar:8001
PYTHON_CMD=python3
GO_CMD=go
VITE_API_BASE_URL=http://aleph-backend:8080
ALEPH_REGISTRY_URL=http://aleph-backend:8080
```

### 10.4 Build e Development

```makefile
make build          # Build frontend + Go binary
make run            # Build + run
make frontend-dev    # Vite dev server (porta 5173)
make nlp-dev         # Python sidecar development
make build-models    # Convert ONNX model
make proto-python    # Rigenera protobuf Python
make clean           # Rimuovi artefatti build
```

### 10.5 CI/CD

5 job:
1. **Go Build**: `go build ./...`, `go vet ./...`, `go test ./...`
2. **Go Lint**: `.golangci.yml` (20 linter)
3. **Frontend Build**: `npm run build`, `npx tsc --noEmit`
4. **Frontend Test**: `vitest run`
5. **E2E Test**: `playwright test`

---

## 11. Testing

### 11.1 Go

**28+ package di test** con `testify`:

| Package | Test principali |
|---------|----------------|
| `decision/` | 13 test — Plan, PlanWithProvider, Act, Observe, Reflect, Admit, BuildToolsMap, inferTools, isKnownTool |
| `genesis/` | 15 test — Sandbox Validate (9 pattern + ctx cancellation), VetoRegistry (Register/Approve/Reject/ListPending/TTL/concurrent/Shutdown), Suggester stub, GenesisEngine |
| `sandbox/` | ExecSandbox, SecurityScanner, CommandAllowlist |
| `health/` | HealthChecker, HistoryStore |
| `diagnostic/` | Pattern classification |
| `handler/` | QueryHandler, ChatSession, SSE |

**Comandi**:
```bash
go test ./...                  # Tutti i test
go test ./internal/decision/  # Test singolo package
go test -race ./...           # Con race detector
```

### 11.2 Frontend

- **Vitest**: unit test con `@testing-library/react`
- **Playwright**: test E2E con mock interceptor SSE
- **TypeScript**: `npx tsc --noEmit` per type-check

```bash
npx vitest run                # Unit test
npx playwright test            # E2E test
npx tsc --noEmit              # Type check
```

### 11.3 Python

- **pytest** con `grpcio-testing`:
  - `test_sentiment.py`: 5 test (positive, negative, neutral, empty, mixed)
  - `test_grpc.py`: 11 test (AnalyzeSentiment 8, StreamPredictions 1, RecordFeedback 2)
  - `conftest.py`: `FakeNLPServicer` mock con `AnalyzeSentimentResponse(score=0.5, label="positive")`

```bash
cd nlp && pytest               # Tutti i test
cd nlp && pytest tests/test_sentiment.py -v  # Singolo file
```

---

## 12. Glossario dei Codici Errore

Il sistema `APIError` (in `internal/errors/`) utilizza codici localizzati in italiano:

| Codice | Significato | HTTP Status |
|--------|-------------|-------------|
| `ERR_AUTENTICAZIONE` | Credenziali non valide o assenti | 401 |
| `ERR_AUTORIZZAZIONE` | Permessi insufficienti per l'operazione | 403 |
| `ERR_NON_TROVATO` | Risorsa richiesta non esistente | 404 |
| `ERR_VALIDAZIONE` | Dati input non validi o malformati | 400 |
| `ERR_RATE_LIMIT` | Numero di richieste supera il limite consentito | 429 |
| `ERR_INTERNO` | Errore interno del server imprevisto | 500 |
| `ERR_SERVIZIO_NON_DISPONIBILE` | Dipendenza esterna non raggiungibile | 503 |
| `ERR_TIMEOUT` | Scadenza del tempo limite per l'operazione | 504 |
| `ERR_LIMITA_DIMENSIONE` | Payload o dati oltre la dimensione massima consentita | 413 |

Ogni errore include `code`, `message` (in italiano), e `details` (opzionale, JSON strutturato).

---

## Appendice A: Struttura Directory Completa

```
aleph-v2/
├── main.go                          # Entry point
├── go.mod / go.sum                  # Dipendenze Go
├── Makefile                         # Build, run, dev, proto
├── Dockerfile                       # Multi-stage Go build
├── docker-compose.yml               # 4 servizi (backend, sidecar, frontend, postgres)
├── .golangci.yml                    # 20 linter
├── buf.gen.yaml                     # Protobuf Go generation
├── .env.example                     # Environment template
│
├── internal/
│   ├── api/
│   │   ├── handler/                 # 33 handler Go
│   │   ├── proto/                   # Protobuf definitions
│   │   │   └── aleph/
│   │   │       ├── v1/              # Core services protobuf
│   │   │       └── nlp/v1/         # NLP sidecar protobuf
│   │   ├── sse/                     # SSE broker
│   │   └── routes/                  # RegisterConfig + RegisterRoutes
│   ├── decision/                    # PAORA Engine + Planner
│   ├── diagnostic/                  # ErrorPattern classification
│   ├── errors/                      # APIError + codici ITA
│   ├── genesis/                     # Suggester → Sandbox → VetoRegistry
│   ├── gnn/                         # Graph Neural Network
│   ├── health/                      # HealthChecker + HistoryStore
│   ├── llm/                         # Provider interface
│   ├── mcp/                         # DiscoveryEngine, SSRF, JSON-RPC
│   ├── middleware/                   # 7 HTTP + 6 ConnectRPC interceptor
│   ├── repository/                  # Metadata 30+ CRUD, Audit, ToolCache
│   ├── sandbox/                     # ExecSandbox, CommandAllowlist, SecurityScanner
│   ├── telemetry/                   # OTel + Prometheus
│   └── tools/                       # Registry + 5 subpackage
│       ├── finance/                 # Profeta, OpenBB, Sentiment
│       ├── osint/                   # Threat level, Correlation, OSINT tools
│       ├── humanecosystems/         # DuckDB layer, 5 tools
│       ├── codeflow/                # Graph, metrics, engines
│       └── adaptation/              # Pipeline, suggestion, versioning
│
├── frontend/
│   ├── src/
│   │   ├── App.tsx                  # Main router, 6 React.lazy
│   │   ├── store/                   # 6 Zustand slice
│   │   ├── components/
│   │   │   ├── terminal/            # TerminalView, Effects, CommandPalette
│   │   │   ├── copilot/            # CopilotView, ChatBubble
│   │   │   ├── SlideOverPanel.tsx  # Panel laterale
│   │   │   ├── errors/             # AlephErrorBoundary, InlineError
│   │   │   └── ... 30+ componenti
│   │   ├── hooks/                   # 9 hook personalizzati
│   │   ├── api/                     # 12 client ConnectRPC + factory
│   │   ├── schemas/                 # 22 Zod schemas
│   │   ├── commands/                # slashCommands.ts (16 comandi)
│   │   └── styles/                  # design-tokens.json, index.css
│   ├── tailwind.config.js
│   ├── tsconfig.app.json
│   ├── vite.config.ts
│   └── package.json
│
├── nlp/
│   ├── main.py                      # gRPC server con graceful shutdown
│   ├── requirements.txt             # 17+ dipendenze + test
│   ├── Dockerfile                   # Python 3.12-slim, non-root
│   ├── pytest.ini                   # Test configuration
│   ├── tests/
│   │   ├── conftest.py              # FakeNLPServicer
│   │   ├── test_sentiment.py        # 5 test
│   │   └── test_grpc.py             # 11 test
│   └── models/                      # ONNX all-MiniLM-L6-v2
│
├── api/
│   └── proto/
│       └── aleph/
│           ├── v1/                  # Core protobuf
│           └── nlp/v1/             # NLP protobuf
│
└── docs/
    ├── API.md                      # API reference (464 linee)
    ├── ARCHITECTURE.md              # Architettura Go (22 package)
    ├── SECURITY.md                  # Security overview
    ├── threat-model.md              # Modello di minaccia (158 linee)
    ├── CI-CD-README.md              # CI/CD pipeline
    ├── error-glossary.md           # 9 codici errore ITA
    ├── CONTRIBUTING.md              # Linee guida contribuzione
    ├── development-bias-checklist.md
    ├── manuale-tecnico.md           # Questo documento
    └── superpowers/plans/           # Piani di esecuzione
```

---

*Fine del Manuale Tecnico — Aleph-v2 v2.0.0*