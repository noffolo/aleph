# Piano di completamento produzione — Revisione v2

**Basata su review congiunte**: Metis (pre-planning), Oracle (architettura), Momus (plan critique)
**Data**: 30 Aprile 2026
**Stima sforzo**: 22-30 giorni (3 tracce parallele, non 5 onde sequenziali)

---

## Situazione attuale (verificata)

Tutti i build passano: `go build ./...` ✅ | `npx tsc --noEmit` ✅ | `npx vite build` ✅ | `go test -race -count=1 ./...` ✅
Il README dichiara "produzione" ma la realtà è eterogenea: **~45% delle capability sono production-grade**, il resto ha implementazione parziale o stub.

### Classificazione 4-livelli (da applicare a OGNI capability README)

| Livello | Definizione |
|---------|-------------|
| **Stub** | Firma/funzione esiste ma restituisce placeholder o valore fisso |
| **Parziale** | Logica implementata ma mancano edge case, error handling, o test |
| **Funzionale** | Funziona nei casi principali, ha test ma non copre tutti i failure mode |
| **Produzione** | Production-grade: testata, monitorata, documentata, resilient |

---

## Principi di esecuzione

1. **Honestà prima del codice**: Ogni capability README deve avere classificazione onesta (Stub/Parziale/Funzionale/Produzione) PRIMA di iniziare qualsiasi implementazione.
2. **Zero regressioni verificabile**: Non è uno slogan. Ogni task include prima/dopo test comparison.
3. **Quality gates bloccanti per ogni traccia**: Non solo alla fine. Ogni deliverable ha gate specifici.
4. **Capacità reali, non vetrine**: Meglio deprecare onestamente uno stub che lasciarlo come "completato" cosmeticamente.
5. **Niente nuove feature finché i gap README non sono chiusi**.

---

## Scope — Aree coperte (con gap reali identificati)

### A) Core platform (Backend Go + Connect RPC)

| Capability | Stato reale | Gap |
|------------|-------------|-----|
| PAORA decision cycle | Parziale | Reflect è pass-through (`return plan, nil`). Observer è keyword-only senza LLM fallback. Admit è threshold check semplice. GNN adapter deferred (nessun modello addestrato). |
| Ingestione multi-fonte | Funzionale | 6 source types implementati. `enrichPredictiveMetadata` inserisce 0.0 placeholder. SQL injection in `fmt.Sprintf` per table name (9 vector in `memory/store.go`). RSS è generic HTTP fetch, non parser dedicato. |
| Auto-repair engine | Parziale | 11+ strategie implementate MA `executeRegenerate` è placeholder (non invoca LLM). `fixPerformance` no-op. `fixTimeout`/`fixCaching` sono string-replace che possono corrompere codice. |
| GENESIS suggestion engine | Stub | `Suggester.Analyze()` restituisce stub suggestions. Sandbox è regex-matching, non container/gVisor isolation. 73 linee totali. |
| DSL compiler / tool stubs | Parziale | `compiler_tool.go` genera template con `# TODO: implement` e placeholders zero-value. |
| Event traceability PAORA | Stub | Non esiste audit trail per i cicli decisionali. |
| MCP discovery | Produzione | DiscoveryEngine + SSRF + health loop. ✅ |
| 8 middleware layers | Produzione | Auth (API-key), audit, CORS, CSRF, rate-limit, timeout, bulkhead, security headers. ✅ |

### B) Intelligence layer (Python NLP sidecar)

| Capability | Stato reale | Gap |
|------------|-------------|-----|
| Sentiment analysis | Stub | 30 parole hardcoded in word-counting heuristic. NON è ONNX/PyTorch. |
| PredictiveEnsemble (Prophet+GBM) | Parziale | Modelli Prophet+GBM funzionano MA `generate_ensemble_forecast` usa `simulator.py` con geometric Brownian motion — demo/simulation, non produzione. |
| Go↔Python gRPC contract | Parziale | Adapter Go chiama 1 RPC (`AnalyzeSentiment`). Python serve 4 RPC. Streaming (`StreamPredictions`) non ha consumer Go. |
| Confidence calibration | Stub | Output probabilistici non calibrati. Sentiment word-counting restituisce score arbitrari. |

