# Piano dettagliato di completamento produzione (100%)

## Obiettivo
Portare Aleph a uno stato **production-grade completo al 100%** su tutte le capability dichiarate nel README, con completamento/integrazione delle parti ancora parziali, senza placeholder e senza TODO futuri.

## Principi di esecuzione
1. **Feature parity totale con README**: ogni capability dichiarata deve avere implementazione verificabile end-to-end.
2. **Definition of Done rigorosa**: ogni stream è “done” solo con codice, test, osservabilità, sicurezza e documentazione allineati.
3. **Zero regressioni**: quality gates obbligatori su backend, frontend, NLP e orchestrazione.
4. **Production UX**: qualità prodotto pari a startup finanziata, con attenzione a performance, affidabilità e chiarezza decisionale.

## Scope di completamento

### A) Core platform (Backend Go + Connect RPC)
- Audit completo di tutti i package `internal/*` per individuare feature flaggate, stub, fallback degradati, path incompleti.
- Chiusura dei gap nei flussi:
  - ingestion multi-fonte (RSS/GitHub/CSV/JSON/sitemap/sheets/email)
  - scenario generation probabilistica
  - ensemble orchestration (Prophet + GBM + NLP + tool dispatch)
  - ciclo decisionale PAORA completo con tracciabilità eventi
  - auto-repair strategy engine (7 strategie) con metriche di efficacia
- Hardening API:
  - contratti request/response consolidati e versionati
  - error taxonomy coerente (4xx/5xx + codici dominio)
  - idempotenza per operazioni critiche

### B) Intelligence layer (Python NLP sidecar)
- Completamento pipeline NLP per sentiment/scoring/normalizzazione segnali.
- Verifica fallback ONNX/PyTorch e comportamento in degradazione.
- Test di compatibilità modello + contract test gRPC con backend.
- Standardizzazione output probabilistico (calibrazione confidence + schema unico).

### C) Data layer (DuckDB + PostgreSQL)
- Revisione migrazioni: ordine, rollback safety, backward compatibility.
- Consolidamento persistenza dual-engine:
  - DuckDB per analytics/VSS
  - PostgreSQL per workload transazionali/metadati
- Data retention e lifecycle policy (snapshot, cleanup, vacuum/maintenance).

### D) Frontend (React/TS/Vite/Tailwind)
- Completa copertura UX dei casi d’uso principali:
  - workspace adattivo
  - lettura scenari e confidence
  - confronto ipotesi e segnali
  - stato agenti/tools e spiegazioni azionabili
- Design system production-ready:
  - componenti coerenti, stati loading/error/empty definiti
  - accessibilità (WCAG AA), keyboard navigation, focus management
- Performance FE: bundle strategy, code-splitting, caching, SSR/CSR boundary review (se applicabile).

### E) Security & Compliance
- Hardening middleware: auth/audit/CORS/CSRF/rate limit/timeout/bulkhead/security headers.
- Secret management e policy di configurazione ambienti (dev/stage/prod).
- Validazione protezioni SSRF su MCP discovery.
- Threat model aggiornato + test negativi (abusi input/tooling).

### F) Observability & Operability
- Logging strutturato end-to-end con correlation IDs.
- Metriche RED/USE per API, job ingestion, NLP sidecar, DB, queue/worker.
- Dashboard operative + alerting SLO-based.
- Runbook incident response, degradazione controllata, rollback standard.

### G) CI/CD & Release engineering
- Pipeline unica con gate bloccanti:
  - lint + test + race + vet + build + security scan
  - frontend typecheck + unit/e2e + build
  - contract/integration test cross-service
- Artefatti immutabili, versioning semantico, release notes automatiche.
- Deploy strategy zero-downtime (blue/green o rolling con health gates).

## Piano esecutivo a onde (detagliato)

### Wave 1 — Discovery tecnico e matrice gap (2-3 giorni)
**Deliverable**
- Matrice “README capability → implementazione reale → test → owner”.
- Elenco gap classificati: bloccanti, major, polish.

**Attività**
- Code walk sistematico backend/frontend/nlp/migrations.
- Inventario endpoint e use-case coperti/non coperti.
- Mappatura dipendenze e colli di bottiglia.

**Exit criteria**
- 100% funzionalità README mappate.
- Backlog esecutivo prioritizzato e congelato.

### Wave 2 — Chiusura gap funzionali core (4-6 giorni)
**Deliverable**
- Tutte le feature core completate senza branch incompleti.

**Attività**
- Implementazione dei gap backend e intelligence.
- Uniformazione contratti RPC/gRPC e payload UI.
- Eliminazione dead code/path sperimentali non governati.

**Exit criteria**
- Nessuna funzionalità README in stato parziale.
- Test unit/integration verdi sui moduli toccati.

### Wave 3 — UX/Prodotto e qualità percepita (3-5 giorni)
**Deliverable**
- Esperienza utente premium E2E, pronta per stakeholder non tecnici.

**Attività**
- Rifinitura IA/UX workspace, scenari, explainability.
- Accessibilità, microcopy, feedback realtime, progressive disclosure.
- Performance tuning frontend.

**Exit criteria**
- Flussi principali completabili senza ambiguità.
- Lighthouse/perf/accessibility entro target concordati.

### Wave 4 — Security, resilienza, operatività (3-4 giorni)
**Deliverable**
- Piattaforma robusta sotto fault e abuso.

**Attività**
- Test hardening middleware e protezioni input.
- Chaos/failure drills su servizi critici (NLP down, DB latency, timeout).
- Dashboard/alert/runbook pronti.

**Exit criteria**
- Nessun high/critical aperto.
- SLO operativi monitorabili in tempo reale.

### Wave 5 — Release candidate e go-live (2 giorni)
**Deliverable**
- RC firmata + checklist go-live completata.

**Attività**
- Full regression + UAT + smoke in ambiente release.
- Documentazione finale (architettura, operatività, sicurezza, onboarding).
- Tag release, rollout controllato, verifica post-deploy.

**Exit criteria**
- Tutti i quality gates verdi.
- Decisione Go/No-Go documentata con evidenze.

## Quality Gates obbligatori (bloccanti)
- Backend: `go build ./...`, `go test -race -count=1 ./...`, `go vet ./...`.
- Frontend: `npx tsc --noEmit`, `npx vitest run`, `npx vite build`.
- Platform: `docker compose config` + smoke multi-servizio.
- Security: secrets scan + dependency audit + policy checks.
- E2E: scenari utente critici validati end-to-end.

## KPI di successo produzione
- **Affidabilità**: error rate API sotto soglia SLO.
- **Performance**: p95 latenza endpoint critici entro target.
- **Qualità predittiva**: tracking drift e calibrazione confidence.
- **Usabilità**: task success rate elevato su flussi core.
- **Operatività**: MTTR ridotto grazie a alerting/runbook.

## Governance di esecuzione
- Daily engineering review con burn-down dei gap.
- Checkpoint bisettimanali prodotto/architettura.
- Change control su scope: nessuna nuova feature finché i gap README non sono chiusi.

## Definizione finale di “100% completato”
Il progetto è considerato completato al 100% solo quando:
1. Ogni capability dichiarata nel README è implementata, integrata e testata E2E.
2. Tutti i quality gates sono verdi in CI e replicabili localmente.
3. Documentazione tecnica/operativa/sicurezza è aggiornata e utilizzabile dal team.
4. È stato eseguito un rollout controllato con monitoraggio attivo senza regressioni critiche.
