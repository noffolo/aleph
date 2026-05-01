# Aleph-v2

**Sistema di Decision Intelligence che trasforma dati eterogenei, segnali ambigui e variabili incerte in scenari espliciti, probabilistici e azionabili.**

---

## Cos'è Aleph

Il problema non è la mancanza di dati. È capire cosa significano, insieme, in un momento specifico.

Aleph è un sistema di Decision Intelligence: un ambiente in cui dati, modelli predittivi e agenti AI collaborano per rendere leggibile la complessità. Non produce risposte automatiche. Produce una mappa migliore dell'incertezza — così che le persone possano decidere con più consapevolezza.

In pratica, Aleph aiuta a rispondere a domande che i dashboard tradizionali non riescono a fare:

- Cosa significano questi dati, nel loro insieme?
- Quali scenari sono plausibili da qui in avanti?
- Quali segnali stanno cambiando, e quanto velocemente?
- Dove il rischio è più alto? Dove c'è margine di manovra?
- Dove serve ancora giudizio umano?

> Aleph è uno dei punti dello spazio che contengono tutti i punti.

---

## A cosa serve

Aleph è pensato per contesti con molte informazioni, segnali deboli e bisogno di sintesi. Può supportare:

- analisi di trend e scenari futuri
- confronto tra ipotesi diverse
- monitoraggio di segnali da fonti eterogenee
- letture strategiche di un contesto
- ricerca, pianificazione e analisi strutturata
- prototipazione di sistemi predittivi e decisionali
- sperimentazione con workflow basati su agenti AI

Aleph non decide al posto di nessuno. Aiuta a vedere meglio il contesto in cui si deve decidere.

---

## Per chi è pensato

### Utenti non tecnici
Aleph si può usare come strumento esplorativo: si osservano scenari, si leggono sintesi, si valutano possibili evoluzioni. L'interfaccia è pensata per rendere accessibili anche situazioni complesse, senza richiedere conoscenze di programmazione.

### Analisti, ricercatori, strategist
Aleph offre un ambiente di esplorazione per dati, scenari, ipotesi e simulazioni. Il sistema combina modelli statistici, machine learning, modelli linguistici e workflow decisionali in un unico spazio di lavoro.

### Sviluppatori
Aleph è una piattaforma tecnica modulare: backend in Go, frontend React/TypeScript, componenti Python per l'intelligence layer, orchestrazione Docker. È possibile estendere il sistema con nuovi strumenti, agenti e skill — sia importandoli da fonti esterne, sia costruendoli direttamente.

---

## Le capacità agentiche

Una delle caratteristiche centrali di Aleph è la sua architettura estendibile tramite agenti e strumenti.

Gli agenti possono analizzare, sintetizzare, proporre interpretazioni o attivare procedure controllate. Possono operare in modo autonomo o collaborativo, all'interno di workflow configurabili.

Gli strumenti (*tools*) e le skill possono essere:

- **importati** da fonti esterne, librerie o sistemi già esistenti;
- **creati da sviluppatori umani** attraverso l'interfaccia o direttamente nel codice;
- **proposti dal sistema** (Genesis), con meccanismi di revisione e approvazione umana (VetoRegistry) prima dell'attivazione.

---

## Funzionalità principali

### Decision Intelligence (PAORA)
Ogni interazione con Aleph segue un ciclo PAORA: **Plan → Act → Observe → Reflect → Admit**. Il sistema pianifica tool call, le esegue in sandbox isolata, osserva i risultati, riflette sull'esito e ammette il risultato o ritenta. Il ciclo è integrato nel backend Go e non richiede orchestrazione esterna.

### Auto-repair Engine
Il sistema rileva e corregge automaticamente anomalie nei dati tramite 7 strategie di fix (valori nulli, outlier, duplicati, vincoli violati, tipi errati, firme temporali, correlazioni). Ogni fix è tracciato e reversibile.

### Auto-suggestion (Genesis)
Il motore Genesis analizza pattern di utilizzo e propone nuovi strumenti e skill. Ogni suggerimento passa attraverso un sandbox di validazione (blocco di codice pericoloso) e un registro di veto con scadenza TTL prima dell'attivazione.

### Memory Store (VSS)
Il sistema memorizza e recupera informazioni rilevanti tramite vettori di similarità (DuckDB `list_cosine_similarity()`) con namespace isolati per progetto. Le memorie sono inserite nel contesto del ciclo decisionale per migliorare la pertinenza delle risposte.

### Scenari predittivi
Il sistema trasforma dati grezzi in scenari leggibili. Uno scenario non è una certezza: è una possibile evoluzione costruita sulla base dei dati disponibili, dei modelli usati e delle ipotesi configurate. Ogni scenario porta con sé un livello di confidenza e le assunzioni su cui si basa.