### C) Data layer (DuckDB + PostgreSQL)

| Capability | Stato reale | Gap |
|------------|-------------|-----|
| DuckDB storage | Produzione | TX isolation, semaphore concurrency, 6 migrazioni, VSS con `array_cosine_similarity`. ✅ |
| PostgreSQL metadata | Funzionale | 4 migrazioni (con numbering gap: manca 000002). Nessuna data retention/vacuum/lifecycle. |
| Cross-engine consistency | Stub | Nessun transaction coordinator: se Postgres fallisce dopo DuckDB insert, dati orfani senza rollback. |

### D) Frontend (React/TS/Vite/Tailwind)

| Capability | Stato reale | Gap |
|------------|-------------|-----|
| 15 views con dati reali | Produzione | ✅ E2E integration, slide-over panels, SSE, streaming. 6 Zustand slices. |
| Re-render performance | Parziale | ~60 campi store causano 6+ re-render per keystroke in 7 componenti. No React.memo/useMemo/useCallback. |
| Scenario comparison UI | Stub | Manca: interfaccia per confrontare ipotesi side-by-side. |
| App.tsx monolithic | Parziale | ~2030 righe con SlideOverContent switch. 33 errori tsc pre-esistenti. |
| TypeScript hardening | Parziale | ~69 `any` rimanenti (68 eliminati in W5-10). |

### E) Security & Compliance

| Capability | Stato reale | Gap |
|------------|-------------|-----|
| Auth middleware | Parziale | API-key-only. Nessun RBAC, JWT, session management, OAuth. 82 righe totali. |
| Secret management | Parziale | AES-256-GCM in `internal/crypto/` ✅ MA API key in localStorage plaintext, SSE key in query param. |
| SSRF validation | Produzione | ✅ Implementato con test. |
| SQL injection protection | Stub | 9 `fmt.Sprintf` vector in `memory/store.go`. `validateSQLName` bypassabile. |
| SAST/DAST in CI | Stub | Nessuno. Solo gitleaks (secrets scan). |

### F) Observability & Operability

| Capability | Stato reale | Gap |
|------------|-------------|-----|
| OpenTelemetry traces | Produzione | ✅ OTLP gRPC exporter, correlation IDs, middleware wrapper. |
| Prometheus metrics | Stub | Nessun `/metrics` endpoint, nessun Prometheus registry. |
| Grafana dashboards | Stub | Non esiste. Non in docker-compose. |
| Alerting | Stub | Nessun Alertmanager, nessuna SLO burn-rate rule. |
| Structured logging | Parziale | `slog` usato ma senza correlation ID propagation nei log. |

### G) CI/CD & Release

| Capability | Stato reale | Gap |
|------------|-------------|-----|
| GitHub Actions CI | Produzione | ✅ Go build/test/vet + Frontend tsc/vitest/build. |
| Security scan | Parziale | gitleaks ✅ ma mancano govulncheck, npm audit, Trivy. |
| Contract/integration gate | Stub | Nessun test cross-servizio (Go↔Python). |
| E2E Playwright in CI | Stub | Config esiste ma non integrato nel workflow. |
| Deploy strategy | Parziale | Tag-triggered Docker push ✅ ma nessun blue/green o health gate. |

---

## Piano esecutivo — 3 Tracce Parallele

Invece delle 5 onde sequenziali del piano originale, strutturiamo in 3 tracce indipendenti che massimizzano il parallelismo:

