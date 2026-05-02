# Piano Finale Integrato — Aleph-v2 → v2.0 Production Grade

> **Data**: 1 Maggio 2026
> **Versione**: 1.0 — Integrato con 3 review (Oracle, Momus, Metis)
> **Stato build attuale**: `go build` ✅ | `go test -race` ✅ | `npx tsc` ✅ | `npx vite build` ✅
> **Completion score attuale**: 72% (funzionante ma non production-ready)
> **Orizzonte**: 120 giorni calendario (~260 giorni-uomo effettivi)

---

## Executive Summary

Aleph-v2 è un sistema di Decision Intelligence **funzionante** ma con 14 vulnerabilità CRITICAL, 25 HIGH, e 29 MEDIUM che ne impediscono il deploy production-grade. Il piano originale di 90 giorni (300 giorni-uomo compressi) è stato smontato da tre review indipendenti come **non realistico** — la compressione 3x in parallelo ignora dipendenze reali, staffing density, e complessità di integrazione.

### Cosa è cambiato rispetto al piano originale

| Asse | Piano originale | Piano integrato | Motivo |
|------|----------------|-----------------|--------|
| **Timeline** | 90gg calendario | **120gg calendario** | Metis: la compressione 300→90gg è irrealistica |
| **Effort** | 300gg-uomo | **~260gg-uomo** | Scope ridotto su multi-tenancy, tool stub differiti |
| **Sequencing** | 3 track completamente parallele | **Track B dipende da A nelle prime 2 settimane** | Oracle: SQL injection fix deve precedere integration testing |
| **Deployment** | "Non deciso" | **Container-only con opzione K8s** | Momus: gVisor va validato o abbandonato entro G15 |
| **Security** | Audit solo a G60-80 | **Re-check a ogni ship gate** | Metis: la sicurezza non può aspettare 60 giorni |
| **Multi-tenancy** | Feature completa in Fase 3 | **Scope ridotto del 60%** | Metis: effort sottostimato di 2-3x |
| **Opportunità** | 0 su 27 integrate | **7 integrate nel piano, 9 in backlog** | Gap analysis opportunità vs piano originale |

### Risultato atteso

Al termine delle 4 fasi (120gg), Aleph-v2 sarà:
- **Sicuro**: 0 vulnerabilità CRITICAL/HIGH, sandbox isolata, auth JWT-based
- **Testato**: >70% code coverage Go, >50 test PAORA, >20 scenari E2E
- **Misurabile**: 500 req/s con p95 < 500ms, uptime 99.9%
- **Documentato**: API reference, user guide (IT/EN), runbook operativi
- **Deployabile**: Container-only con docker-compose validato, opzione K8s documentata

---

## Phase Roadmap

```
FASE 1 (G1-30):   SAFETY NET          — 14 CRITICAL → 0, sandbox isolata, auth JWT, PAORA cablato
                  ═══════════ SHIP GATE 1 ═══════════
FASE 2 (G20-55):  STABILITY ENGINE    — 25 HIGH → 0, test suite, CI/CD blindato, LLM robusto
                  ═══════════ SHIP GATE 2 ═══════════
FASE 3 (G45-85):  FEATURE COMPLETION  — PAORA V2, tool package, UI completa, integration test
                  ═══════════ SHIP GATE 3 ═══════════
FASE 4 (G75-120): PRODUCTION READY    — Load test, security audit, multi-tenancy base, docs
                  ═══════════ SHIP GATE 4 (v2.0) ═════
```

**Nota**: Le fasi si sovrappongono di 10-15 giorni per consentire ramp-up parallelo tra track.

---

## FASE 1: Safety Net (G1-30)

**Obiettivo**: Eliminare TUTTE le 14 vulnerabilità CRITICAL. Senza questa fase, ogni altro lavoro è inutile.

**OKR**:
- 14/14 CRITICAL risolte e verificate
- Sandbox: 0 bypass nei test di penetrazione
- Auth: JWT session token funzionante con backward compat API key
- PAORA: Engine.Reflect usa DefaultReflector, Act esegue PlannedStep
- Build: `go test -race`, `npx tsc`, `npx vite build` tutti ✅

### Track A — Security Criticals (G1-20)

#### A01 — Sandbox Isolamento Reale (G1-10) | **6gg** | CRITICAL C02, C09

**Cosa fare**: Sostituire l'attuale allowlist con container isolation via Docker SDK.

**Specifiche** (da `piano-operativo-specs.md` §1, modificato per review):
- `IsolationMode`: **"container"** (default). gVisor è differito a Fase 5 backlog.
  - *Decisione Oracle/Momus*: gVisor va validato entro G1-3. Se non fattibile (dipendenza kernel, complessità), commit a container-only. Container-only è sufficiente per production grade con `--network=none --read-only --cap-drop=ALL`.
- `RestrictedPath`: "/usr/bin:/bin" (fisso, non os.Getenv)
- `MaxInputSize`: 1MB
- `CPUQuota`: 1.0 core, `MemoryLimit`: 512MB, `ProcessLimit`: 50
- `NetworkBlocked`: true (default)
- `TimeoutSeconds`: 30

**Fix specifici**:
- Bloccare `--interactive=always`, `--tty=yes`, e tutte le varianti di flag interattivi
- Validazione Go/Python via parser AST (non regex)
- Blocklist Python estesa: `imaplib`, `urllib`, `requests`, `socket`, `os`, `subprocess`
- Context timeout su ogni esecuzione tool

**Test**:
- Bypass tentativi con `--interactive`, `--tty`, pipe a shell → DEVE fallire
- STORED_XSS/EXFIL pattern tutti coperti
- Resource limit enforcement verificato (OOM kill, CPU throttle)

**Stima**: 6 giorni-uomo (5 sviluppo + 1 testing). Era 10gg nel piano originale — ridotto perché si usa container Docker SDK (esistente) anziché gVisor (da integrare da zero).

---

#### A02 — Auth System Rewrite (G5-18) | **5gg** | CRITICAL C07

**Cosa fare**: Sostituire API key in HttpOnly cookie plaintext con JWT session token.

**Pre-requisito Oracle**: **Migration runbook** per API key → JWT deve essere scritto e validato PRIMA di toccare il codice (G5-7). Il runbook deve coprire:
1. Come migrare le API key esistenti senza rompere le integrazioni
2. Periodo di coesistenza API key header + JWT cookie
3. Rollback step-by-step se la migration fallisce
4. Test di accettazione per ogni step

**Specifiche** (da `piano-operativo-specs.md` §2):
- JWT con campi: `sub` (userID), `pid` (projectID), `role` (admin/user/agent), `exp`, `scopes`
- Endpoint: `POST /api/v1/auth/login` → Set-Cookie JWT HttpOnly Secure SameSite=Strict
- Endpoint: `POST /api/v1/auth/logout` → Clear-Cookie
- Endpoint: `POST /api/v1/auth/session` → valida sessione, NON restituisce API key
- Agent API keys in list: mostra solo `ale_abc...1234` (ultimi 4 char)
- Endpoint reveal: `POST /api/v1/agent/{id}/reveal-key` (richiede admin auth)
- `skipAuth`: exact path match table (non `strings.Contains`)