### Analisi del sentiment (NLP)
L'analisi del sentiment utilizza un approccio euristico basato su dizionari di parole chiave (italiano e inglese). Non vengono impiegati modelli transformer o reti neurali per la classificazione del testo. Il punteggio è una media pesata di conteggi lessicali, con label `positive`/`negative`/`neutral`. Il flag `is_calibrated` è `false` (punteggi non calibrati probabilisticamente).

Il sidecar Python espone anche endpoint predittivi (`StreamPredictions`) e di feedback (`RecordFeedback`) via gRPC, pensati per scenari sperimentali e simulazioni. I dati possono essere reali o sintetici (flag `is_synthetic`). Le predizioni non sostituiscono giudizio umano.

### Modelli ensemble (sperimentale)
Il sistema include componenti per modelli statistici (Prophet), gradient boosting (GBM) e simulazioni di mercato, in fase di integrazione sperimentale. Questi moduli sono disponibili via gRPC ma non ancora calibrati per uso decisionale in produzione.

### Data Ingestion Pipeline
Aleph supporta l'importazione dati da sorgenti eterogenee: RSS/Atom feed, GitHub (issues, repo metadata), CSV/JSON file upload, sitemap XML, Google Sheets, email (IMAP). Ogni fonte ha un fetcher dedicato con validazione e sanitizzazione SSRF-safe.

### File System Watcher
Il watcher (fsnotify) monitora le directory di progetto per nuovi file e li importa automaticamente con debounce (500ms), abilitando flussi di lavoro basati su drop di file.

### Workspace adattivo
L'interfaccia è pensata come spazio di lavoro, non come dashboard. L'obiettivo non è mostrare dati: è aiutare a orientarsi tra informazioni, scenari e azioni possibili.

### Architettura resiliente
Il sistema è progettato per essere modulare e resistente agli errori. Circuit breaker, timeout configurabili, rate limiting, bulkhead pattern e graceful shutdown sono integrati.

---

## Architettura

```
Utente
  │
  ▼
Interfaccia Web
React · TypeScript · Vite · Tailwind CSS
  │
  ▼
Backend (Connect RPC + REST)
Go · h2c · SSE
  │
  ├── Persistenza dati
  │   PostgreSQL 16 (metadati relazionali)
  │   DuckDB (storage analitico + VSS)
  │
  ├── Intelligence layer (Python sidecar, gRPC)
  │   NLP euristico · Prophet · GBM · Simulazioni mercato
  │
  ├── Monitoraggio
  │   Prometheus :9090 · Grafana :3000 · Alertmanager :9093
  │
  └── Orchestrazione
      Docker · Docker Compose · 6 servizi
```

### Stack tecnologico

| Area | Tecnologia | Stato |
|---|---|---|
| Backend | Go 1.24, Connect RPC | produzione |
| Database | DuckDB + PostgreSQL 16 | produzione |
| VSS | DuckDB `list_cosine_similarity()` | produzione |
| NLP Sentiment | Python, dizionari euristici (IT/EN) | produzione |
| NLP Predizioni | Python, Prophet, GBM, simulazioni | sperimentale |
| Frontend | React 18, TypeScript 5, Vite, Tailwind CSS, Zustand | produzione |
| Auto-repair | Go (7 fix strategies) | produzione |
| Decision Intelligence | PAORA (Plan→Act→Observe→Reflect→Admit) | produzione |
| File Watcher | fsnotify con debounce | produzione |
| Auto-suggestion | Genesis (Suggester→Sandbox→VetoRegistry) | produzione |
| Security | CSP, CSRF, rate limiting, Argon2id, SSRF guard, sandbox | produzione |
| Monitoraggio | Prometheus :9090, Grafana :3000, Alertmanager :9093 | produzione |
| Contract testing | Go ↔ Python NLP gRPC (build tag) | produzione |
| Orchestrazione | Docker, Docker Compose (6 servizi) | produzione |

---

## Installazione

### Requisiti
- Git
- Docker e Docker Compose

Per lavorare direttamente sul codice: Go (backend), Node.js (frontend), Python (intelligence layer).

### Avvio con Docker Compose

```bash
git clone https://github.com/noffolo/aleph.git
cd aleph
cp .env.example .env
```

Configura le variabili nel file `.env` (in particolare `KEY_ENCRYPTION_KEY` è obbligatoria), poi:

```bash
docker compose up --build -d
```

L'interfaccia sarà disponibile su `http://localhost:5173`.

---

## Configurazione