```
Settimana 1          Settimana 2          Settimana 3          Settimana 4
──────────────────────────────────────────────────────────────────────────
TRACK A: Discovery + Backend/NLP Redesign
W1 (2-3gg) ───── W2 (10-15gg) ─────────────────────────────────
  │                 │
  │                 ├─ PAORA Reflect redesign
  │                 ├─ GENESIS Suggester/Sandbox reali
  │                 ├─ Sentiment NLP (ONNX reale o deprecazione onesta)
  │                 ├─ Repair engine AST-aware upgrade
  │                 ├─ SQL injection fix (bloccante)
  │                 └─ gRPC contract testing
  │
TRACK B: Monitoring + Frontend (parallel)
                  W3-Mon (3-5gg) ──── W3-UX (2-3gg)
                    │                   │
                    ├─ Prometheus+Grafana ├─ Scenario comparison UI
                    ├─ /metrics endpoint  ├─ WCAG 2.1 AA audit
                    └─ Alertmanager       └─ Lighthouse ≥90
  │
TRACK C: Security + Resilience (parallel)
                  W4-Sec (3-5gg) ──── W5-Rel (2-3gg)
                    │                   │
                    ├─ RBAC auth model   ├─ Contract test CI gate
                    ├─ SAST/DAST in CI   ├─ E2E Playwright in CI
                    ├─ Threat model      ├─ Load testing
                    └─ Chaos drills      └─ Release tag + post-deploy verify
```

### Track A — Discovery → Backend + NLP Redesign

#### W1 — Discovery tecnico onesto (2-3 giorni)

**Deliverable**: Matrice capability README classificata 4-livelli (Stub/Parziale/Funzionale/Produzione)
**Vincolo**: Ogni capability DEVE avere classificazione verificabile con comando shell

**Attività**:
- Code walk sistematico di ogni `internal/*` package:
  - `internal/decision/` — PAORA Reflect è stub? Confermare con grep `return plan`
  - `internal/genesis/` — Suggester.Analyze cosa restituisce? Test contro output atteso
  - `nlp/` — Sentiment è word-counting o ONNX? Verificare vocab size e assenza modello
  - `internal/repair/` — Ogni strategia ha test? Quali sono no-op?
  - `internal/ingestion/` — Trovare tutti i `fmt.Sprintf` con input utente
  - `internal/memory/store.go` — Verificare 9 vector SQL injection
- Mappatura endpoint: endpoint effettivi vs. READMA capability dichiarate
- Dipendenze cross-servizio: Go backend ↔ Python NLP ↔ Frontend

**Exit criteria**:
- Matrice capability README completa con classificazione e comando di verifica per ogni entry
- Backlog prioritizzato per Track A, B, C con stime individuali

---

#### W2 — Backend + NLP gap closure (10-15 giorni)

**Attenzione**: I seguenti NON sono "gap da chiudere" ma **redesign architetturali**. Trattarli come polish incrementale produrrà codice che sembra migliore ma non funziona fondamentalmente meglio.

##### W2a — PAORA Reflect redesign (2-3gg)
- **Stato**: `return plan, nil` pass-through
- **Obiettivo**: Reflect che analizza il piano, valuta scostamenti tra risultato atteso e osservato, genera riflessione strutturata
- **Cosa NON fare**: Aggiungere logging al pass-through. Serve nuova implementazione.
- **Test**: `go test -race -count=1 ./internal/decision/...` — Reflect deve produrre output non vuoto per input realistici

##### W2b — GENESIS Suggester + Sandbox reali (2-3gg)
- **Stato**: `Suggester.Analyze()` stub, Sandbox regex-matching
- **Obiettivo**: Suggester basato su pattern matching + embedding similarity (non LLM). Sandbox con restrizioni reali (resource limits, timeout, syscall filtering via seccomp o Docker).
- **ON-ICE**: Se non c'è tempo per sandbox containerizzata, documentare come "heuristic filter" e classificare come Parziale.
- **Test**: Suggester deve produrre ≥1 suggerimento valido per tool input conosciuto. Sandbox deve bloccare codice malevolo noto.

##### W2c — NLP Sentiment reale (3-5gg)
- **OPZIONE A** (consigliata): Deprecare onestamente la sentiment analysis come "confidence: low", rimuovere claim README di ML, usare word-counting come baseline documentata.
- **OPZIONE B**: Sostituire con vero modello ONNX (classificazione testo leggero) + pipeline di calibrazione.
- **Cosa NON fare**: Ingannare il word-counter con più parole. Non migliora nulla.
- **Test gRPC contract**: `docker compose up nlp-sidecar && go test ./internal/nlp/... --tags=contract`