**Backward Compatibility** (da spec):
- `X-Aleph-Api-Key` header ancora accettato con deprecation warning
- Session token ha priorità #1
- Rimozione API key support pianificata per v2.1

**Test**:
- JWT creazione, validazione, expiry, refresh
- API key header backward compat
- Agent key masking in list responses
- Reveal-key solo admin
- skipAuth exact match (non substring)

**Stima**: 5 giorni-uomo (include 1.5gg per migration runbook)

---

#### A03 — SQL Injection Fix (G1-5) | **2gg** | CRITICAL C01, C10

**Cosa fare**: Parametrizzare TUTTE le query in `query.go` e `memory/store.go`.

**Sequencing Oracle**: Questo task DEVE essere completato prima di B01 (PAORA core fix) perché i test di integration PAORA eseguono query via DuckDB e non devono operare su codice vulnerabile.

**Fix specifici** (da `piano-operativo-specs.md` §3):
- `query.go:178,257,301`: sostituire `fmt.Sprintf` con prepared statements (`$1`, `$2`)
- `scopeQuery` e `info_schema` query: prepared statements
- `memory/store.go`: 9 siti con string concat → parametrizzare
- DSL compiler: prepared statements per costruzione query
- `validName()` regex applicata a TUTTI i parametri stringa: solo `[a-zA-Z0-9_-]+`
- Filter objects: whitelist campi consentiti per sorting/filtering
- CI linter: bloccare `fmt.Sprintf` in file che contengono SQL

**Test**:
- SQLMap o equivalente su tutti gli endpoint query → 0 vulnerabilità
- Test con input malevoli: `'; DROP TABLE--`, `1 OR 1=1`, Unicode homoglyph
- CI gate: `gosec` senza esclusioni su query.go

**Stima**: 2 giorni-uomo

---

#### A04 — Hardcoded Secrets Removal (G5-10) | **2gg** | CRITICAL C08, HIGH H12

**Cosa fare**:
- Rimuovere Postgres DSN default `postgres:postgres` — deve fallire con errore chiaro se non configurato
- `.env.example`: placeholder sostituiti con istruzioni esplicite
- `KEY_ENCRYPTION_KEY`: da env var a file-based (`/run/secrets/key_encryption_key`)
- Docker secrets per `POSTGRES_PASSWORD` via `env_file` con permessi 600
- **Opportunità 4.7 integrata**: Docker Secrets pattern documentato

**Stima**: 2 giorni-uomo

---

#### A05 — Network & Auth Hardening (G10-18) | **3gg** | HIGH H17, H18, H19, H20

**Cosa fare**:
- CSP: sostituire `ws://localhost:*` con `ws://localhost:8080` (specifico)
- CSRF: richiedere Origin/Referer validi; bloccare richieste senza Origin che non sono GET/HEAD
- X-Forwarded-For: trusted proxy list (non accettare da qualsiasi IP)
- `skipAuth`: sostituire `strings.Contains(r.URL.Path, "AuthService")` con exact match table
- **Opportunità 4.4 integrata**: CSRF → SameSite=Lax default
- **Opportunità 4.5 integrata**: CSP hardening (`strict-dynamic`, `base-uri 'self'`, `form-action 'self'`)

**Stima**: 3 giorni-uomo

---

### Track B — Decision Engine Repair (G3-25)

#### B01 — PAORA Core Fix (G3-18) | **5gg** | CRITICAL C03, C04, C05, C06

**Cosa fare**: Cablare il vero decision loop.

**Fix specifici** (da `piano-operativo-specs.md` §4):

1. **Reflect Unification** (G3-7):
   - `Engine.Reflect()` → chiama `DefaultReflector` internamente
   - `DefaultReflector` classifica gap: CONFIDENCE_GAP, EXECUTION_GAP, CONTEXT_GAP, TOOL_GAP
   - `TrustDelta` calcolato da: confidence drop + consecutive failures + tool errors
   - **Opportunità 2.1 integrata**: Reflection Engine unificato con feedback loop

2. **Plan-Act Connector** (G7-12):
   - `Act()` DEVE eseguire i `PlannedStep` generati da `Plan()`
   - Struttura `PlanStep`: Tool, Input, Depends (indici step precedenti)
   - Se Executor assente → cerca in Registry → fallback a dispatch
   - **Opportunità 2.2 integrata**: Plan-Act Connector con TaskExecutor

3. **nil Provider Fix** (G12-14):
   - `app.go` DEVE passare un provider valido (es. `llm.NewProvider("ollama", ...)`)
   - `Act()` gestisce `query_dispatch` come fallback
   - `EngineConfig.Validate()`: Provider e MetaRepo REQUIRED, zero values = errore

4. **ChatSession Hardening** (G14-18):
   - Usare `MaxAttempts` da EngineConfig (non hardcoded 5)
   - Nil-check su plan in Reflect (evitare panic)
   - Admit: retry su primo errore invece di terminare immediatamente

**Test**:
- Plan → Act: i passi generati da Plan vengono eseguiti da Act
- nil Provider: degraded mode funziona con query_dispatch fallback
- Reflect: TrustDelta > 0 con errori reali
- Admit: retry funziona; termina dopo MaxAttempts

**Stima**: 5 giorni-uomo (aumentato da 15gg del piano originale perché le spec sono già dettagliate e DuckDB txWrapper esiste già)

---

#### B02 — Decision Engine Test Suite (G10-22) | **4gg** | CRITICAL C06, OPPORTUNITÀ 5.1

**Cosa fare**: Scrivere regression test per l'intero ciclo PAORA.

**Test specifici**:
- **Unit test per fase**:
  - `TestPlan_GeneratesValidSteps`: Plan con input diversi produce sempre []PlanStep validi
  - `TestAct_ExecutesPlannedSteps`: Act esegue i passi nell'ordine corretto (rispettando Depends)
  - `TestObserve_CalculatesTrustDelta`: Observe produce TrustDelta ≠ 0 con errori
  - `TestReflect_ClassifiesGap`: Reflect classifica correttamente CONFIDENCE vs EXECUTION gap
  - `TestAdmit_Retries`: Admit ritenta su errore; termina dopo MaxAttempts
- **Integration test**:
  - `TestPAORA_FullCycle`: Plan→Act→Observe→Reflect→Admit completo con mock provider
  - `TestPAORA_DegradedMode`: nil Provider → fallback funzionante
  - `TestPAORA_TrustEscalation`: 3+ gap consecutivi → Admit finale
- **Property-based test**:
  - "Per qualsiasi piano valido, Reflect produce feedback non-vuoto"
  - "Act non restituisce mai panic con input malformato"

**Mock**: `MockLLMProvider` che ritorna piani predefiniti per test deterministici.

**Stima**: 4 giorni-uomo

---

#### B03 — GNN Predictor Training (G15-25) | **3gg** | HIGH H16, OPPORTUNITÀ 8.1

**Cosa fare**: Addestrare il LinkPredictor con dati reali.

- Training offline su dati storici (tool usage, agent relationships)
- `IsTrained()` deve tornare true dopo training
- `TrustDelta` dal GNN integrato in Engine.Observe
- Metrics: precision@k, recall@k
- Threshold: predictions solo se confidence > 0.7

