# Aleph-v2 Production Gate Plan (v2 — Integrato con Review)

**Data**: 2026-04-29
**Stato**: FINALE — integrate review di Momus (Plan Critic), Metis (Plan Consultant), Oracle (Architecture)
**Mandato**: Zero-Todo — nessun placeholder, nessun debito tecnico, nessun bug prima della chiusura Phase 2

---

## Principio Guida

Prima di riparare QUALSIASI cosa: **testare TUTTO**. Ogni funzione, sotto-funzione, variabile. Mappare i problemi. Poi riparare in ordine di dipendenza.

**Ogni wave termina con**:
```
go build ./... && go test -race -count=1 ./... && go vet ./... && npx vite build && vitest run && npx tsc --noEmit
git tag checkpoint/wave-{W}
```

---

## Build State Attuale

| Verifica | Stato |
|----------|-------|
| go build ./... | ✅ |
| go test -race -count=1 ./... | ✅ (tutto cached) |
| go vet ./... | ⚠️ (PEG struct tags pre-esistenti in dsl/ast.go) |
| npx vite build | ✅ (3.31s) |
| npx tsc --noEmit | ❌ (yjs mancante) |
| vitest run | ❌ (3 failures in 2 file) |
| docker compose up (full stack) | ❌ (NLP healthcheck fallisce sempre) |

---

## Tier A — Broken/Missing (P1)

1. **Genesis Suggester** (`internal/genesis/suggester.go:32-34`) — `Analyze()` sempre vuoto
2. **MemoryStore** (`internal/memory/memory.go:9, 28-29`) — stub senza metodi di storage
3. **File Watcher** (`service/watcher/watcher.go:39-41`) — rileva file ma non triggera ingestion
4. **fixPerformance** (`synthesis/repair.go:806-808`) — no-op

## Tier B — Parziale/Stubbato (10+)

- String-replace codegen in `repair.go` (fixTimeout può corrompere, fixCaching inietta TODO)
- GNN training risultato scartato (`_ = p.trainer.Train`)
- Embedding failures silenziosamente incomplete
- Migration failures non propagate (solo log)
- 3 test frontend rotti (ToolForm i18n, useAppActions mock)
- NLP 15+ print() invece di logging
- NLP healthcheck porta 50051 invece di 8001 — **BUG CONFERMATO**
- Mock StreamPredictions ritorna nil,nil
- yjs mancante blocca npx tsc --noEmit
- CORS middleware non al path documentato
- Email credential leak in tmpdir (script Python)
- **LLM Provider è nil** (`app.go:226`: `Provider: nil`) — blocca W2 (Ontology), W3 (ProbeRunner), W4 (Genesis)

---

## Security — 25 Findings

### 6 CRITICAL
| # | Vettore | File | Fix |
|---|---------|------|-----|
| 1 | SQL injection: query.go information_schema | `internal/handler/query.go` | Parameterize |
| 2 | SQL injection: ingestion CREATE TABLE/VIEW | `internal/ingestion/engine.go` | Whitelist validation |
| 3 | SQL injection: GetDataStats/GetDataLineage | `internal/handler/query.go` | Parameterize |
| 4 | SQL injection: DSL compiler WHERE | `internal/dsl/compiler.go` | Parameterize |
| 5 | Code execution: runDynamic blockedImports | `internal/ingestion/engine.go` | Block unsafe/reflect/os/io/crypto/encoding |
| 6 | API key in sessionStorage | `frontend/src/store/authSlice.ts` | httpOnly cookie |

### 8 HIGH
- SSE e swagger.json endpoint non autenticati
- ExecSandbox senza Docker isolation
- Path traversal: SanitizeProjectID mancante a call site
- Condizionale encryption: plaintext quando KEY_ENCRYPTION_KEY non settato
- Email credential leak in tmpdir (script Python)
- SHA-256 hashing senza sale
- **SSRF fail-open** (engine.go Transport vs blockSSRF)
- **Dual SSRF mechanism** (due implementazioni, una fail-open)

### 8 MEDIUM
- DNS rebinding
- Rate limiting per-process (no X-Forwarded-For, no Redis)
- CSP unsafe-inline
- Notification webhook secret in payload
- Auth bypass via substring
- No CSRF
- NLP healthcheck port mismatch (50051 vs 8001)
- Ollama non presente in docker-compose.yml

---

## Modifiche Rispetto a v1 (Basate su Review)