##### W2d — Repair engine AST-aware upgrade (2-3gg)
- **Stato**: 11+ strategie con string-manipulation naive
- **Obiettivo**: Fix strutturali basati su AST (non `strings.Contains`). `executeRegenerate` deve almeno provare una rigenerazione.
- **Vincolo**: Mantenere backward compatibility per strategie funzionanti (`fixMissingImports`, `fixSyntaxError` base).
- **NO-GO**: Se l'upgrade AST richiede più di 3gg, documentare strategie come "heuristic best-effort" e classificare.

##### W2e — SQL injection fix + auth hardening (2gg - BLOCKING)
- **BLOCKER**: Fixare 9 vector `fmt.Sprintf` in `memory/store.go`. Parametrizzare TUTTE le query SQL.
- **Aggiungere**: `validateSQLName` rafforzato con regex rigorosa o whitelist.
- **RBAC foundation**: Strato base di autorizzazione (ruoli: admin/user/readonly). Non serve OAuth completo, ma API-key deve supportare scope.
- **API key**: Rimuovere da localStorage plaintext. Usare httpOnly cookie o encrypted session storage.
- **Test**: `go test -race -count=1 ./internal/memory/...` + test con table name malevolo

##### W2f — gRPC contract testing + NLP output standardization (1-2gg)
- **Pinnare** proto definition tra Go backend e Python NLP
- **Aggiungere** contract test CI che verifica: tutti i metodi proto hanno implementazione lato Python, e Go consumer matcha i tipi
- **Standardizzare** output probabilistico: schema confidence unico (0.0-1.0) con flag `is_calibrated`

---

### Track B — Monitoring Stack + Frontend UX

#### W3-Mon — Monitoring infrastructure (3-5gg, parallelo a W2)

**NON differibile a W4.** Senza monitoring stack, le exit criteria di W5 ("SLO monitorabili") sono impossibili.

**Attività**:
- Aggiungere `/metrics` endpoint Go (Prometheus counter/histogram per request count, latency, error rate)
- Aggiungere `prometheus` + `grafana` + `alertmanager` a `docker-compose.yml`
- Creare 3 dashboard Grafana minime: (1) API/Servizi, (2) NLP sidecar, (3) DB/Storage
- Configurare SLO burn-rate alerting rules (error budget 5% rolling 30gg)
- Correlation ID propagation nei log strutturati

**Exit criteria**:
- `curl -s http://localhost:9090/metrics | grep aleph_` → metriche popolate
- Grafana accessibile su `localhost:3000` con 3 dashboard
- Alertmanager configurabile con regole SLO

#### W3-UX — Scenario comparison + Frontend polish (2-3gg, parallelo a W2)

**Obiettivo**: Solo ciò che manca veramente. Frontend è 96% pronto.

**Attività**:
- **Scenario comparison view**: UI per confrontare 2-3 scenari side-by-side (probabilità, segnali, confidence)
- **WCAG 2.1 AA audit**: keyboard navigation, focus management, contrast ratio, aria labels
- **Lighthouse target**: ≥90 performance, ≥90 accessibility, ≥90 best-practices
- **Re-render fix**: React.memo su componenti che usano `useStore()` con slice selettori

**Cosa NON fare**: Non rifare il design system. Non aggiungere animazioni. Non microcopy tuning infinito.

**Exit criteria**:
- Scenario comparison view: seleziona 2 scenari → confronto visivo side-by-side
- Lighthouse ≥90 su tutti e 3 i metriche
- WCAG 2.1 AA pass (scan automatico + keyboard-only navigation test)

---

### Track C — Security, Resilience, Release

#### W4-Sec — Security hardening + resilience (3-5gg, parallelo a W2+W3)

