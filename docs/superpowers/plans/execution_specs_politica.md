# Aleph — Specs Ingestione Dati Politici Italiani

> **Linked plan:** `docs/superpowers/plans/execution_plan_politica.md`
> **Status:** SPECS — requisiti e contratti per il piano esecutivo

---

## 1. Requisiti Funzionali

### FR-01: Ingestione Dati Elettorali
**Priority:** P0 | **Source:** Eligendo API (`eleapi.interno.gov.it`)

- Il sistema deve ingerire i risultati elettorali di tutte le elezioni italiane dal 2000 ad oggi
- Livello di granularità: comune (con mapping ISTAT), con aggregazione opzionale a provincia/regione
- Tipi elezione supportati: politiche (Camera/Senato), europee, regionali, comunali, provinciali, referendum
- I dati raw delle liste (con nome esatto dall'API) devono essere salvati compressi su disco
- I dati normalizzati (con party canonical ID) devono essere scritti su DuckDB
- Il mapping partiti usa una tabella di alias configurabile (`party_aliases.json`), con override manuale prioritario

**Acceptance Criteria:**
- `POST /api/sources/ingest` con `source_type="election"` e `config_json={"election_type":"politiche","level":"comune","year":2022}` produce:
  - File `data/raw/election/2022-politiche-comune/getenti.json.gz`
  - File `data/raw/election/2022-politiche-comune/scrutini_<ISTAT>.json.gz` per ogni comune
  - Righe nella tabella DuckDB `election_results` con `party_canonical` popolato dove il match alias esiste
  - Watermark aggiornato in `ingestion_watermark`

### FR-02: Ingestione PEP (OpenSanctions)
**Priority:** P1 | **Source:** `opensanctions.org/datasets/it_deputies/`, `it_senate`, `it_peps`

- Scaricare dataset FollowTheMoney JSON da OpenSanctions
- Parsing entità FtM → `pep_entities` DuckDB table
- Campi: `id, name, country, birth_date, position, party, dataset, first_seen, last_seen`
- Rate limiting: 2 req/s
- Watermark: file hash/last-modified per rilevare aggiornamenti dataset

### FR-03: Ingestione Appalti Pubblici (ANAC)
**Priority:** P1 | **Source:** `dati.anticorruzione.it/opendata/`

- Scaricare CSV annuali CIG (Codice Identificativo Gara), ~150 MB/anno
- Encoding: ISO-8859-1 (ANAC storico) e UTF-8 (recente)
- Caricamento diretto DuckDB via `read_csv_auto` (no parsing Go-side)
- `public_contracts` table: `cig, anno, importo, stazione_appaltante, aggiudicatario, partecipanti`
- Dry-run mode: `--dry-run` flag che fa solo HEAD request per validare disponibilità
- Rate limiting: 5 req/s

### FR-04: Ingestione Finanziamenti Partiti
**Priority:** P1 | **Source:** `github.com/ondata/liberiamoli-tutti`

- Clone/pull repo Git, import `political_finance.csv`
- `party_funding` table: `donation_amount, donation_year, recipient_party, donor_type, donor_name, source_name`
- Watermark: git commit hash come cursor

### FR-05: Ingestione Openpolis (OPDM)
**Priority:** P2 | **Source:** `service.opdm.openpolis.io` REST API

- Endpoint: memberships, persons, organizations, properties
- API key configurabile nel config_json
- Pagination-aware (next cursor)
- Rate limiting: ~0.116 req/s (10k/giorno distribuito su 24h)
- `opdm_memberships` table: `person_id, org_id, role, start_date, end_date`

### FR-06: Ingestione Parlamento (SPARQL + Codice Civico)
**Priority:** P2 | **Source:** `dati.camera.it/sparql`, `dati.senato.it/sparql`

- SPARQL query per votazioni, presenza, gruppi parlamentari
- `parliament_votes` table: `legislatura, data, titolo, esito, voto_deputato, gruppo`
- Rate limiting: 5 req/s per endpoint SPARQL, backoff esponenziale su 429
- Integrazione opzionale API Codice Civico per cross-reference contratti/politici

### FR-07: Frontend
**Priority:** P1

- Dropdown `source_type` con optgroup "Politica Italiana": election, pep, public_contracts, party_funding, opdm, parliament
- Campi condizionali per ogni source type (election_type, livello, anno, dataset, API key, legislatura)
- Date range ereditato dal componente DateFilter esistente
- Dry-run checkbox opzionale

### FR-08: Health Monitoring
**Priority:** P1

- `GET /api/health/ingestion` restituisce stato di ogni source type:
  ```json
  {
    "sources": [
      {"source_name": "election", "last_run": "2026-05-23T10:00:00Z", "status": "healthy", "record_count": 1500},
      {"source_name": "pep", "last_run": "2026-05-20T08:00:00Z", "status": "stale"}
    ],
    "total_sources": 2,
    "timestamp": "2026-05-23T12:00:00Z"
  }
  ```
- Status: "healthy" (<7 giorni), "stale" (>7 giorni), "error" (ultima run fallita)

### FR-09: Cross-Reference Views
**Priority:** P2

Tre viste SQL DuckDB predefinite:
1. `v_politician_full_profile`: JOIN pep + opdm + parliament per profilo completo politico
2. `v_contract_party_link`: JOIN public_contracts + pep per collegare appalti a PEP
3. `v_funding_timeline`: Aggregato party_funding per anno/partito

Create automaticamente dopo ogni ingestion (CREATE OR REPLACE VIEW per idempotenza).

---

## 2. Requisiti Non-Funzionali

### NFR-01: Resilienza
- Ogni source type fallisce indipendentemente (error isolation)
- Partial failure: se un ente fallisce, i precedenti rimangono salvati
- Retry: 3 tentativi con exponential backoff (1s, 2s, 4s) su errori 5xx e 429
- Timeout HTTP: 30 secondi default, configurabile per source

### NFR-02: Performance
- ANAC 150 MB CSV: ingestione < 10 minuti (DuckDB read_csv_auto nativo)
- Election full ingest (tutti i comuni): < 8 ore a 1 req/s
- DuckDB write: esecuzione sequenziale (single-writer constraint). No ingestion parallele.

### NFR-03: Observability
- Structured logging (`log/slog`) per ogni operazione: `source_type`, `records_ingested`, `duration`, `error`
- Log level INFO per progress, WARN per unmapped data, ERROR per fallimenti
- Health endpoint con last_run per ogni source

### NFR-04: Testabilità
- Unit test con mock HTTP server (go-vcr per fixture recording)
- Dry-run mode per ogni source type
- Test data isolati: DuckDB in-memory per test

### NFR-05: Configurabilità
- API keys via config_json (mai hardcoded)
- Rate limit parametrizzabile per source
- Endpoint URL configurabili
- Party aliases via `configs/party_aliases.json` (editabile senza ricompilare)

---

## 3. Data Model

### 3.1 DuckDB Tables

```sql
-- FASE 1: Elezioni
CREATE TABLE election_results (
    id INTEGER PRIMARY KEY,
    election_type TEXT NOT NULL,     -- politiche, europee, regionali, comunali, provinciali, referendum
    level TEXT NOT NULL,             -- comune, provincia, regione
    year INTEGER NOT NULL,
    comune TEXT NOT NULL,
    comune_istat TEXT NOT NULL,      -- codice ISTAT 6 cifre
    lista TEXT NOT NULL,             -- nome lista dall'API Eligendo (raw)
    party_canonical TEXT,            -- party ID normalizzato (null se unmapped)
    voti INTEGER NOT NULL,
    percentuale REAL,
    seggi INTEGER,
    elettori INTEGER,
    votanti INTEGER,
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_election_type_year ON election_results(election_type, year);
CREATE INDEX idx_election_istat ON election_results(comune_istat);

-- FASE 2: PEP
CREATE TABLE pep_entities (
    id TEXT PRIMARY KEY,             -- OpenSanctions entity ID
    name TEXT NOT NULL,
    country TEXT,
    birth_date TEXT,
    position TEXT,
    party TEXT,
    dataset TEXT,                    -- it_deputies, it_senate, it_peps
    first_seen TEXT,
    last_seen TEXT,
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- FASE 2: Appalti Pubblici
CREATE TABLE public_contracts (
    cig TEXT PRIMARY KEY,            -- Codice Identificativo Gara
    anno INTEGER NOT NULL,
    importo REAL,
    stazione_appaltante TEXT,
    aggiudicatario TEXT,
    partecipanti TEXT,               -- JSON array di partecipanti
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_contracts_anno ON public_contracts(anno);

-- FASE 2: Finanziamenti Partiti
CREATE TABLE party_funding (
    id INTEGER PRIMARY KEY,
    donation_amount REAL NOT NULL,
    donation_year INTEGER NOT NULL,
    recipient_party TEXT NOT NULL,
    donor_type TEXT,                 -- Persona Fisica, Società, Associazione
    donor_name TEXT,
    source_name TEXT,                -- fonte del dato (es. "Bilancio Camera 2023")
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_funding_party_year ON party_funding(recipient_party, donation_year);

-- FASE 2: Openpolis Memberships
CREATE TABLE opdm_memberships (
    id INTEGER PRIMARY KEY,
    person_id TEXT NOT NULL,
    org_id TEXT NOT NULL,
    role TEXT,
    start_date TEXT,
    end_date TEXT,
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- FASE 2: Parlamento
CREATE TABLE parliament_votes (
    id INTEGER PRIMARY KEY,
    legislatura INTEGER NOT NULL,
    data TEXT NOT NULL,
    titolo TEXT,
    esito TEXT,                      -- APPROVATA, RESPINTA
    voto_deputato TEXT,              -- nome parlamentare
    gruppo TEXT,                     -- gruppo parlamentare
    chamber TEXT,                    -- camera, senato
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_votes_legislatura ON parliament_votes(legislatura, data);

-- INFRASTRUTTURA: Watermark
CREATE TABLE ingestion_watermark (
    source_name TEXT PRIMARY KEY,
    last_run TIMESTAMP NOT NULL,
    cursor TEXT DEFAULT '',          -- posizione/pagina/hash per incremental sync
    metadata TEXT DEFAULT ''         -- JSON con info aggiuntive (record count, etc.)
);

-- INFRASTRUTTURA: Schema Migrations
CREATE TABLE schema_migrations (
    version INT PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMP NOT NULL
);
```

### 3.2 Raw Data Storage (Filesystem)

```
data/raw/
├── election/
│   └── 2022-politiche-comune/
│       ├── getenti.json.gz
│       ├── scrutini_058091.json.gz   # Roma
│       └── scrutini_015146.json.gz   # Milano
├── pep/
│   ├── it_deputies.json.gz
│   └── it_senate.json.gz
├── public_contracts/
│   └── CIG_2024.csv
├── party_funding/
│   └── political_finance_2024.csv
├── opdm/
│   └── memberships_page_1.json.gz
└── parliament/
    └── votes_legislatura_19.json.gz
```

### 3.3 Party Aliases (configs/party_aliases.json)

```json
{
  "FRATELLI D'ITALIA": "fratelli-italia",
  "FRATELLI D'ITALIA - GIORGIA MELONI": "fratelli-italia",
  "FDI": "fratelli-italia",
  "PARTITO DEMOCRATICO": "partito-democratico",
  "PD": "partito-democratico",
  "MOVIMENTO 5 STELLE": "movimento-5-stelle",
  "M5S": "movimento-5-stelle",
  "LEGA": "lega",
  "LEGA PER SALVINI PREMIER": "lega",
  "FORZA ITALIA": "forza-italia",
  "FI": "forza-italia",
  "AZIONE": "azione",
  "ITALIA VIVA": "italia-viva",
  "SINISTRA ITALIANA": "sinistra-italiana",
  "ALLEANZA VERDI E SINISTRA": "verdi-sinistra",
  "+EUROPA": "piu-europa",
  "NOI MODERATI": "noi-moderati",
  "SUD CHIAMA NORD": "sud-chiama-nord",
  "UNIONE POPOLARE": "unione-popolare"
}
```

---

## 4. API Contracts

### 4.1 Ingestion Trigger

```
POST /api/sources/ingest
Content-Type: application/json

{
  "source_type": "election",
  "config_json": {
    "election_type": "politiche",
    "level": "comune",
    "year": 2022
  },
  "dry_run": false
}
```

**Response 202 Accepted:**
```json
{
  "task_id": "ing_abc123",
  "source_type": "election",
  "status": "started",
  "estimated_duration": "varies by source"
}
```

**Response 400 Bad Request:**
```json
{
  "error": "invalid election_type: fantasia",
  "valid_values": ["politiche", "europee", "regionali", "comunali", "provinciali", "referendum"]
}
```

### 4.2 Health Endpoint

```
GET /api/health/ingestion
```

**Response 200:**
```json
{
  "sources": [
    {
      "source_name": "election",
      "last_run": "2026-05-23T10:00:00Z",
      "status": "healthy",
      "cursor": "",
      "record_count": 1500
    }
  ],
  "total_sources": 1,
  "timestamp": "2026-05-23T12:00:00Z"
}
```

### 4.3 Source Type Listing

```
GET /api/sources/types
```

**Response 200:**
```json
{
  "source_types": [
    {"name": "election", "category": "Politica Italiana", "description": "Risultati elettorali via Eligendo API"},
    {"name": "pep", "category": "Politica Italiana", "description": "PEP data via OpenSanctions"},
    ...
  ]
}
```

### 4.4 External API Contracts

**Eligendo API:**
- `GET {base}/getentiFI?te={TE_CODE}&liv={livello}` → enti JSON
- `GET {base}/scrutiniFI?te={TE_CODE}&cod={ISTAT}` → risultati JSON
- Rate limit: 1 req/s (imposto da Aleph, non documentato dall'API)

**OpenSanctions:**
- `GET {base}/entities/{dataset}` → FtM JSON array
- Rate limit: 2 req/s

**ANAC Open Data:**
- `GET {base}/CIG_{anno}.csv` → CSV ISO-8859-1
- Rate limit: 5 req/s

**OPDM Openpolis:**
- `GET {base}/memberships?page={cursor}` → JSON paginato
- Header: `Authorization: Bearer {api_key}`
- Rate limit: 10k req/giorno

**Camera SPARQL:**
- `POST {base}?query={SPARQL}` → JSON results.bindings
- Rate limit: 5 req/s, backoff su 429

---

## 5. Error Handling Matrix

| Scenario | Response | Retry? | Fallback |
|---|---|---|---|
| API timeout (>30s) | Log WARN, skip entity | Sì (3x) | Prossimo ente |
| API 429 Rate Limited | Log WARN, backoff | Sì (exponential) | RateLimiter.Wait() |
| API 5xx Server Error | Log ERROR, skip entity | Sì (3x) | Prossimo ente |
| API 404 Not Found | Log ERROR, skip | No | Prossimo ente |
| CSV malformed row | Log WARN, skip row | No | Continua parsing |
| DuckDB write locked | Queue, retry 3x | Sì | Sequenziale |
| Party unmapped | Log INFO, canonical=NULL | No | Salva con NULL |
| Disk full (raw save) | Log ERROR, continue | No | Salta raw, scrivi DB |

---

## 6. Testing Strategy

### Unit Tests
- Ogni handler: test con mock HTTP server + DuckDB in-memory
- PartyMapper: exact match, alias match, override priority, no-match
- RateLimiter: burst, rate enforcement, timing assertions
- Watermark: CRUD lifecycle, ListAll
- Registry: Register, Create, List, ErrSourceTypeNotFound

### Integration Tests
- `runElection()`: pipeline completa con mock server multi-endpoint
- CSV ingestion: encoding ISO-8859-1 e UTF-8
- SPARQL parsing: sample RDF/JSON responses
- Cross-reference views: CREATE OR REPLACE idempotenza

### Dry-Run Tests
- ANAC: HEAD request verifica disponibilità CSV
- OPDM: validate API key con richiesta minima
- SPARQL: sintassi query valida

### Fixture Recording (go-vcr)
- Registrare risposte API reali per test deterministici
- Fixture files in `testdata/fixtures/`

---

## 7. Review Feedback Traceability

| Issue | Reviewer | Severity | Resolution | Task |
|---|---|---|---|---|
| No watermark/incremental strategy | Oracle | 🔴 Critical | Watermark table + migrations | 0.1 |
| No testing strategy (go-vcr) | Oracle | 🔴 Critical | go-vcr fixtures in ogni task | 2-10 |
| No observability/health endpoint | Oracle | 🔴 Critical | HealthHandler + slog | 5 |
| DuckDB single-writer concurrency | Oracle | 🟡 High | Sequential execution documented | NFR-02 |
| Dual-write DuckDB bloat | Oracle | 🟡 Medium | Raw su filesystem, solo normalized su DB | 3,6-10 |
| No dry-run mode | Oracle | 🟡 Medium | Dry-run flag in ANAC + OPDM + SPARQL | 7,9,10 |
| Jaro-Winkler overkill | Oracle | 🟢 Low | Alias table replaces fuzzy matching | 1 |
| Switch-statement coupling | Oracle | 🟢 Low | Registry pattern with GlobalRegistry | 0.3 |
| CSV encoding ISO-8859-1 | Metis | ⚠️ Warning | DuckDB read_csv_auto con encoding param | 7 |
| Frontend Phase 2 missing | Metis | ⚠️ Warning | Full frontend support + conditional fields | 4 |
| No QA scenarios | Momus | ❌ REJECT | VERIFICA section in ogni task | All |
