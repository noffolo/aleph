# Aleph — Piano Ingestione Dati Politici Italiani (DRAFT)

> **For agentic workers:** Use superpowers:subagent-driven-development or superpowers:executing-plans.
> **Status:** IN REVIEW — sottoposto a Momus, Metis, Oracle

**Goal:** Dotare Aleph di source type per ingerire dati elettorali (Eligendo) e finanziamenti alla politica italiana (OpenSanctions, ANAC, ondata/liberiamoli-tutti, OPDM, codice-civico, Camera/Senato SPARQL), con dual-write raw/normalizzato e frontend dedicato.

**Architecture:** Due fasi indipendenti. Fase 1: `source_type="election"` → Eligendo API con rate limiting, fuzzy matching partiti, dual DuckDB write. Fase 2: 5 nuovi source type (`pep`, `public_contracts`, `party_funding`, `opdm`, `parliament`) per coprire l'ecosistema dei finanziamenti. Ogni source type segue il pattern: config JSON → fetcher dedicato → DuckDB write → frontend form.

**Tech Stack:** Go 1.26, DuckDB, testify, Jaro-Winkler (`xrash/smetrics`), SPARQL, FollowTheMoney JSON

---

## FASE 1: Dati Elettorali (Eligendo API)

### Task 1: Election helper types + ISTAT code lookup
**Files:** `internal/ingestion/sources/election.go`, `eligendo_codes.json`
- TE codes mapping, canonical party list (~50 partiti), ElectionConfig/ElectionResult structs
- `go:embed` per ISTAT code lookup
- Test: TE code mapping, ISTAT lookup

### Task 2: ElectionFetcher con rate limiting + API calls
**Files:** `internal/ingestion/sources/election.go`, `election_test.go`
- `ElectionFetcher` con 1 req/s rate limiter, headers Eligendo obbligatori
- `GetEntities()` → `getentiFI`, `GetScrutini()` → `scrutiniFI`
- Test: rate limiter waits ≥1s, TE code mapping

### Task 3: Fuzzy matching normalization engine
**Files:** `internal/ingestion/sources/election.go`, `election_test.go`
- `preprocessListName()`, `normalizePartyName()` con Jaro-Winkler
- Soglia 0.85, override manuale via `party_mapping` map
- Test: exact match, fuzzy match (FdI con suffisso), no match, manual override

### Task 4: runElection() in engine.go
**Files:** `internal/ingestion/engine.go`, `election_test.go` (integration)
- `case "election"` nello switch, `parseElectionConfig()`, `runElection()`
- Iterazione getenti → scrutini per ogni ente → dual write DuckDB
- Supporto date_filter via DateRangeConfig esistente
- Test: config parsing, election_type validation, integration con mock HTTP

### Task 5: Frontend — DataSourceForm election fields
**Files:** `DataSourceForm.tsx`, `DataSourceFormSlideOver.tsx`
- Dropdown election_type (6 tipi), dropdown level (comune/provincia/regione)
- Date range ereditato da datefilter esistente
- TypeScript compile check

---

## FASE 2: Finanziamenti & Dati Politici

### Task 6: source_type="pep" — OpenSanctions PEP data
**Files:** Create `internal/ingestion/sources/pep.go`, `pep_test.go`
- Scarica `opensanctions.org/datasets/it_deputies/` + `it_senate` → FtM JSON
- Parsing FollowTheMoney entities → DuckDB table `pep_entities`
- Schema: `id, name, country, birth_date, position, party, dataset, first_seen, last_seen`
- Rate limiting: 2 req/s (OpenSanctions è generoso)
- Frontend: dropdown dataset (deputati/senato/PEP), date range

### Task 7: source_type="public_contracts" — ANAC Open Data
**Files:** Create `internal/ingestion/sources/anac.go`, `anac_test.go`
- Scarica CSV da `dati.anticorruzione.it/opendata/` (CIG annuali, partecipanti, aggiudicatari)
- Anno come parametro config, scarica CSV ~150 MB/anno
- DuckDB table `public_contracts`: `cig, anno, importo, stazione_appaltante, aggiudicatario, partecipanti`
- Supporto incrementale: delta CSV mensili per aggiornamento
- Rate limiting: 5 req/s
- Frontend: dropdown anno, tipo dataset (CIG/partecipanti/aggiudicatari/subappalti)

### Task 8: source_type="party_funding" — ondata/liberiamoli-tutti
**Files:** Create `internal/ingestion/sources/funding.go`, `funding_test.go`
- Clona/pulla repo `ondata/liberiamoli-tutti` → importa `political_finance.csv`
- DuckDB table `party_funding`: `donation_amount, donation_year, recipient_party, donor_type, donor_name, source_name`
- Supporto aggiornamento: git pull + reimport
- Frontend: config vuoto (single source), date range

### Task 9: source_type="opdm" — Openpolis API
**Files:** Create `internal/ingestion/sources/opdm.go`, `opdm_test.go`
- REST API `service.opdm.openpolis.io` con API key configurable
- Endpoint: memberships, properties, persons, organizations
- DuckDB table `opdm_memberships`: `person_id, org_id, role, start_date, end_date`
- Rate limiting: 10k req/giorno (anonimo), illimitato (registrato)
- Frontend: dropdown entity_type, API key field, date range

### Task 10: source_type="parliament" — Camera/Senato SPARQL + Codice Civico
**Files:** Create `internal/ingestion/sources/parliament.go`, `parliament_test.go`
- SPARQL query a `dati.camera.it/sparql` e `dati.senato.it/sparql`
- Estrai: votazioni, presenza, gruppi parlamentari
- DuckDB table `parliament_votes`: `legislatura, data, titolo, esito, voto_deputato, gruppo`
- Integrazione API `codice-civico` per cross-reference contratti/politici
- Rate limiting: 5 req/s per endpoint
- Frontend: dropdown camera/senato/entrambi, legislatura, date range

### Task 11: DuckDB cross-reference views (bonus)
**Files:** Modifica `internal/ingestion/engine.go`
- View SQL predefinite: `politician_full_profile` (JOIN pep + opdm + parliament)
- `contract_party_link` (JOIN public_contracts + pep via nome/codice_fiscale)
- `funding_timeline` (party_funding aggregato per anno/partito)
- Eseguite automaticamente dopo ogni ingestion task

---

## Riepilogo

| # | Task | Source Type | Fonte | Formato |
|---|---|---|---|---|
| 1-5 | Election | `election` | Eligendo API | JSON |
| 6 | PEP | `pep` | OpenSanctions | FtM JSON |
| 7 | Appalti | `public_contracts` | ANAC Open Data | CSV |
| 8 | Finanziamenti | `party_funding` | ondata/liberiamoli-tutti | CSV |
| 9 | Openpolis | `opdm` | OPDM API | JSON |
| 10 | Parlamento | `parliament` | SPARQL + Codice Civico | RDF/JSON |
| 11 | Cross-ref | — | DuckDB SQL | — |

**Totale: 11 task, 6 nuovi source type**