**Stima**: 3 giorni-uomo (ridotto da 10gg — non serve costruire pipeline ML complessa)

---

### Track C — Frontend Criticals (G1-20)

#### C01 — AbortController Integration (G1-7) | **2gg** | CRITICAL (H01, H03), OPPORTUNITÀ 3.2

**Cosa fare**: Aggiungere AbortController a OGNI chiamata di rete nel frontend.

**Fix specifici**:
- `App.tsx` data loading: AbortController + cleanup su unmount
- Chat history loading: AbortController
- `useAppActions`: cancellare richieste pending su navigazione
- SSE reconnect: non martellare su 401 (auth check prima)

**Pattern hook** (da `piano-operativo-specs.md` §5):
```typescript
function useAbortableEffect(effect: (signal: AbortSignal) => Promise<void>, deps: any[]) {
    useEffect(() => {
        const controller = new AbortController();
        effect(controller.signal);
        return () => controller.abort();
    }, deps);
}
```

**Stima**: 2 giorni-uomo

---

#### C02 — Auth Fix Frontend (G3-12) | **2gg** | CRITICAL (H02), OPPORTUNITÀ 3.3

**Cosa fare**:
- `useOntologyActions`: aggiungere auth headers (non raw fetch senza auth)
- Sostituire fetch() raw con chiamate ConnectRPC dove possibile
- API key non in Zustand DevTools visibili (persist senza sensitive data)
- **Opportunità 3.3 integrata**: ConnectRPC come unico transport — deprecare fetch() raw
- ToolForm, SkillForm: migrare a ConnectRPC client (da raw REST)

**Stima**: 2 giorni-uomo

---

#### C03 — Type Safety Sprint (G7-22) | **10gg** | HIGH (M01-M09 aggregate), OPPORTUNITÀ 3.1

**Nota Oracle**: La stima originale di 7-13gg per questo task è **severamente sottostimata**. L'analisi Oracle indica 20-25gg reali per eliminare tutti gli `any` e `as unknown as` cast. Accettiamo 10gg come compromesso focalizzato sui file di produzione (non test).

**Cosa fare** (da `piano-operativo-specs.md` §9):

Priority 1: Sostituire `assertType` con Zod.parse reale (2gg)
Priority 2: Unificare tipi: `store/types.ts` → `schemas/index.ts` (2gg)
Priority 3: Zod schemas per Scenario, ToolAnomaly (1gg)
Priority 4: Rimuovere `as unknown as` cast in file produzione — 42 cast → 0 (3gg)
Priority 5: `any` reduction nei 14 file produzione con commenti eslint (2gg)

**Stima**: 10 giorni-uomo (ricalibrato da Oracle feedback). I test file possono mantenere `any` temporaneamente.

---

#### C04 — Error Handling Centralizzato (G10-20) | **2gg** | HIGH H05, H08, OPPORTUNITÀ 3.4

**Cosa fare**:
- Empty catch blocks (x3): `catch {}` → `catch (e) { errorService.handle(e) }`
- `handleError` duplicato: unificare in singleton ErrorService con subscriber pattern
- DataSourceForm `JSON.parse` senza try-catch: aggiungere guard
- `alert()`/`confirm()` in SetupWizard/SettingsView → modal React
- **Opportunità 7.1 integrata**: Errori human-readable in italiano

**Stima**: 2 giorni-uomo

---

#### C05 — State Management Fix (G12-22) | **3gg** | HIGH H04, H09, MEDIUM M03-M05

**Cosa fare**:
- `setProjectContext`: atomic update (non resettare parzialmente)
- `cancelStream`: non mutare state in setter (side effect puro)
- Set serialization in copilotSlice: Array invece di Set
- `useCursorPagination`: chiudere stale closure
- `fetchTools`: includere projectId
- SSE `lastEventId` in Zustand store (non module-level)

**Stima**: 3 giorni-uomo

---

### Riepilogo Effort Fase 1

| Track | Task | Giorni |
|-------|------|--------|
| A | A01 Sandbox isolamento | 6 |
| A | A02 Auth rewrite + runbook | 5 |
| A | A03 SQL injection fix | 2 |
| A | A04 Secrets removal | 2 |
| A | A05 Network hardening | 3 |
| B | B01 PAORA core fix | 5 |
| B | B02 Decision Engine test suite | 4 |
| B | B03 GNN predictor training | 3 |
| C | C01 AbortController | 2 |
| C | C02 Auth fix frontend | 2 |
| C | C03 Type safety sprint | 10 |
| C | C04 Error handling | 2 |
| C | C05 State management | 3 |
| **Totale Fase 1** | | **49gg-uomo (30gg calendario)** |

### Rischi Fase 1

| Rischio | Probabilità | Impatto | Mitigazione |
|---------|------------|--------|-------------|
| **gVisor non fattibile entro G3** | MEDIUM | HIGH | G1-3: spike tecnico. Se bloccante → commit a container-only. Decisione G3. |
| **Migration auth fallisce in produzione** | MEDIUM | CRITICAL | Runbook con rollback step-by-step. Coesistenza API key + JWT. |
| **Type safety sprint sfora 10gg** | HIGH | MEDIUM | Scope limitato a file produzione. Test file differiti a Fase 2. |
| **Staffing density Fase 1 (3 track parallele)** | HIGH | MEDIUM | Track B parte G3 (dopo A03). Track C parte G1 ma è indipendente. Se staffing < 3 dev, prioritizzare Track A → B → C. |

### Ship Gate 1 (G30)