La configurazione avviene tramite file `.env`. Il file `.env.example` contiene un esempio completo delle variabili disponibili, che includono chiavi API, modelli AI (Ollama), porte dei servizi, impostazioni CORS, rate limiting, database, logging e ambiente di esecuzione.

Non committare mai `.env` nel repository.

---

## Sicurezza

Aleph può gestire chiavi API, richieste verso modelli AI e dati potenzialmente sensibili.

- `KEY_ENCRYPTION_KEY` obbligatoria — chiavi API cifrate con AES-256-GCM
- Chiavi API hashate con Argon2id (legacy SHA-256 rilevato e migrato)
- CORS ristretto a origini esplicite
- CSP senza `unsafe-inline`
- Rate limiting per IP (X-Forwarded-For → X-Real-IP → RemoteAddr)
- Protezione SSRF con risoluzione DNS, re-validation redirect, blocco IP privati
- Sandbox di esecuzione con blocklist di package pericolosi (`os/exec`, `syscall`, `unsafe`)
- SQL injection: query parametrizzate + regex `validName()` su identificatori
- Audit logging per tutte le operazioni tool

Per segnalare una vulnerabilità, non aprire una issue pubblica. Consulta `SECURITY.md`.

---

## Uso per sviluppatori

### Backend
```bash
go mod download
go build ./...
go test -race -count=1 ./...
go vet ./...
```

### Frontend
```bash
cd frontend
npm install
npm run dev      # sviluppo (hot reload)
npm run build    # produzione
npx vitest run   # test unitari
npx tsc --noEmit # type check
```

### Docker
```bash
docker compose config          # valida YAML
docker compose up --build      # ricostruisce tutto
docker compose down            # ferma i servizi
```

### Test NLP (Python)
```bash
cd nlp
python3 -m pytest tests/ -v
```

### Struttura del repository

```
.
├── frontend/              # Interfaccia web React/TypeScript/Vite
│   ├── src/
│   │   ├── api/           # Connect RPC client
│   │   ├── components/    # UI componenti (25+ componenti)
│   │   ├── hooks/         # Custom hooks (SSE, paginazione, domain actions)
│   │   ├── store/         # Zustand slices (8 store)
│   │   └── schemas/       # Zod validazione (22 schemi)
│   └── e2e/               # Playwright E2E test
├── internal/              # Codice backend Go (40+ package)
│   ├── api/               # Handler Connect RPC + REST, SSE, protobuf, routes
│   ├── decision/          # PAORA engine
│   ├── ingestion/         # Data ingestion (7 sorgenti)
│   ├── memory/            # VSS memory store
│   ├── genesis/           # Auto-suggestion engine
│   ├── repair/            # Auto-repair (7 fix strategies)
│   ├── middleware/        # 8 middleware
│   ├── sandbox/           # Tool execution isolata
│   ├── mcp/               # MCP discovery
│   ├── storage/           # DuckDB + PostgreSQL
│   ├── workflow/          # Workflow engine
│   ├── llm/               # Provider LLM (Ollama, OpenAI, Anthropic)
│   ├── tools/             # 6 tool suite
│   └── ...
├── nlp/                   # Python NLP sidecar
├── migrations/            # 26 file migrazione (DuckDB + PostgreSQL)
├── api/proto/             # Definizioni protobuf sorgente
├── .github/workflows/     # CI + Security + Deploy
├── Dockerfile
├── docker-compose.yml     # 6 servizi
├── docs/                  # manuale-tecnico, ARCHITECTURE, API, CHANGELOG
├── README.md
└── LICENSE
```

---

## Stato del progetto

Aleph-v2 è in fase **produzione** (v2.0.0, Maggio 2026).

| Verifica | Stato |
|----------|-------|
| `go build ./...` | ✅ |
| `go test -race -count=1 ./...` | ✅ (40+ package) |
| `go vet ./...` | ✅ |
| `npx tsc --noEmit` | ✅ (0 errori produzione) |
| `npx vite build` | ✅ (< 3s, 22 chunk) |
| `npx vitest run` | ✅ |
| `npx playwright test` | ✅ (E2E) |
| `docker compose config` | ✅ |
| Gitleaks secrets scan | ✅ |
| Prometheus + Grafana | :9090 / :3000 |

---

## Licenza

Distribuito con licenza **GPL-3.0**. Vedi `LICENSE` per i dettagli.

---

*Aleph nasce per esplorare una domanda semplice ma ambiziosa: come trasformare il rumore informativo in comprensione utile. Non promette decisioni automatiche né previsioni infallibili. Il suo obiettivo è costruire uno spazio in cui dati, modelli e persone possano collaborare per leggere meglio la complessità.*