| Tema | Review Source | Azione |
|------|--------------|--------|
| Phase 0 scope illimitato | Momus | Aggiunto stop rule: top 60 funzioni o 4h |
| LLM Provider nil | Metis, Oracle | NUOVA W1.5: Wire LLM Provider |
| Ollama service | Oracle | NUOVO pre-W4: aggiungere a docker-compose |
| W2 negotiation sottostimato 4x | Momus, Metis, Oracle | Split W2 in 3 sub-wave (W2A/B/C) |
| W2-06 non è implementazione | Momus, Oracle | Spostato in Phase 0 come analysis |
| W3 prima di W2 | Metis, Oracle | W3-01/04 in parallelo con W2 |
| MemoryStore: VSS da verificare | Momus, Metis, Oracle | PRE-CHECK: verificare array_cosine_similarity |
| Genesis: usage tracking mancante | Metis, Oracle | Nuovo task pre-W4-01 |
| SSRF dedup + W1 non W8 | Momus, Oracle | SSRF moved to W1 |
| Stime realistiche | Momus (+45% realistic) | Ricalcolate con buffer |
| Rollback checkpoints | Momus | git tag dopo ogni wave |
| Frontend view smoke test | Momus | Aggiunto a Phase 0 |
| CSP audit pre-requisito | Momus | W8: audit prima di removare unsafe-inline |
| W5+W6 merge | Metis | Unite in "Codegen + NLP Polish" |
| MemoryStore 3 sub-task | Metis | Schema, VSS, fallback, integration test |

---

## Piano Esecutivo — 10 Waves, ~85-100h

---

### Phase 0 — Inventory + Test Baseline (4-6h)
*PRIMA DI TOCCARE CODICE. Mappare TUTTI i problemi esistenti. Scope limitato.*

| ID | Cosa | Output | Scope Rule |
|----|------|--------|------------|
| P0-01 | Inventory funzioni Go: top 60 per complessità ciclomatica + API esportate + handlers HTTP | `docs/bug-registry.md` | STOP a 60 funzioni o 4h |
| P0-02 | Frontend: ogni view lazy-loaded rende? loading/empty/error states? as any count? | Report frontend | 6 React.lazy views |
| P0-03 | NLP Python: print/log? error handling? test coverage? | Report NLP | 4 file principali |
| P0-04 | Build pipeline: docker compose up full stack funziona? | Test E2E smoke | 1 run |
| P0-05 | **Post-ontology value flow analysis** (ex W2-06): cosa succede dopo che l'ontologia è definita? Mappare 5 pipeline chiave. | `docs/ontology-value-flow.md` | Max 1h, 5 entry points |
| P0-06 | **DuckDB vector capability check**: verificare `array_cosine_similarity()` disponibile | Report tecnico | 30min |
| P0-07 | **NLP healthcheck port BUG**: verificare incrociata che 50051→8001 sia l'unico fix | BUG CONFIRMED | Già verificato |

---

### Wave 1 — Fix Infrastructure + Security Criticals (10-12h)
*Test funzionanti + security critiche. Tutti i P0 fixati.*

| ID | Cosa | Dove | Stima |
|----|------|------|-------|
| W1-01 | Fix NLP healthcheck port 50051→8001 | `nlp/Dockerfile:21`, `docker-compose.yml:42` | 5min |
| W1-02 | Fix vitest failures (3 tests, i18n IT) | `frontend/src/` test files | 1h |
| W1-03 | Fix TS typecheck — yjs come devDep | `frontend/package.json` | 30min |
| W1-04 | SQL injection — parameterize 5 vettori (con whitelist per DDL) | `query.go`, `engine.go`, `compiler.go` | 4h |
| W1-05 | Code execution — bloccare unsafe/reflect/os/io/crypto/encoding | `engine.go blockedImports` | 1h |
| W1-06 | API key — sessionStorage → httpOnly cookie | `frontend/src/store/authSlice.ts` | 2h |
| W1-07 | SHA-256 → argon2 con sale | `internal/handler/auth.go` | 1.5h |
| W1-08 | KEY_ENCRYPTION_KEY enforced in config validation | `internal/config/config.go` | 30min |
| W1-09 | **SSRF fail-closed + dedup** — unificare due meccanismi in `internal/ssrf/validator.go` | `engine.go Transport` + `blockSSRF` | 2h |
| W1-10 | **Email credential leak** — fix tmpdir Python script | `ingestion/engine.go runEmailFetch` | 1h |

---

### Wave 1.5 — LLM Wiring + Infrastructure (4-5h) [NUOVA]
*Prerequisito per W2, W3, W4. LLM Provider attualmente nil.*