**Check obbligatori**:
- [ ] `go test -race -count=1 ./...` ✅ (0 FAIL)
- [ ] `npx tsc --noEmit` ✅ (0 errori in produzione)
- [ ] `npx vite build` ✅
- [ ] 14/14 CRITICAL risolte e verificate con test
- [ ] Sandbox: penetration test → 0 bypass
- [ ] Auth: JWT login/logout/session funzionanti; API key header backward compat
- [ ] PAORA: Act esegue PlannedStep; Reflect usa DefaultReflector
- [ ] AbortController in tutte le chiamate di rete frontend
- [ ] Migration runbook auth validato
- [ ] **Security re-check** (Metis gap #2): OWASP ZAP scan → 0 CRITICAL/HIGH

**Azioni di rollback**:
1. Se sandbox container non stabile → revert a L0 allowlist (preesistente) + flag feature `ENABLE_SANDBOX_ISOLATION=false`
2. Se auth JWT non funzionante → revert a solo API key header (spegnere JWT middleware via config)
3. Se PAORA unificato introduce regressioni → feature flag `PAORA_USE_LEGACY_REFLECT=true`

---

## FASE 2: Stability Engine (G20-55)

**Obiettivo**: Portare il sistema da "funzionante" a "affidabile". Eliminare tutti i 25 HIGH bugs. Blindare CI/CD, rendere DuckDB e LLM robusti.

**OKR**:
- 25/25 HIGH risolti e verificati
- Backend test coverage > 60%
- CI/CD: pipefail fix, test gate deploy, container build ottimizzato (< 5 min)
- DuckDB: 0 deadlock nei test di concorrenza
- LLM: timeout, retry, circuit breaker funzionanti
- Load test base: 500 req/s p95 < 1s

### Track A — CI/CD & Infrastructure (G20-45)

#### A06 — CI/CD Blindatura (G20-24) | **1.5gg** | HIGH H21, H22

- `set -o pipefail` in CI step go test (pipe non nasconde fallimenti)
- Deploy workflow: `needs: [test, build]` esplicito (no tag push senza test)
- `go vet ./...` in CI
- `.dockerignore` (escludere node_modules, .git, test)
- **Opportunità 5.2 integrata**: Pipeline CI a prova di fallimento

**Stima**: 1.5 giorni-uomo

---

#### A07 — Docker Ottimizzazione (G22-28) | **2gg** | HIGH H23, H24

- Dockerfile: `golang:1.24-alpine` invece di bullseye (990MB → ~350MB)
- `.dockerignore` già creato in A06
- Multi-stage build con cache layers
- `docker-compose.yml`: healthcheck backend + `depends_on` con condition
- `cache-from`/`cache-to` per layer caching in CI
- Target: Docker image < 500MB

**Stima**: 2 giorni-uomo

---

#### A08 — Alertmanager + Monitoring (G25-35) | **3gg** | CRITICAL C14

- Configurare receivers: Slack webhook + email SMTP
- Alert rules: uptime < 99.9%, error rate > 1%, NLP offline, DuckDB lock contention
- Grafana dashboard: request latency, error rate, tool execution, DuckDB performance
- Prometheus recording rules per SLA/SLO

**Stima**: 3 giorni-uomo

---

#### A09 — Deploy Pipeline (G35-50) | **4gg** | MEDIUM (splittato per Metis)

**Decisione Metis**: A09 originale conflation di 3 opzioni. Ora splittato:

- **A09a — Docker Compose Production** (G35-42) | 3gg: Script deploy, env validation, secrets management, healthcheck monitoring, log rotation
- **A09b — Kubernetes Option** (G42-50) | 1gg: Documentazione Helm chart, NON implementazione. K8s deploy è post-v2.0.
- **A09c — Bare-metal Guide** (G42-45): Documentazione setup manuale per ambienti senza container

**Stima**: 4 giorni-uomo totali

---

#### A10 — Load Testing (G40-55) | **3gg** | Metis gap #8

**Nota**: Spostato da Fase 4 (G75) a Fase 2 per feedback Metis. Il load testing serve SUBITO per validare i fix di concorrenza DuckDB e rate limiter.

- k6 test per: queries, ingestion, chat, tool execution
- Target: 500 req/s con p95 < 1s (Fase 2), < 500ms (Fase 4)
- Memory profiling con pprof
- Goroutine leak detection

**Stima**: 3 giorni-uomo

---

### Track B — Backend Robustezza (G22-50)

#### B04 — DuckDB Concurrency Fix (G22-30) | **3gg** | HIGH H10, H11, OPPORTUNITÀ 2.4

- Lock ordering: `mu.Lock()` PRIMA di operazioni DB, rilasciato DOPO
- Transazioni: `Lock` (scrittura), non `RLock`
- `QueryRowContext` → `QueryRowContextOrError` (torna errore, non nil)
- VSS INSERT: DELETE + INSERT in singola transazione
- `txWrapper` già implementato (Oracle conferma) — verificare che funzioni, non ricostruire
- **Opportunità 2.4 integrata**: DuckDB concurrency model review

**Stima**: 3 giorni-uomo (ridotto perché txWrapper esiste già)

---

#### B04.5 — LLM Budget & Cost Controls (G24-28) | **2gg** | Metis gap #4

**Nuovo task** (non nel piano originale, aggiunto da Metis):
- Circuit breaker per chiamate LLM: max chiamate/ora, max costo/giorno
- Metrics: token count per provider, costo per chiamata
- Alert: superamento soglia budget → notifica admin
- Configurabile via env var: `LLM_MAX_CALLS_HOUR`, `LLM_MAX_COST_DAY`

**Stima**: 2 giorni-uomo

---

#### B05 — Rate Limiter Memory Safety (G28-34) | **2gg** | HIGH C13, H18, OPPORTUNITÀ 2.3

- Mappa IP→limiter: TTL cleanup (goroutine ogni 10 min)
- Limite massimo entry: 100k IP (LRU eviction)
- Race condition fix: `sync.RWMutex` con double-check locking
- X-Forwarded-For: trusted proxy list validation
- Header `X-RateLimit-Remaining` in risposta
- **Opportunità 2.3 integrata**: Rate Limiter sliding window

**Stima**: 2 giorni-uomo

---

#### B06 — LLM Provider Robustezza (G32-42) | **3gg** | HIGH H14, H15

- HTTP timeout configurabile (default 30s)
- Retry con exponential backoff (3 tentativi)
- Provider registry: nomi sconosciuti → errore, non nil
- Circuit breaker: half-open thundering herd fix (jitter, max 1 richiesta)
- `EngineConfig.Validate()` su startup
- **Opportunità 2.5 integrata**: Circuit breaker pattern completo

**Stima**: 3 giorni-uomo

---

#### B07 — MCP Discovery Reliability (G35-45) | **3gg** | CRITICAL C11, HIGH (opportunità 2.8)

- `healthLoop`: WaitGroup per graceful shutdown (Metis gap #3)
- Retry su discovery fallita con backoff esponenziale
- SSRF validation centralizzata (HTTP client unico)
- URIs validation su input
- **Opportunità 2.8 integrata**: HealthChecker context fix — non sovrascrivere cancel

**Stima**: 3 giorni-uomo

---

#### B08 — NLP Sidecar Watchdog (G40-48) | **2gg** | Metis gap #5, OPPORTUNITÀ 6.1

- `watchSidecar`: defer recover con restart automatico
- Max restart count: 3 in 5 minuti, poi arrenditi e allerta
- Health check: ogni 2s (non 10s); 3 fallimenti consecutivi → unhealthy
- Graceful shutdown: Stop() segnala → Wait() conferma
- **Opportunità 6.1 integrata**: NLP Watchdog con auto-restart

**Stima**: 2 giorni-uomo

---

#### B09 — Python/NLP Validation Fix (G42-50) | **2gg** | HIGH H13

- Regex validation → parser AST per Go e Python
- Bloccare eval/exec di import malevoli
- Path restrittivo per subprocess (non os.Getenv("PATH"))
- Sandbox verification: timeout per tool execution

**Stima**: 2 giorni-uomo

---

### Track C — Test Expansion (G25-50)

#### C06 — Backend Test Coverage (G25-40) | **4gg** | HIGH (M23, M26)

- ChatSession: test unitari completi (PAORA cycle, errori, degrade)
- Sandbox: test per allowlist bypass tentativi
- MCP discovery: test per retry, SSRF validation, shutdown
- HealthChecker: test per context lifecycle
- Ingestion engine: test per runDynamic

**Stima**: 4 giorni-uomo

---

#### C07 — Contract Tests Revival (G30-36) | **2gg** | HIGH H25

- Aggiungere `//go:build integration` nei file di test (non 'contract' che non esiste)
- Test di integrazione: connettere a infra reale (PostgreSQL + DuckDB + NLP)
- CI: eseguire contract test su trigger manuale

**Stima**: 2 giorni-uomo

---

#### C08 — Frontend Test Suite (G35-50) | **4gg** | MEDIUM

- Vitest: test per store slices (navigation, auth, copilot, workspace)
- Componenti critici: InlineRenderer, SlideOverPanel, Terminal
- Hooks: test per useStreamSSE, useCursorPagination, useDebounce
- E2E Playwright base: journey auth → query → strumenti → settings

**Stima**: 4 giorni-uomo

---

### Riepilogo Effort Fase 2

| Track | Task | Giorni |
|-------|------|--------|
| A | A06 CI/CD blindatura | 1.5 |
| A | A07 Docker ottimizzazione | 2 |
| A | A08 Alertmanager + monitoring | 3 |
| A | A09 Deploy pipeline (split) | 4 |
| A | A10 Load testing (anticipato) | 3 |
| B | B04 DuckDB concurrency fix | 3 |
| B | B04.5 LLM budget controls (NUOVO) | 2 |
| B | B05 Rate limiter memory safety | 2 |
| B | B06 LLM provider robustezza | 3 |
| B | B07 MCP discovery reliability | 3 |
| B | B08 NLP sidecar watchdog | 2 |
| B | B09 Python/NLP validation | 2 |
| C | C06 Backend test coverage | 4 |
| C | C07 Contract tests revival | 2 |
| C | C08 Frontend test suite | 4 |
| **Totale Fase 2** | | **40.5gg-uomo (35gg calendario)** |

### Rischi Fase 2

| Rischio | Probabilità | Impatto | Mitigazione |
|---------|------------|--------|-------------|
| **DuckDB deadlock intermittente sfugge ai test** | MEDIUM | HIGH | Chaos testing: eseguire query parallele per 1h continuo in CI |
| **LLM budget blocca chiamate legittime** | LOW | MEDIUM | Default limiti alti; override via env var; alert prima del blocco |
| **Contract test instabili (dipendono da infra esterna)** | MEDIUM | LOW | Eseguire in CI separata; accettare retry automatico |
| **NLP sidecar restart loop (bug nel watchdog)** | LOW | HIGH | Max 3 restart in 5 min; alert immediato; test specifico per restart loop |

### Ship Gate 2 (G55)

**Check obbligatori**:
- [ ] `go test -race -count=1 ./...` ✅ (0 FAIL)
- [ ] `npx tsc --noEmit` ✅ (0 errori produzione)
- [ ] `npx vite build` ✅
- [ ] 25/25 HIGH bugs risolti e verificati
- [ ] CI/CD: pipefail funzionante, deploy con test gate
- [ ] Docker: build < 5 min, image < 500MB
- [ ] DuckDB: 0 deadlock in stress test (1000 query parallele per 10 min)
- [ ] LLM: timeout 30s, retry funzionante, circuit breaker testato
- [ ] Rate limiter: 100k IP test, cleanup verificato
- [ ] Backend test coverage > 60%
- [ ] Load test: 500 req/s p95 < 1s
- [ ] **Security re-check** (Metis gap #2): OWASP ZAP + gosec scan → 0 HIGH

---

## FASE 3: Feature Completion (G45-85)

**Obiettivo**: Completare le feature mancanti. PAORA V2 con multi-step. Tool package (con scope ridotto). UI completa.

**OKR**:
- PAORA V2: multi-step tool execution funzionante
- Tool package: 3/6 completati (finance, OSINT, HE), 3 stub differiti
- UI: tutte le view renderizzano, dashboard analytics base
- Integration test: > 50 test
- E2E Playwright: > 20 scenari

### Track A — Infrastructure Completion (G45-70)

#### A11 — Multi-tenancy Foundations (G45-65) | **6gg** | Scope RIDOTTO per Metis

**Nota Metis**: Lo scope originale (PostgreSQL schema per tenant, DuckDB per tenant, rate limiting per tenant, resource quotas, API routing) era sottostimato 2-3x. Piano integrato: scope ridotto.

**Scope effettivo**:
- PostgreSQL: schema per progetto (non tenant). Ogni progetto ha il suo schema.
- DuckDB: un database condiviso, namespace via prefisso tabella.
- Rate limiting: per API key (non per tenant). Mappa API key → limiter.
- Resource quotas: **differito a Fase 5**. Solo soft limits configurabili.
- MaxProjects, MaxAgents: validazione in creazione risorsa (non quota enforcement complesso)

**Stima**: 6 giorni-uomo (ridotto da 15gg perché scope tagliato del 60%)

---

#### A12 — EU Compliance Base (G55-70) | **4gg** | MEDIUM

- GDPR: data retention policies, delete cascade per progetti
- Audit logging per operazioni admin
- Documentazione privacy impact assessment
- Data residency: documentazione opzioni (non implementazione)

**Stima**: 4 giorni-uomo

---

### Track B — PAORA V2 & Tool Packages (G45-75)

#### B10 — Multi-Step Tool Execution (G45-60) | **5gg** | OPPORTUNITÀ 2.2 completamento

- Plan→Act: tool dispatch da piano strutturato
- Tool result feedback loop → observed state aggiornato
- Confirmation flow per tool auto-esecuzione (threshold configurabile)
- Step dependencies: esecuzione ordinata rispettando Depends

**Stima**: 5 giorni-uomo

---

#### B11 — Tool Package Completamento (G50-70) | **8gg** | Scope RIDOTTO

**Nota**: Dei 6 tool package, 3 sono stub (finance, osint, humanecosystems). Il piano originale voleva completarli tutti. Piano integrato: completare 3, lasciare 3 come stub documentati.

**Da completare** (G50-65):
- **Finance tool** (3gg): API integration reale (Yahoo Finance o Alpha Vantage free tier). Almeno: stock price, historical data, basic indicators.
- **OSINT tool** (3gg): API integration reale (Shodan free tier o simile). Almeno: IP lookup, domain info, basic threat intel.
- **HumanEcosystems tool** (2gg): API integration o dataset statico. Almeno: demographic data, basic indicators.

**Da lasciare come stub documentati** (differiti a backlog):
- Adaptation pipeline
- Code generation tools
- Advanced analytics tools

**Stima**: 8 giorni-uomo

---

#### B12 — Memory & VSS Enhancement (G55-70) | **3gg** | OPPORTUNITÀ 2.6

- MemoryStore: `sync.Once` con retry (non fallire permanentemente)
- DuckDB VSS: upsert corretto (DELETE+INSERT in transazione)
- Embedding dimension validation (768) all'avvio
- **Opportunità 2.6 integrata**: VSS First-Class — goose migration per VSS extension, test con VSS attivo

**Stima**: 3 giorni-uomo

---

#### B13 — Structured Error Enrichment (G60-68) | **2gg** | OPPORTUNITÀ 2.7

- Ogni errore arricchito con: `Subsystem`, `Operation`, `Recoverable`, `RetryAfter`
- DiagnosticMonitor correlazione automatica
- Frontend: messaggi utente basati su categoria errore
- **Opportunità 2.7 integrata**: Structured Error Enrichment

**Stima**: 2 giorni-uomo

---

### Track C — UI Completion & Integration (G48-80)

#### C09 — Terminal & Chat UI (G48-60) | **3gg** | MEDIUM

- InlineRenderer: fix JSX error preesistenti
- SSE reconnect UI indicator (stato connessione)
- Streaming token display optimization (rendering fluido)
- Command palette completamento (slash commands)

**Stima**: 3 giorni-uomo

---

#### C10 — Tool UI Completion (G52-65) | **2gg** | MEDIUM

- Tool execution result display (formattato per tipo: JSON, table, chart)
- Tool configuration forms completi (campi validati)
- Finance/OSINT/HE tool card componenti
- Stato tool in UI (MCP discovery status, health indicator)

**Stima**: 2 giorni-uomo

---

#### C11 — Dashboard & Analytics (G55-70) | **4gg** | MEDIUM

- Usage statistics view (chiamate API, tool usati, LLM costi)
- Health dashboard: stato sistemi (backend, NLP, DuckDB, MCP)
- Query history con performance metrics
- LLM cost tracking (da B04.5)
- **Opportunità 7.3 integrata**: Stato sistema in tempo reale in UI

**Stima**: 4 giorni-uomo

---

#### C12 — Integration Test Suite (G60-78) | **5gg** | MEDIUM

- API integration test: auth → project → query → ingestion → tool execution
- DuckDB+Postgres dual write test
- SSE event flow test
- NLP sidecar integration test
- Multi-step PAORA integration test

**Stima**: 5 giorni-uomo

---

#### C13 — Frontend Polish (G65-80) | **3gg** | MEDIUM, OPPORTUNITÀ 3.6, 3.8, 7.5

- **Opportunità 3.6**: React Query o SWR per caching e dedup fetch
- **Opportunità 3.7**: Bundle splitting fine (vendor chunk 295KB → 150KB); Factory chunk dual import fix
- **Opportunità 3.8**: CSS purge-safe audit; tutte le classi dinamiche via lookup table
- **Opportunità 7.5**: Performance perception: skeleton loader, optimistic UI per operazioni CRUD, debounce ricerca 300ms

**Stima**: 3 giorni-uomo

---

### Riepilogo Effort Fase 3

| Track | Task | Giorni |
|-------|------|--------|
| A | A11 Multi-tenancy (scope ridotto) | 6 |
| A | A12 EU compliance base | 4 |
| B | B10 Multi-step tool execution | 5 |
| B | B11 Tool package (3/6) | 8 |
| B | B12 Memory & VSS enhancement | 3 |
| B | B13 Structured error enrichment | 2 |
| C | C09 Terminal & Chat UI | 3 |
| C | C10 Tool UI completion | 2 |
| C | C11 Dashboard & analytics | 4 |
| C | C12 Integration test suite | 5 |
| C | C13 Frontend polish | 3 |
| **Totale Fase 3** | | **45gg-uomo (40gg calendario)** |

### Rischi Fase 3

| Rischio | Probabilità | Impatto | Mitigazione |
|---------|------------|--------|-------------|
| **API esterne (Finance/OSINT) non disponibili o cambiano** | MEDIUM | MEDIUM | Usare free tier con API key. Fallback a dati mock in development. |
| **PAORA multi-step introduce bug di coordinazione** | MEDIUM | HIGH | Test suite estensiva (B02). Feature flag per rollback. |
| **Multi-tenancy scope insufficiente per produzione** | LOW | MEDIUM | Documentato come "Phase 1 multi-tenancy". Fase 5 per completamento. |

### Ship Gate 3 (G85)

**Check obbligatori**:
- [ ] `go test -race -count=1 ./...` ✅ (0 FAIL)
- [ ] `npx tsc --noEmit` ✅
- [ ] `npx vite build` ✅ (< 3s)
- [ ] Tool packages (finance/osint/he): funzionanti con test
- [ ] PAORA V2 multi-step: test passano (Plan→Act→Observe→Reflect→Admit con 3+ step)
- [ ] UI: tutte le view renderizzano senza errori
- [ ] Integration test suite: > 50 test
- [ ] E2E Playwright: > 20 scenari
- [ ] Backend test coverage > 70%
- [ ] Frontend test coverage > 50%
- [ ] **Security re-check**: OWASP ZAP full scan → 0 CRITICAL, ≤ 3 HIGH

---

## FASE 4: Production Ready (G75-120)

**Obiettivo**: Portare Aleph-v2 in produzione. Security audit, performance, documentazione, release.

**OKR**:
- Security audit: 0 CRITICAL, ≤ 3 HIGH
- Performance: 500 req/s p95 < 500ms
- Documentazione: API ref, user guide (IT/EN), runbook
- v2.0 release tagged

### Track A — Security & Performance (G75-105)

#### A13 — Security Audit Completo (G75-95) | **6gg** | HIGH

- Penetration testing (OWASP Top 10 completo)
- ZAP automated scan → remediation
- gosec senza esclusioni
- Dependency audit: `govulncheck`, `npm audit`, trivy per Docker image
- **Opportunità 5.6 integrata**: Vulnerability scanning continuo in CI
- Bug bounty program setup (documentazione)
- Responsible disclosure policy (`SECURITY.md`)

**Stima**: 6 giorni-uomo

---

#### A14 — Performance Optimization (G80-100) | **5gg** | MEDIUM

- DuckDB: EXPLAIN ANALYZE su query hotspot; indici VSS tuning
- Query optimization: ridurre query ridondanti
- Connessione pooling ottimizzato
- Memory profiling con pprof → fix leak
- Goroutine leak detection → fix
- Target: 500 req/s p95 < 500ms

**Stima**: 5 giorni-uomo

---

#### A15 — Onboarding Zero-Config (G85-98) | **3gg** | OPPORTUNITÀ 7.2

- First-run wizard: configura provider LLM, crea primo agente
- Demo data: 2-3 agenti preconfigurati, 1 datasource fittizio
- Tooltip tour guidato per nuovi utenti
- **Opportunità 7.2 integrata**: Onboarding Zero-Config

**Stima**: 3 giorni-uomo

---

### Track B — Reliability & Polish (G78-105)

#### B14 — DuckDB Backup & Recovery (G78-88) | **2gg** | MEDIUM M15

- Backup con fsync garantito
- Backup schedulato (cron interni o esterni)
- Recovery procedure documentata
- Test restore da backup

**Stima**: 2 giorni-uomo

---

#### B15 — Error Recovery & Resilience (G80-92) | **3gg** | MEDIUM (M13, M14, M18)

- Act: propagare errori reali (non sempre nil)
- Observe: soglia troncamento configurabile (non hardcoded 1900)
- TrustDelta da Engine.Observe: valori reali (non sempre 0)
- validateToolName: restringere pattern matching

**Stima**: 3 giorni-uomo

---

#### B16 — Accessibility Audit (G85-95) | **2gg** | OPPORTUNITÀ 7.4

- Tab order su tutti i form
- Focus trap su modal/slideover
- ARIA labels su icone decorative
- Color contrast su testi piccoli (13px JetBrains Mono)
- Keyboard shortcut per comandi principali
- WCAG 2.1 AA compliance documentata

**Stima**: 2 giorni-uomo

---

### Track C — Documentation & Release (G80-120)

#### C14 — Documentazione Completa (G80-105) | **6gg** | MEDIUM

- API Reference: OpenAPI/Swagger completo da protobuf + ConnectRPC
- User guide: italiano + inglese (workflow principali, concetti)
- Developer onboarding: architettura, setup, come contribuire
- Deployment guide: Docker Compose (primario), K8s (reference)
- Runbook operativi: per ogni subsystem (startup, shutdown, recovery, monitoring)

**Stima**: 6 giorni-uomo

---

#### C15 — Pre-Release Verification (G95-115) | **4gg** | MEDIUM

- TypeScript strict mode: abilitare `strict: true`, fix errori residui
- Bundle size: < 500KB total, chunk splitting ottimizzato
- Performance audit: Lighthouse > 90, React DevTools profiler
- Cross-browser test: Chrome, Firefox, Safari
- Responsive design: mobile/tablet/desktop

**Stima**: 4 giorni-uomo

---

#### C16 — v2.0 Release (G105-120) | **3gg** | FINALE

- CHANGELOG.md completo da commit history
- Version bump: `v2.0.0`
- Git tag + GitHub Release
- Docker image publish
- Release announcement (blog post, social)
- Migration guide v1.x → v2.0 per utenti esistenti

**Stima**: 3 giorni-uomo

---

### Riepilogo Effort Fase 4

| Track | Task | Giorni |
|-------|------|--------|
| A | A13 Security audit completo | 6 |
| A | A14 Performance optimization | 5 |
| A | A15 Onboarding zero-config | 3 |
| B | B14 DuckDB backup & recovery | 2 |
| B | B15 Error recovery & resilience | 3 |
| B | B16 Accessibility audit | 2 |
| C | C14 Documentazione completa | 6 |
| C | C15 Pre-release verification | 4 |
| C | C16 v2.0 release | 3 |
| **Totale Fase 4** | | **34gg-uomo (45gg calendario)** |

### Rischi Fase 4

| Rischio | Probabilità | Impatto | Mitigazione |
|---------|------------|--------|-------------|
| **Penetration test trova nuove vulnerabilità** | MEDIUM | HIGH | Iniziare audit a G75 per avere tempo di remediation. Budget 2gg extra per fix. |
| **Performance target non raggiungibile senza riscritture** | LOW | MEDIUM | Target 500 req/s è conservativo. Se non raggiunto, documentare come known limitation per v2.1. |
| **Accessibilità richiede refactor UI significativo** | LOW | LOW | Scope: audit + fix critici (keyboard nav, contrast). WCAG AAA differito. |

### Ship Gate 4 (G120) — v2.0 Release

**Check obbligatori**:
- [ ] `go test -race -count=1 ./...` ✅ (0 FAIL)
- [ ] `npx tsc --noEmit` ✅ (0 errori)
- [ ] `npx vite build` ✅
- [ ] `npx vitest run` ✅
- [ ] `npx playwright test` ✅ (> 20 scenari)
- [ ] Security audit: 0 CRITICAL, ≤ 3 HIGH
- [ ] Load test: 500 req/s p95 < 500ms
- [ ] Performance: p99 chat < 2s, p99 query < 500ms
- [ ] Test coverage: Go > 70%, Frontend > 50%
- [ ] Documentazione completa: API ref, user guide (IT/EN), runbook
- [ ] Docker image < 500MB
- [ ] CI/CD: build < 10 min
- [ ] Multi-tenancy base funzionante
- [ ] GDPR compliance base verificata
- [ ] **Security re-check finale**: tutti gli scan (ZAP, gosec, trivy, npm audit) → superati
- [ ] `v2.0.0` tagged su GitHub

---

## Riepilogo Effort Totale

| Fase | Giorni-uomo | Giorni calendario | Track parallele |
|------|-------------|-------------------|-----------------|
| **Fase 1**: Safety Net | 49 | 30 | A+B+C |
| **Fase 2**: Stability Engine | 40.5 | 35 | A+B+C |
| **Fase 3**: Feature Completion | 45 | 40 | A+B+C |
| **Fase 4**: Production Ready | 34 | 45 | A+B+C |
| **Totale** | **168.5gg-uomo** | **~120gg calendario** | |

**Nota sul dimensionamento**: 
- 168.5 giorni-uomo distribuiti su 120 giorni calendario richiedono **~1.4 dev full-time equivalenti in media**
- I picchi di parallelismo sono in Fase 1 (1.6 FTE) e Fase 3 (1.1 FTE)
- Con 2 sviluppatori full-time + 1 part-time, il piano è eseguibile in 120 giorni
- Rispetto ai 300gg-uomo/90gg del piano originale, questo è realistico perché:
  1. La compressione è ~1.4x (non 3.3x)
  2. Le dipendenze tra track sono rispettate
  3. Lo scope è ridotto dove le review hanno trovato sottostime (multi-tenancy, type safety, gVisor)

---

## Review Impact Summary

### Oracle — 3 Blocking Conditions

| Condizione | Azione | Dove |
|------------|--------|------|
| **Migration runbook per A02 auth rewrite** | Aggiunto pre-requisito esplicito in A02: runbook G5-7 prima del codice. | Fase 1, A02 |
| **Sequencing A03 SQL injection prima di B01 integration** | A03 completato entro G5. B01 parte G3 ma integration test iniziano G10 (dopo A03). | Dipendenze Fase 1 |
| **Validare gVisor feasibility G1-3 o commit container-only** | Spike tecnico G1-3. Decisione G3. Container-only è il default. | Fase 1, A01 |

**Altri finding Oracle incorporati**:
- DuckDB txWrapper già implementato → B04 verifica, non ricostruisci (risparmio 2gg)
- DefaultReflector dead code → B01 lo cabla (non riscrive)
- nil Provider degraded mode → B01 fix con provider esplicito
- HealthChecker context leak → B07 fix
- C03 Type Safety sprint ricalibrato 20-25gg → 10gg (scope ridotto a file produzione)

### Momus — Plan Critic (Punteggio: 6.8/10)

| Rischio/Suggerimento | Azione | Dove |
|---------------------|--------|------|
| **G1-30 staffing density (12 task concorrenti)** | Fase 1 task distribuiti: Track B parte G3, Track C task non-bloccanti partono G1. | Struttura Fase 1 |
| **gVisor dependency senza fallback** | Container-only è il default. gVisor validato o abbandonato entro G3. | Fase 1, A01 |
| **PAORA unification scope** | Staged: B01 cabla Reflect + Plan-Act. B10 multi-step in Fase 3. | Fase 1 B01, Fase 3 B10 |
| **A03 line numbers stale** | Rimosso riferimenti a numeri di linea. Usati nomi funzione. | Fase 1, A03 |
| **B05/C06 cross-track duplication** | C06 rimosso (B02 copre PAORA test). C06 ora è backend test coverage generale. | Fase 2, C06 |
| **A14 load test target mismatch (1000 vs 500 req/s)** | Target unificato a 500 req/s in tutto il piano. | Ship Gate 4 |
| **Add migration gap 001→003 task** | Non applicabile (migration gap era in un piano precedente, non in questo). | N/A |
| **Add rollback strategy per ship gate** | Rollback actions aggiunte a ogni Ship Gate. | Ship Gate 1-4 |

### Metis — Plan Consultant (Verdetto: CONDITIONAL)

| Gap | Azione | Dove |
|-----|--------|------|
| **#1: No Day 0 baseline** | Aggiunto A00 implicito: metriche attuali documentate in Executive Summary. Build state verificato. | Executive Summary |
| **#2: No security re-check F1→F4** | Security re-check obbligatorio in OGNI ship gate (ZAP scan, gosec). | Ship Gates 1-4 |
| **#3: A09 conflates 3 deployment options** | Splittato in A09a (Docker Compose), A09b (K8s docs), A09c (bare-metal guide). | Fase 2, A09 |
| **#4: No LLM cost controls** | Nuovo task B04.5: LLM budget circuit breaker. Integrato con dashboard C11. | Fase 2, B04.5 |
| **#5: No NLP sidecar dead man's switch** | B08: NLP watchdog con auto-restart, max 3 tentativi, alerting. | Fase 2, B08 |
| **#6: No rollback procedures** | Aggiunte a ogni ship gate. | Ship Gates 1-4 |
| **#7: A11 multi-tenancy under-scoped 2-3x** | Scope ridotto del 60%. Resource quotas differiti a backlog. | Fase 3, A11 |
| **#8: Load testing only at G75** | Spostato a Fase 2 (G40-55) per validare fix DuckDB/rate limiter subito. | Fase 2, A10 |

**Hidden intentions risolti**:
- "300gg/90gg compressione irrealistica" → Ricalibrato a 168.5gg/120gg (1.4x, non 3.3x)
- "Non deciso deployment contraddetto da A09 K8s spec" → Docker Compose primario; K8s documentato (non implementato)
- "Priorità reale è reliability non features" → Fase 1 e 2 sono 100% safety/reliability. Features solo in Fase 3.

---

## Opportunità Integrate vs Differite

### Integrate nel piano (7)

| Opportunità | Dove | Priorità | Effort |
|-------------|------|----------|--------|
| 2.1 Reflection Engine Unificato | Fase 1, B01 | CRITICAL | Incluso in 5gg |
| 2.2 Plan-Act Connector | Fase 1, B01 | CRITICAL | Incluso in 5gg |
| 2.6 DuckDB VSS First-Class | Fase 3, B12 | MEDIUM | 3gg |
| 2.7 Structured Error Enrichment | Fase 3, B13 | MEDIUM | 2gg |
| 3.2 AbortController Pattern | Fase 1, C01 | CRITICAL | 2gg |
| 6.1 NLP Watchdog Restart | Fase 2, B08 | HIGH | 2gg |
| 7.1 Errori Human-Readable | Fase 1, C04 | MEDIUM | Incluso in 2gg |

### Nel backlog (Fase 5, post-v2.0) — 9 opportunità

| Opportunità | Priorità | Effort stimato |
|-------------|----------|---------------|
| 3.7 Bundle Splitting Perfetto | LOW | 1gg |
| 3.8 CSS Purge-Safe Audit | MEDIUM | 1gg |
| 4.7 Docker Secrets Pattern | MEDIUM | 0.5gg |
| 5.5 Fuzzing & Property-Based Test | MEDIUM | 2gg |
| 5.6 Vulnerability Scanning Continuo | MEDIUM | 1gg |
| 6.3 NLP Model Caching | MEDIUM | 1gg |
| 7.5 Performance Perception | MEDIUM | 1gg |
| 8.3 Embedding Caching | LOW | 1gg |
| 8.4 Tool Suggestion ML | LOW | 3gg |

### Differite a Fase 5 per vincoli di scope (6)

| Item | Motivo |
|------|--------|
| gVisor sandbox (L3) | Complessità kernel-dipendente. Container-only è sufficiente per v2.0. |
| K8s deployment completo | Posticipato a post-v2.0. Documentazione in A09b. |
| Resource quotas enforcement | Tagliato da A11 per Metis feedback. |
| Tool packages (3/6 stub) | Differiti adaptation, codegen, analytics. |
| GNN advanced training pipeline | Training base in B03. Online fine-tuning differito. |
| LLM fallback chain (ollama→anthropic→openai) | Nice-to-have. Non blocking per v2.0. |

---

## Dipendenze tra Track

```
TRACK A (Security/Infra)
  A03 SQL injection ──┐
  A01 Sandbox         ├──→ B01 PAORA core (G3)
  A02 Auth ───────────┘
  A04 Secrets
  A05 Network
      │
      ├──→ A06 CI/CD ──→ A09 Deploy ──→ A11 Multi-tenancy ──→ A13 Audit
      │                   A08 Alert   A12 Compliance    A14 Perf
      │                   A10 Load                     A15 Onboarding
      
TRACK B (Decision Engine)
  B01 PAORA core ──→ B02 Test suite ──→ B04 DuckDB ──→ B10 Multi-step ──→ B15 Recovery
  B03 GNN                B04.5 LLM budget  B05 Rate limit  B11 Tools       B14 Backup
                                           B06 LLM robust  B12 VSS         B16 a11y
                                           B07 MCP disc    B13 Errors
                                           B08 NLP watch
                                           B09 Validation

TRACK C (Frontend)
  C01 AbortCtrl ──→ C03 Types ──→ C06 Backend tests ──→ C09 Chat UI ──→ C12 Integr. tests
  C02 Auth fix      C04 Errors     C07 Contract        C10 Tool UI     C13 Polish
                    C05 State      C08 FE tests         C11 Dashboard   C14 Docs
                                                                        C15 Verify
                                                                        C16 Release
```

**Regole di sequenziamento**:
1. Track B parte G3 (dopo A03 SQL injection completato)
2. Track C parte G1 (tasks indipendenti: AbortController, auth fix)
3. Fase 2 Track B dipende da B01-B03 completati
4. Fase 3 Track B dipende da B04-B09 completati
5. Fase 4 è indipendente (tutte le track convergono sulla release)

---

## Metriche Obiettivo (OKR Finali)

| KPI | Target v2.0 | Misurabile via |
|-----|------------|----------------|
| Vulnerabilità CRITICAL | 0 | Security audit |
| Vulnerabilità HIGH | ≤ 3 | OWASP ZAP |
| Test coverage Go | > 70% | `go test -cover ./...` |
| Test coverage Frontend | > 50% | `vitest run --coverage` |
| TS strict errors | 0 | `npx tsc --noEmit` |
| Build time CI | < 10 min | GitHub Actions |
| P99 latency chat | < 2s | k6 load test |
| P99 latency query | < 500ms | k6 load test |
| Requests/sec | > 500 | k6 load test |
| Sandbox escape | 0 | Penetration test |
| Uptime | 99.9% | Prometheus/Alertmanager |
| Docker image size | < 500MB | `docker images` |
| Integration tests | > 50 | `go test ./...` |
| E2E scenarios | > 20 | Playwright |
| PAORA tests | > 50 | `go test ./decision/...` |

---

*Piano generato il 1 Maggio 2026. Integra 4 documenti sorgente + 3 review indipendenti. Versione 1.0.*