**Attività**:
- **Sistema threat model**: Mappare data flow, trust boundaries, attori. Coprire auth, ingestion, MCP discovery.
- **SAST in CI**: Aggiungere `govulncheck` + `npm audit --audit-level=high` al workflow
- **DAST baseline**: `trivy image` scan su immagine Docker
- **Chaos drills**: NLP sidecar down test, DB latency injection, timeout escalation
- **Audit trail**: Completare logging eventi PAORA con correlation ID e strutturazione

**Exit criteria**:
- `govulncheck ./...` — no high/critical vulnerabilities
- `npm audit --audit-level=high` — 0 vulnerabilities
- `trivy image aleph-v2:latest` — no critical CVEs
- Chaos drill: NLP down → sistema degrada graceful senza crash
- Audit trail consultabile per evento decisionale PAORA

#### W5-Rel — Release candidate + go-live (2-3gg)

**Attività**:
- **Contract test CI gate**: `docker compose up nlp-sidecar && go test ./internal/nlp/... --tags=contract` nel workflow
- **E2E Playwright in CI**: Scenario utente critico (login → scenario → confronto → decision)
- **Load test**: `go test -bench=. ./...` baseline + bottleneck identification
- **Documentazione**: Aggiornare README con capacità oneste (non rivendicare ML dove non c'è). Architettura, operatività, sicurezza.
- **Release tag semantico**: `v2.x.x` + release notes automatiche
- **Rollout controllato**: Tag Docker, deploy, post-deploy smoke, rollback procedure

**Exit criteria**:
- Tutti i quality gates verdi in CI
- Contract test Go↔Python NLP passa
- E2E Playwright smoke passa in CI
- Documentazione aggiornata e onesta
- Go/No-Go documentato con evidenze

---

## Quality Gates Bloccanti (per traccia, non globali)

### Track A Gate (W2)
- [ ] `go build ./...` ✅
- [ ] `go test -race -count=1 ./internal/decision/...` ✅ (PAORA Reflect non vuoto)
- [ ] `go test -race -count=1 ./internal/memory/...` ✅ (SQL injection fix)
- [ ] `docker compose up nlp-sidecar && go test ./internal/nlp/... --tags=contract` ✅
- [ ] Nessuna capability README in stato Stub senza piano di remediation documentato

### Track B Gate (W3)
- [ ] `curl -s http://localhost:9090/metrics | grep aleph_` ✅
- [ ] Lighthouse ≥90 (performance, accessibility, best-practices)
- [ ] Scenario comparison: seleziona 2 scenari → confronto visibile
- [ ] `go test -race -count=1 ./...` ✅

### Track C Gate (W4+W5)
- [ ] `govulncheck ./...` — no high/critical
- [ ] `npm audit --audit-level=high` — 0 vulnerabilities
- [ ] `trivy image aleph-v2:latest` — no critical CVEs
- [ ] Contract test gRPC Go↔Python in CI ✅
- [ ] E2E Playwright smoke in CI ✅
- [ ] `docker compose config` ✅
- [ ] RELEASE TAG `v2.x.x` + release notes

---

## Acceptance Criteria (comandi eseguibili, NON descrizioni)

### Capability verificabili con comandi shell

| # | Criterio | Comando |
|---|----------|---------|
| 1 | PAORA Reflect non è pass-through | `grep -c "return plan, nil" internal/decision/observer.go` → 0 |
| 2 | SQL injection fix | `grep -c 'fmt\.Sprintf.*SELECT.*%' internal/memory/store.go` ≤ storico (deve calare) |
| 3 | Sentiment non è word-counting | `wc -l nlp/sentiment_vocab.txt` → 0 (file rimosso) OPPURE classificato Stub in README |
| 4 | GENESIS Suggester produce output | `go test -run TestSuggester -v ./internal/genesis/...` → output non vuoto |
| 5 | gRPC contract matcha | `go test --tags=contract -run TestNLPContract ./internal/nlp/...` ✅ |
| 6 | Metrics endpoint | `curl -s http://localhost:9090/metrics \| grep aleph_request_total` → match trovato |
| 7 | Lighthouse ≥90 | `npx lighthouse http://localhost:5173 --output=json \| jq '.categories.*.score'` → tutti ≥0.9 |
| 8 | Nessun high/critical in Go | `govulncheck ./... \| grep -E "HIGH\|CRITICAL"` → 0 |
| 9 | Nessun high in npm | `npm audit --audit-level=high` → exit code 0 |
| 10 | Trivy OK | `trivy image aleph-v2:latest \| grep CRITICAL` → 0 |

---

## Risk Register

| # | Rischio | Probabilità | Impatto | Mitigazione |
|---|---------|-------------|---------|-------------|
| R1 | Stub trattati come gap → miglioramento cosmetico senza sostanza | Alta | Critico | W1: classificazione 4-livelli obbligatoria. Stub richiede decisione di redesign o deprecazione. |
| R2 | PAORA Reflect redesign troppo complesso per 2-3gg | Media | Alto | Scope ridotto: implementare reflection strutturata base. LLM-based reflection deferita a post-MVP. |
| R3 | NLP sentiment non sostituibile in 3-5gg | Media | Alto | Deprecare onestamente come "baseline euristica". Rimettere ONNX in roadmap futura. |
| R4 | Nessun monitoring stack a W5 | Media | Critico | Track B parte in W2, non W4. Obbligatorio Prometheus+Grafana prima di W5. |
| R5 | Cross-engine inconsistenza DuckDB↔PostgreSQL | Media | Alto | Documentare limite: nessuna transazione cross-engine. Accettare consistenza eventuale. |
| R6 | Frontend re-render storm non diagnosticabile | Bassa | Medio | Aggiungere React DevTools profilazione in W3-UX. Se non risolvibile, documentare come limite noto. |

---

## Governance

- **Daily check-in**: 15min sync su progresso tracce, blocker, dipendenze incrociate
- **We use git tags**: `checkpoint/track-a-w1`, `checkpoint/track-b-w3-mon`, ecc. Dopo ogni milestone
- **Rollback**: Se un checkpoint rompe i quality gates, revert a tag precedente
- **Change control**: Nessuna nuova feature. Solo chiusura gap documentati nella matrice W1
- **Definition of 100% completato**: Ogni capability README è Stub/Parziale/Funzionale/Produzione con comando di verifica associato. Tutti i quality gates verdi. Documentazione onesta. Rollout controllato senza regressioni critiche.

---

---

## Collegamento con le specifiche esecutive

Questo piano strategico è accoppiato al file di specifiche esecutive:

| Documento | Ruolo |
|-----------|-------|
| [`gpt-plan-revised.md`](gpt-plan-revised.md) | **Piano strategico**: gap analysis, scope, tracce parallele, risk register, governance |
| [`gpt-plan-specs.md`](gpt-plan-specs.md) | **Specifiche esecutive**: task breakdown, acceptance criteria in comandi shell, dependency graph, quality gates matrix |

I due documenti sono progettati per essere letti insieme: il piano fornisce il "perché" e il "cosa", le specifiche forniscono il "come" e il "verifica che".

---

## Appendice: Mappatura review originali

| Review | Risultato chiave | Dove incorporato |
|--------|------------------|------------------|
| Metis | Stub vs. gap classification | W1: classificazione 4-livelli obbligatoria |
| Oracle | ~45% production-grade | Matrice stato reale in Scope A-G |
| Momus | GENESIS assente dal piano | Aggiunto a W2b |
| Momus | Quality gates non per-wave | Gates ora per-traccia |
| Oracle | DuckDB/PostgreSQL consistency | Risk register R5 |
| Metis | Monitoring stack da zero | Track B W3-Mon parallelo |
| Oracle | Reflect pass-through | W2a redesign esplicito |
| Momus | Auto-repair 11+ strategie, 7 citate | W2d con classificazione realistica |
| Oracle | gRPC contract mismatch | W2f + gate specifico |
| Metis | Effort sottostimato 50-60% | Timeline 22-30gg vs 14-20gg |