| ID | Cosa | Dove | Stima |
|----|------|------|-------|
| W1.5-01 | Wire `llm.Provider` non-nil a DecisionEngine e handlers | `internal/app/app.go:226` | 1.5h |
| W1.5-02 | **Ollama service** in docker-compose.yml (model download + healthcheck) | `docker-compose.yml` | 30min |
| W1.5-03 | DuckDB VSS capability test + fallback strategy document | `internal/storage/duckdb.go` | 1h |
| W1.5-04 | **Usage tracking subsystem**: middleware che intercetta tool calls → DuckDB | `internal/service/tracker/` | 2h |

---

### Wave 2A — Ontology: Types + API Contract (4-5h) [SPLITTO DA W2]
*Prima di toccare codice ontology: definire tipi e protocollo.*

| ID | Cosa | Dove | Stima |
|----|------|------|-------|
| W2A-01 | Definire `OntologySuggestion`, `Relationship`, `NegotiationState` types | `internal/dsl/ontology_types.go` | 1.5h |
| W2A-02 | Protobuf per `ProposeOntologyDiff`, `AcceptDiff`, `RejectDiff`, `ListVersions` | `api/ontology.proto` | 1.5h |
| W2A-03 | DB schema per ontology versioning (DuckDB o filesystem versionato) | `internal/repository/ontology.go` | 1.5h |

---

### Wave 2B — Ontology: LLM Emerge + Relationship Detection (7-8h)
*Dipende da W1.5 (LLM wired). Prompt convergence con 3 test case.*

| ID | Cosa | Dettaglio | Stima |
|----|------|-----------|-------|
| W2B-01 | **LLM Emerge** — prompt: descrivi sorgente → oggetti + relazioni. 3 test case per convergenza. | `project.go` | 4h |
| W2B-02 | **Relationship detection** — FK inference, name matching, content overlap | `project.go` | 3h |
| W2B-03 | **DSL compiler fix** — read_parquet() → DuckDB views | `dsl/compiler.go` | 1.5h |

---

### Wave 2C — Ontology: Negotiation + Frontend (8-10h)
*Dipende da W2A (types, proto, DB schema).*

| ID | Cosa | Dettaglio | Stima |
|----|------|-----------|-------|
| W2C-01 | **Negotiation API backend** — accept/reject/modify handlers + version storage | `internal/handler/ontology.go` | 3h |
| W2C-02 | **Ontology frontend** — diff viewer, accept/reject UI, version picker | `OntologyView.tsx` | 5h |
| W2C-03 | Integration test: emerge → propose → accept → verify view | E2E test | 1.5h |

---

### Wave 3 — New Ingestion Methods (18-22h)
*Parallelo con W2. W3-01/04 iniziano insieme a W2.*

| ID | Cosa | Stima | Note |
|----|------|-------|------|
| W3-01 | **GitHub repos** — API auth + rate limit + pagination + schema (issues, PRs, commits) | 8h | Non 4h: auth, rate limiting, pagination |
| W3-02 | **Sitemap XML** — parse → crawl → ingest | 2h | |
| W3-03 | **ProbeRunner reale** — LLM-based source probing + auto-classify + ProbeResult interface | 4h | ProbeResult da rendere interface |
| W3-04 | **Generic JSON/API** — REST con paginazione automatica | 3h | |
| W3-05 | **Google Sheets** — API → DuckDB | 3h | Se time permits |
| W3-06 | **Rate limiting + chunking** per HTTP fetches + worker pool per enrichment | 2h | |

---

### Wave 4 — Critical Stubs (12-15h)
*MemoryStore prima di Genesis Suggester (dipendenza dati).*

| ID | Cosa | Dove | Dettaglio | Stima |
|----|------|------|-----------|-------|
| W4-01 | **MemoryStore DuckDB schema** — table, Insert(), Search() con VSS, Delete() | `memory/memory.go` | Schema + VSS | 3h |
| W4-02 | **MemoryStore fallback** — cosine similarity senza VSS (SQL manuale) | `memory/memory.go` | Fallback | 2h |
| W4-03 | **MemoryStore integration test** — Insert+Search+Delete con DuckDB reale + wire in app | `memory/memory_test.go` + `app.go` | Test | 2h |
| W4-04 | **File Watcher** — detection → trigger ingestion reale | `service/watcher/watcher.go` | | 2h |
| W4-05 | **Genesis Suggester Analyze()** — pattern d'uso → 3 tipi concreti di suggerimento (tool automation, skill suggestion, cross-user pattern). Dipende da W1.5-04 (usage tracking). | `suggester.go` | 3 tipi boundati | 4h |
| W4-06 | **fixPerformance** — pattern detection nei dati d'uso | `synthesis/repair.go:806` | | 2h |

---

### Wave 5 — Codegen Repair + NLP Polish (8-10h) [MERGE W5+W6]

| ID | Cosa | Dove | Stima |
|----|------|------|-------|
| W5-01 | **fixTimeout/fixCaching** — fixare i 2 bug specifici del string-replace (NON riscrivere AST) | `synthesis/repair.go` | 1.5h |
| W5-02 | **GNN training** — usare risultato, non scartarlo. Definire prima cosa modella. | `synthesis/repair.go` | 1.5h |
| W5-03 | **Embedding failures** — error propagation reale | `memory/embed.go` | 1h |
| W5-04 | **Migration errors** — propagate, non solo log | `internal/repository/metadata.go` | 1h |
| W5-05 | **NLP print() → logging** (17 calls) + random seed + gRPC channel TLS | `nlp/main.py` | 3h |
| W5-06 | **Dockerfile slim** — remove gcc/g++ (-300MB) + ONNX git-lfs | `nlp/Dockerfile` | 1h |

---

### Wave 6 — Config Hardening + CI/CD (5-6h)

| ID | Cosa | Stima |
|----|------|-------|
| W6-01 | **.env.example** — KEY_ENCRYPTION_KEY placeholder | 10min |
| W6-02 | **Hardcoded creds** — remove postgres da Dockerfile:48 | 30min |
| W6-03 | **CORS middleware** — fix path e documentazione | 30min |
| W6-04 | **nginx** — gRPC-Web headers | 1h |
| W6-05 | **CI** — docker-push secrets, test -race timeout, job chain fix | 2h |
| W6-06 | **.dockerignore** — .git, node_modules, __pycache__, ONNX, .env | 30min |

---

### Wave 7 — Security Medium + Release Gate (10-12h)

| ID | Cosa | Dove | Stima | Note |
|----|------|------|-------|------|
| W7-01 | **CSP policy audit** — censire inline styles/script → policy | `frontend/` | 2h | Audit PRIMA di removare unsafe-inline |
| W7-02 | **CSP unsafe-inline removal** — apply policy, test UI non rotta | `frontend/index.html`, nginx | 2h | |
| W7-03 | **Rate limiting** — X-Forwarded-For + Redis-backed | `internal/middleware/ratelimit.go` | 3h | |
| W7-04 | **CSRF** — token validation | `internal/middleware/csrf.go` | 2h | |
| W7-05 | **SSE auth** — proteggere endpoint | `internal/handler/events.go` | 1h | |
| W7-06 | **Release gate** — checklist finale: go build, go test -race, go vet, tsc --noEmit, vite build, vitest, docker compose up, CSP scan, SQL injection scan | `docs/release-checklist.md` | 1h | |

---

## Decisioni di Design — Confermate dalle Review

1. **Genesis** — resta auto-improvement engine. Architettura Suggester → Sandbox → Veto corretta. Manca: persistence (in-memory ora), deployment step, usage tracking. Aggiunto W1.5-04 e W4-05.

2. **MemoryStore** — DuckDB FLOAT[] arrays. Oracle conferma: viable fino a ~500K records (100K: 30-50ms query). Oltre: serve vector index. Aggiunto pre-check VSS (P0-06) e fallback (W4-02).

3. **NLP healthcheck** — BUG CONFERMATO: porta 50051 vs 8001. Fix in W1-01.

4. **Ollama** — MANCANTE da docker-compose. Aggiunto W1.5-02. Senza, MemoryStore non può generare embeddings.

5. **LLM Provider** — nil in app.go:226. Aggiunto W1.5-01. Blocca W2, W3, W4.

6. **W2 ordering** — split in W2A (types), W2B (LLM backend), W2C (frontend+negotiation). W3-01/04 parallelo con W2.

7. **Stime** — ricalcolate: ~85-100h totali (da 55-60h). Realistico.

8. **Rollback** — git tag checkpoint/wave-N dopo ogni wave. Non negoziabile.

---

## Allegati

- `docs/bug-registry.md` — Bug registry (da Phase 0)
- `docs/ontology-value-flow.md` — Post-ontology value flow analysis (da Phase 0)
- `docs/architecture/data-model.md` — DuckDB/PostgreSQL boundary document (da P0-06)
- Security audit: 25 findings, 10 in W1, 4 in W7
