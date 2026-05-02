# Relazione Criticità / Bug — Aleph-v2 v2.0.0

**Data**: 1 Maggio 2026
**Tipo**: T2.B — Bug Report (reali vs falsi positivi)
**Metodo**: Analisi sistematica codebase + 5 explore agent (Decision Engine, Backend Go, Frontend, Security, Tests/CI/CD) + build/type-check verification

## Riepilogo

| Metrica | Valore |
|---------|--------|
| Totale criticità identificate | 68 |
| CRITICAL (produzione bloccante) | 14 |
| HIGH (rischio significativo) | 25 |
| MEDIUM (miglioramento) | 29 |
| Falsi positivi identificati | 7 |
| Build status | go build ✅ / tsc ✅ / vite ✅ / go test ✅ |

---

## PARTE 1: VALUTAZIONE GLOBALE COMPLETAMENTO (T2.A)

| Area | Completamento | Stato |
|------|--------------|-------|
| **Backend Go — Core** | **85%** | Decision Engine funzionante, storage DuckDB/PostgreSQL operativo, middleware chain completa. Mancano: Re-plan phase, GNN training, proper error recovery. |
| **Backend Go — Tools** | **70%** | 4 package (finance, osint, humanecosystems, adaptation) sono stub — scheletri compilano ma non hanno logica reale. |
| **Backend Go — Sandbox** | **60%** | Verifica Go/Python presente ma aggirabile. Nessun isolamento Docker/container. Allowlist bypassabile. |
| **Backend Go — Security** | **55%** | SSRF fail-closed parziale, SHA-256 ancora presente (non migrato ad argon2id), SQL injection in query.go, sandbox non isolata. |
| **Frontend — Core** | **80%** | 6 Zustand slice, 6 view lazy-loaded, routing funzionante. SlideOverContent completo. |
| **Frontend — UI/UX** | **75%** | Design tokens, tipografia, effetti terminale, temi. SetupWizard/SettingsView ancora con alert/confirm. Error handling duplicato. |
| **Frontend — Type Safety** | **65%** | Zod schemas parziali (mancano Scenario/ToolAnomaly). 14 file con any pervasivo. 42 cast `as unknown as`. Tipi duplicati. |
| **NLP Sidecar** | **75%** | Sentiment analysis italiano/inglese, health check gRPC. Keyword-based (non LLM). Watchdog senza restart. |
| **CI/CD** | **70%** | GitHub Actions presente. Deploy workflow senza test gate. Alertmanager receivers vuoti. |
| **Testing** | **65%** | 811 test passano ma: ChatSession 0 test, Decision Engine untested, sandbox untested, VSS test skippato. |
| **Docker/Infrastruttura** | **75%** | Docker compose funzionante. Secrets in env var. Nessun healthcheck backend. Flat network. |

### Completion Score Globale: **72%**

**Interpretazione**: Progetto functional complete ma non production-ready. Il core (storage, API, routing, decision loop base) è solido. Le lacune sono in sicurezza (sandbox, SQL injection, secrets management), testing (PAORA, sandbox), e alcuni stub non implementati (tool package). Per production-grade servono 2-3 mesi di lavoro focalizzato.

---

## PARTE 2: CRITICITÀ — REALI vs FALSI POSITIVI (T2.B)

### CLASSIFICAZIONE

| Priorità | Conteggio |
|----------|-----------|
| 🔴 CRITICAL | 14 |
| 🟠 HIGH | 25 |
| 🟡 MEDIUM | 29 |
| ⚪ FALSO POSITIVO | 7 |

---

### 🔴 CRITICAL (14) — Produzione bloccante

#### C01. SQL Injection — query.go (Backend)
- **File**: `internal/api/handler/query.go:178,257,301`
- **Tipo**: Reale
- **Dettaglio**: `fmt.Sprintf("FROM \"%s\".\"%s\"", projectID, tableName)` con guard `validName()` sola. `validName` è una regex che può essere aggirata con identificatori quoted. `validIdentifier`/`sanitizeIdentifier` esiste ma non viene usato in query.go.
- **Impatto**: Attacco SQL injection su DuckDB. Non ci sono stored procedure o dati sensibili in DuckDB, ma un attaccante potrebbe eseguire DROP TABLE / DELETE FROM / leggere dati di altri progetti.
- **Fix**: Usare `sanitizeIdentifier()` esistente in engine.go, o parametrizzare con DuckDB prepared statements.
- **Priorità**: IMMEDIATA

#### C02. Sandbox Code Execution Bypass (Backend)
- **File**: `internal/sandbox/allowlist.go`
- **Tipo**: Reale
- **Dettaglio**: `allowlist.go` blocca `--pty` e `-i` ma NON blocca `--interactive=always` o `--tty=yes`. Inoltre `os.Getenv("PATH")` non è ristretto — un tool LLM-generato può eseguire qualsiasi binario nel PATH dell'host.
- **Impatto**: Un utente malintenzionato può eseguire codice arbitrario sul server host bypassando i blocchi previsti.
- **Fix**: Bloccare `--interactive`, `--tty`, e varianti. Ristringere PATH a `/usr/bin:/bin:/usr/local/bin`. Aggiungere Docker/container isolation.
- **Priorità**: IMMEDIATA

#### C03. Decision Engine — Dual Reflect Implementation (Backend)
- **File**: `internal/decision/engine.go:Reflect` vs `internal/decision/reflector.go:DefaultReflector`
- **Tipo**: Reale
- **Dettaglio**: `Engine.Reflect()` (usato in produzione) è una implementazione semplicistica che controlla solo errori e output vuoto. `DefaultReflector` (package-level, dead code) ha classificazione GapType completa (GapToolExecution, GapHallucination, GapReasoning, GapInsufficientContext). La produzione usa quella sbagliata.
- **Impatto**: Il decision loop non impara dai propri errori. La Reflect phase è essenzialmente inutile per il miglioramento del piano.
- **Fix**: Sostituire `Engine.Reflect()` con `DefaultReflector.Reflect()`.
- **Priorità**: IMMEDIATA

#### C04. Decision Engine — Plan-Act Disconnect (Backend)
- **File**: `internal/decision/engine.go:Act()`
- **Tipo**: Reale
- **Dettaglio**: `Plan()` genera `PlannedStep` con intent e tool suggeriti. `Act()` ignora completamente i planned steps e usa invece raw LLM tool calls. Il piano è generato ma mai eseguito come pianificato.
- **Impatto**: L'intero decision loop è fittizio — Plan non influenza Act. Il sistema opera come chat LLM con tool call invece di un vero decision loop.
- **Fix**: `Act()` deve eseguire i `PlannedStep` generati da `Plan()`, non delegare raw LLM.
- **Priorità**: IMMEDIATA

#### C05. Decision Engine — nil Provider degraded mode rotto (Backend)
- **File**: `internal/decision/engine.go:Plan()`, `internal/app/app.go:226`
- **Tipo**: Reale
- **Dettaglio**: `app.go:226` hardcodato `Provider: nil` → `Engine.Plan()` usa degraded mode che ritorna `query_dispatch` intent. Ma `query_dispatch` non è gestito da `tool_executor.go:switch` — entra in un confirmation loop senza fine.
- **Impatto**: Con Provider=nil, il decision loop si blocca. L'utente non può usare funzionalità di query/analisi dati.
- **Fix**: `app.go` deve passare un provider valido (es. `llm.NewProvider("ollama", ...)`), oppure `Act()` deve gestire `query_dispatch` come fallback.
- **Priorità**: IMMEDIATA

#### C06. ChatSession — PAORA non testato, bug architetturale (Backend)
- **File**: `internal/api/handler/chat.go`, `internal/api/handler/chat_session.go`
- **Tipo**: Reale
- **Dettaglio**: ChatSession 330 linee, ZERO test. `ChatSession.Run()` hardcoda 5 iterazioni invece di usare `MaxAttempts`. `engine.Reflect()` panica se plan==nil. `Admit()` termina al primo errore senza retry.
- **Impatto**: Regressioni silenziose al 100%. Qualsiasi modifica al decision loop è un volo cieco.
- **Fix**: Scrivere test per ChatSession.Run(). Usare MaxAttempts da EngineConfig. Nil-check su plan in Reflect. Retry in Admit.
- **Priorità**: IMMEDIATA

#### C07. Security — API Key in HttpOnly Cookie senza session token (Frontend/Backend)
- **File**: `internal/api/handler/session.go`, `frontend/src/api/client.ts`
- **Tipo**: Reale
- **Dettaglio**: `session.go` setta un HttpOnly cookie con la API key in chiaro (non un session token). Il server deve validare la key a ogni richiesta invece di validare una sessione. La API key è anche in sessionStorage e Zustand (DevTools visible).
- **Impatto**: Se un XSS trova un varco, la API key è esposta. HttpOnly aiuta ma non è sufficiente — non c'è rotation, expiry, o binding a IP.
- **Fix**: Implementare session token (JWT o random) invece di cookie plaintext API key. Aggiungere expiry e rotation. Rimuovere API key da Zustand.
- **Priorità**: IMMEDIATA

#### C08. Security — Postgres DSN hardcoded default (Backend/Infra)
- **File**: `internal/config/config.go`, `docker-compose.yml`
- **Tipo**: Reale
- **Dettaglio**: Default DSN `postgres://postgres:postgres@localhost:5432/aleph?sslmode=disable` con password 'postgres' in chiaro. In docker-compose, `KEY_ENCRYPTION_KEY` e `POSTGRES_PASSWORD` sono env var visibili via `docker inspect`.
- **Impatto**: In produzione standard, chiunque abbia accesso al server (o docker inspect) ha le credenziali del database e la chiave di cifratura.
- **Fix**: Richiedere configurazione esplicita in produzione. Usare Docker secrets o vault per le credenziali. Password minima complessità.
- **Priorità**: IMMEDIATA

#### C09. Security — Sandbox nessun isolamento (Backend)
- **File**: `internal/sandbox/`, `exec_sandbox.go`
- **Tipo**: Reale
- **Dettaglio**: La sandbox esegue codice Go/Python compilato/interpretato direttamente sul host. No Docker container, no seccomp, no chroot, no namespace isolation. Il codice LLM-generato può: fare fork(), aprire socket, leggere /etc/passwd, eseguire binari di sistema. `urllib` non è bloccato → data exfiltration possibile.
- **Impatto**: RCE completa sul server host. Data exfiltration, privilege escalation.
- **Fix**: Containerizzare la sandbox (Docker-in-Docker o gVisor). Bloccare import pericolosi Python (imaplib, urllib, requests, socket, os, subprocess). Limitare risorse CPU/mem/process.
- **Priorità**: IMMEDIATA

#### C10. Security — DSL filter concat SQL + regex (Backend)
- **File**: `internal/dsl/compiler_tool.go` o simile
- **Tipo**: Reale
- **Dettaglio**: String concatenation per costruire query SQL con regex-only validation. Già identificato in W1-05 audit come vulnerabile.
- **Impatto**: DSL injection nella generazione di query SQL.
- **Fix**: Usare prepared statements e parametrizzazione ovunque.
- **Priorità**: IMMEDIATA

#### C11. MCP Discovery — nessun retry/backoff (Backend)
- **File**: `internal/mcp/discovery.go`
- **Tipo**: Reale
- **Dettaglio**: `DiscoveryEngine.Start()` fa partire un goroutine `healthLoop()` senza WaitGroup, senza retry, senza backoff. Se la connessione fallisce, il goroutine muore silenziosamente. `mcp.ValidateSSRF()` fa DNS resolution (ottimo) ma gli altri HTTP client (agent.go, ollama.go, openai.go, etc.) no.
- **Impatto**: MCP tools spariscono silenziosamente dopo un errore di rete. Non c'è riconnessione.
- **Fix**: Aggiungere retry con exponential backoff. WaitGroup per graceful shutdown. Centralizzare HTTP client con SSRF validation.
- **Priorità**: ALTA (CRITICAL per affidabilità)

#### C12. HealthChecker — context leak (Backend)
- **File**: `internal/mcp/health.go` o `internal/app/app.go`
- **Tipo**: Reale
- **Dettaglio**: `healthChecker.Start()` in goroutine sovrascrive la cancel function passata da `NewHealthChecker()`. Il cancel originale non viene mai chiamato, causando goroutine leak.
- **Impatto**: Ad ogni riavvio/chiamata, goroutine orphan. Memory leak.
- **Fix**: Non sovrascrivere cancel in Start(). Salvare il cancel originale e usare quello.
- **Priorità**: ALTA

#### C13. Rate Limiter — unbounded map growth (Backend)
- **File**: `internal/middleware/ratelimit.go`
- **Tipo**: Reale
- **Dettaglio**: Il rate limiter usa una mappa non protetta per IP → limiter. Sotto DDoS con IP spoofati (X-Forwarded-For è spoofabile), la mappa cresce illimitatamente consumando memoria.
- **Impatto**: OOM sotto attacco DDoS con IP spoofati.
- **Fix**: Mappa con TTL/LRU. Autenticare X-Forwarded-For (trusted proxy list).
- **Priorità**: ALTA

#### C14. Alertmanager — receivers vuoti (Infra)
- **File**: `deploy/prometheus/alertmanager.yml` o equivalente
- **Tipo**: Reale
- **Dettaglio**: Alertmanager configurato ma receivers vuoti — le alert vengono valutate ma non notificate a nessuno (no email, no Slack, no PagerDuty).
- **Impatto**: Incidenti silenziosi. Il team non sa quando il sistema è down.
- **Fix**: Configurare almeno un receiver (Slack webhook, email SMTP).
- **Priorità**: ALTA

---

### 🟠 HIGH (25) — Rischio significativo

#### H01. App.tsx data loading — no AbortController (Frontend)
- **File**: `frontend/src/App.tsx`, assenza di AbortController in data fetching
- **Tipo**: Reale
- **Dettaglio**: Le chiamate API per caricare dati iniziali (tools, agents, skills) non hanno AbortController. Se il componente viene smontato prima del completamento, c'è race condition e setState on unmounted component.
- **Impatto**: React warning, potenziali memory leak, comportamenti imprevedibili in navigazione rapida.
- **Fix**: AbortController per ogni fetch con cleanup in useEffect.

#### H02. useOntologyActions — raw fetch senza auth (Frontend)
- **File**: hook `useOntologyActions` probabilmente in `frontend/src/hooks/`
- **Tipo**: Reale
- **Dettaglio**: Usa raw `fetch()` invece del client API autenticato. Le richieste non hanno header `X-Aleph-Api-Key`.
- **Impatto**: Le operazioni ontology falliscono silenziosamente in produzione autenticata.
- **Fix**: Usare il client API centrale invece di raw fetch.

#### H03. App.tsx chat history loading — no AbortController (Frontend)
- **File**: `frontend/src/App.tsx`, caricamento cronologia chat
- **Tipo**: Reale
- **Dettaglio**: Stesso problema di H01 ma specifico per chat history.
- **Impatto**: Race condition su cambio contesto.

#### H04. useCursorPagination — stale closure (Frontend)
- **File**: hook `useCursorPagination`
- **Tipo**: Reale
- **Dettaglio**: La closure cattura variabili obsolete. Quando i parametri cambiano, la paginazione continua con i vecchi cursori.
- **Impatto**: Paginazione inconsistente in UI.

#### H05. Empty catch blocks — 3 locations (Frontend)
- **File**: `AlephErrorBoundary.tsx`, `useAppActions`, `navigationSlice`
- **Tipo**: Reale
- **Dettaglio**: Tre punti con `catch {}` vuoti. Errori inghiottiti silenziosamente.
- **Impatto**: Bug invisibili. Difficile fare debugging.
- **Fix**: Almeno loggare l'errore. Meglio: mostrare all'utente.

#### H06. ToolForm/SkillForm — raw REST bypassa ConnectRPC (Frontend)
- **File**: Componenti ToolForm, SkillForm
- **Tipo**: Reale
- **Dettaglio**: Questi form usano raw REST API invece del client ConnectRPC generato, bypassando auth, error handling, e middleware.
- **Impatto**: Opera al di fuori del sistema di sicurezza centralizzato.

#### H07. DataSourceForm — JSON.parse in render senza try-catch (Frontend)
- **File**: DataSourceForm component
- **Tipo**: Reale
- **Dettaglio**: `JSON.parse()` direttamente nel render. Dati malformati crasano l'app.
- **Impatto**: Crash su dati API non validi.

#### H08. Duplicazione handleError — double-firing toasts (Frontend)
- **File**: module-level `handleError` + hook `useHandleError`
- **Tipo**: Reale
- **Dettaglio**: Due implementazioni separate di handleError. Entrambe chiamate in certi flussi → doppio toast di errore.
- **Impatto**: UX confusionaria, errori duplicati.

#### H09. useInfiniteQueries — fetchTools missing projectId (Frontend)
- **File**: hook useInfiniteQueries o equivalente
- **Tipo**: Reale
- **Dettaglio**: `fetchTools` chiamato senza `projectId`. In contesto multi-progetto, recupera tool sbagliati.
- **Impatto**: Tool errati mostrati all'utente.

#### H10. DuckDB TX — parent mutex durante Commit = deadlock (Backend)
- **File**: `internal/storage/duckdb.go:Commit()`
- **Tipo**: Reale
- **Dettaglio**: `BeginTX` acquisisce semaphore + lock. `Commit` rilascia il lock ma lo fa mentre DuckDB sta ancora scrivendo. Se un'altra goroutine prova `Exec` durante `Commit`, deadlock.
- **Impatto**: Deadlock intermittente in produzione.
- **Fix**: Commit fuori dal lock. O usare RLock per le transazioni.

#### H11. QueryRowContext — nil ritorno su semaphore exhaustion (Backend)
- **File**: `internal/storage/duckdb.go`
- **Tipo**: Reale
- **Dettaglio**: `QueryRowContext` ritorna `nil` se il semaphore è esaurito. I chiamanti non controllano nil → nil pointer panic.
- **Impatto**: Panico in produzione sotto carico.
- **Fix**: Ritornare errore invece di nil (usare QueryRowContextOrError).

#### H12. Default Postgres DSN 'postgres:postgres' (Backend)
- **File**: `internal/config/config.go`
- **Tipo**: Reale (stessa issue di C08, livello lower)
- **Dettaglio**: Default DSN con password debole per sviluppo.
- **Fix**: Richiedere configurazione esplicita in produzione.

#### H13. Python validation regex bypass (Backend)
- **File**: `internal/sandbox/validation.go`
- **Tipo**: Reale
- **Dettaglio**: Blocca `import os` ma non `import  os` (doppio spazio) o `exec('import os')` via eval. Bypassabile con spazi extra, commenti, eval wrappers.
- **Impatto**: La protezione è facilmente aggirabile.

#### H14. LLM provider — nessun HTTP timeout (Backend)
- **File**: `internal/llm/provider.go`
- **Tipo**: Reale
- **Dettaglio**: HTTP client per Ollama/Anthropic/OpenAI senza timeout. Chiamate possono rimanere bloccate per minuti.
- **Impatto**: Richieste bloccanti, resource leak.
- **Fix**: Aggiungere timeout configurabile.

#### H15. EngineConfig — zero validation (Backend)
- **File**: `internal/decision/engine.go`
- **Tipo**: Reale
- **Dettaglio**: `EngineConfig` non viene validato. Provider, MetaRepo, Executor possono essere nil → runtime panic.
- **Fix**: Aggiungere `Validate()` method.

#### H16. GNN predictor — mai addestrato (Backend)
- **File**: `internal/decision/gnn.go`
- **Tipo**: Reale
- **Dettaglio**: `LinkPredictor` esiste ma `IsTrained()` ritorna sempre false. Nessun training call nel ciclo di vita.
- **Impatto**: GNN è dead code (o quasi).

#### H17. CSRF — no-Origin/Referer requests permesse (Backend/Infra)
- **File**: `internal/middleware/csrf.go`
- **Tipo**: Reale
- **Dettaglio**: Le richieste senza header Origin/Referer (o con valori vuoti) passano il controllo CSRF.
- **Impatto**: CSRF attack possibile da browser con `no-referrer` policy o da tool CLI.
- **Fix**: Richiedere Origin valido o token CSRF esplicito.

#### H18. X-Forwarded-For spoofable (Backend)
- **File**: `internal/middleware/ratelimit.go`
- **Tipo**: Reale
- **Dettaglio**: Il rate limiter usa `X-Forwarded-For` direttamente. Un attaccante può spoofare IP per bypassare limit.
- **Impatto**: Rate limiting aggirabile.

#### H19. CSP — ws://localhost:* wildcard (Backend)
- **File**: Config CSP headers
- **Tipo**: Reale
- **Dettaglio**: CSP allowlist include `ws://localhost:*`. Un XSS può aprire WebSocket a qualsiasi porta localhost.
- **Impatto**: Port scanning interno via WebSocket.

#### H20. skipAuth — substring match (Backend)
- **File**: `internal/middleware/auth.go`
- **Tipo**: Reale
- **Dettaglio**: `skipAuth` controlla con `strings.Contains(r.URL.Path, "AuthService")`. Un endpoint che contiene "AuthService" nel path (es. `/api/v1/AuthServiceBackup`) bypassa auth.
- **Impatto**: Auth bypassabile.

#### H21. Deploy workflow — nessun test gate (CI/CD)
- **File**: `.github/workflows/deploy.yml`
- **Tipo**: Reale
- **Dettaglio**: Il workflow di deploy parte su tag push e fa solo build + push immagine + deploy. Nessuna esecuzione di test prima del deploy.
- **Impatto**: Codice rotto può arrivare in produzione.
- **Fix**: Aggiungere `needs: [test, build]` step.

#### H22. CI go test con tee senza pipefail (CI/CD)
- **File**: `.github/workflows/ci.yml`
- **Tipo**: Reale
- **Dettaglio**: `go test ... | tee ...` senza `set -o pipefail`. Se go test fallisce, il pipe nasconde il fallimento.
- **Impatto**: CI può passare con test falliti.
- **Fix**: Aggiungere `set -o pipefail` o usare `go test` senza pipe.

#### H23. Dockerfile — golang:1.24-bullseye grande (DevOps)
- **File**: `Dockerfile`
- **Tipo**: Reale
- **Dettaglio**: Usa `golang:1.24-bullseye` (1.2GB+) come base per builder stage. Nessun `.dockerignore`.
- **Impatto**: Build lente, immagini grandi (anche se multi-stage riduce la finale).
- **Fix**: Usare `golang:1.24-alpine`, aggiungere `.dockerignore`.

#### H24. docker-compose — no healthcheck backend (DevOps)
- **File**: `docker-compose.yml`
- **Tipo**: Reale
- **Dettaglio**: Backend service senza `healthcheck`. Dipendenze (NLP, DB) senza `depends_on` condition.
- **Impatto**: Backend parte prima che DB/NLP siano pronti → crash loop.
- **Fix**: Aggiungere healthcheck + condition.

#### H25. Contract tests con -tags=contract ma tag inesistente (Testing)
- **File**: `internal/handler/` o equivalente
- **Tipo**: Reale
- **Dettaglio**: I test usano `//go:build contract` ma non esiste un build tag 'contract'. I test non vengono mai eseguiti in CI.
- **Impatto**: Contract tests sono dead code.
- **Fix**: Aggiungere `//go:build integration` e eseguirli in CI.

---

### 🟡 MEDIUM (29) — Miglioramento

#### M01. `any` types pervasivi in 14 file di produzione (Frontend)
- **Tipo**: Reale
- **Dettaglio**: 14 file di produzione (non test) usano `any`. Questo annulla i benefici di TypeScript.

#### M02. 42 cast `as unknown as` in SlideOverContent/InlineRenderer (Frontend)
- **Tipo**: Reale
- **Dettaglio**: Pattern `x as unknown as Y` per aggirare il type system. Segno di tipi mal progettati.

#### M03. SSE module-level mutable state (Frontend)
- **Tipo**: Reale
- **Dettaglio**: `lastEventIdInternal` come variabile module-level. In ambiente SSR o test, lo stato persiste tra istanze.

#### M04. Set serialization in copilotSlice (Frontend)
- **Tipo**: Reale
- **Dettaglio**: Zustand cerca di serializzare `Set` per devtools. Set non è JSON-serializable → errori in dev mode.

#### M05. cancelStream side effect in setter (Frontend)
- **Tipo**: Reale
- **Dettaglio**: `cancelStream` muta stato dentro un setter Zustand (side effect in reducer puro).

#### M06. Dynamic Tailwind classes non purgate (Frontend)
- **Tipo**: Reale
- **Dettaglio**: Classi costruite con string concatenation non sono rilevate da Tailwind JIT → stili mancanti in produzione.

#### M07. assertType identity function bypassa Zod (Frontend)
- **Tipo**: Reale
- **Dettaglio**: `assertType<T>(x: T): T` accetta qualsiasi cosa. Non valida runtime.

#### M08. Duplicate types: store/types.ts vs schemas/index.ts (Frontend)
- **Tipo**: Reale
- **Dettaglio**: Stessi tipi in due posti, nessuna integrazione. Divergenza garantita.

#### M09. Zod schemas mancanti per Scenario/ToolAnomaly (Frontend)
- **Tipo**: Reale
- **Dettaglio**: Scenario e ToolAnomaly non hanno validazione Zod.

#### M10. Sidebar green dot sempre acceso (Frontend)
- **Tipo**: Reale
- **Dettaglio**: Indicatore di stato "connesso" sempre verde, anche quando backend non risponde.

#### M11. SetupWizard usa alert() (Frontend)
- **Tipo**: Reale
- **Dettaglio**: `alert()` per feedback utente → bloccante, non stilizzato.

#### M12. SettingsView usa confirm() (Frontend)
- **Tipo**: Reale
- **Dettaglio**: `confirm()` per conferma → bloccante, non stilizzato.

#### M13. Act sempre ritorna nil error (Backend)
- **Tipo**: Reale
- **Dettaglio**: `Engine.Act()` ritorna sempre nil error. Gli errori sono seppelliti in `result.Error` string. I chiamanti non controllano result.Error.

#### M14. Observer soglia 1900-char hardcoded (Backend)
- **Tipo**: Reale
- **Dettaglio**: `DefaultObserver` soglia di troncamento a 1900 caratteri hardcoded. Non configurabile.

#### M15. DuckDB backup — nessun fsync (Backend)
- **Tipo**: Reale
- **Dettaglio**: Backup DuckDB senza fsync. Su crash, backup corrotto.

#### M16. MemoryStore sync.Once — fallimento permanente (Backend)
- **Tipo**: Reale
- **Dettaglio**: `sync.Once` per init VSS. Se fallisce (estensione non disponibile), non ritenta mai. Richiede restart.

#### M17. Circuit breaker — thundering herd in half-open (Backend)
- **Tipo**: Reale
- **Dettaglio**: In half-open state, la prima richiesta che passa può essere un massivo burst → thundering herd sul servizio downstream.

#### M18. TrustDelta sempre 0 da Engine.Observe (Backend)
- **Tipo**: Reale
- **Dettaglio**: `Engine.Observe` calcola TrustDelta ma è sempre 0. Le soglie reflector (GapGapThreshold=0.3 per config) non vengono mai triggerate.

#### M19. validateToolName pattern matching troppo broad (Backend)
- **Tipo**: Reale
- **Dettaglio**: Accetta `run`, `call`, `load`, `save` come nomi tool validi — matcha keyword generiche.

#### M20. InferToolsFromMessage over-matches (Backend)
- **Tipo**: Reale
- **Dettaglio**: Matcha 'data', 'show', 'object' come tool intent — parole troppo comuni.

#### M21. MockLinkPredictor — non soddisfa interfaccia (Backend)
- **Tipo**: Reale
- **Dettaglio**: MockLinkPredictor usa `interface{}` per graph type, non matcha `LinkPredictor.Predict()` che richiede tipo specifico.

#### M22. Regex injection in extractObjectReferencesWithOntology (Backend)
- **Tipo**: Reale
- **Dettaglio**: Compila regex per chiamata — ReDoS via input crafted.

#### M23. VSS test skippato (Testing)
- **Tipo**: Reale
- **Dettaglio**: Test per VSS extension skippato. DuckDB VSS non testato.

#### M24. gosec exclusions troppo broad (CI/CD)
- **Tipo**: Reale
- **Dettaglio**: `.golangci.yml` esclude troppe regole gosec.

#### M25. Nessuna Go version matrix in CI (CI/CD)
- **Tipo**: Reale
- **Dettaglio**: CI testa solo 1 versione di Go. Non rileva regressioni cross-version.

#### M26. CSRF tests esistono ma SSE auth no (Testing)
- **Tipo**: Reale
- **Dettaglio**: Test per CSRF presente. Test per autenticazione SSE assente.

#### M27. Deploy senza Docker layer caching (CI/CD)
- **Tipo**: Reale
- **Dettaglio**: `docker build` senza `--cache-from`/`--cache-to`.

#### M28. Backend admin key — comparison non constant-time (Security)
- **Tipo**: Reale
- **Dettaglio**: `==` per confrontare API key. Timing attack possibile.

#### M29. Agent API keys ritornate in chiaro in list (Security)
- **Tipo**: Reale
- **Dettaglio**: `GET /api/v1/agents` ritorna API keys in chiaro.

---

### ⚪ FALSI POSITIVI (7) — Classificati come bug ma NON lo sono

#### FP01. **"NLP port 50051→8001 non migrato"**
- **Realtà**: Già migrato. `main.py:255` usa 8001. `docker-compose.yml` mappa 8001. `config.go` usa 8001. 50051 esiste solo in vecchi file docs.
- **Classificazione**: FALSO POSITIVO ✅

#### FP02. **"yjs causa tsc failure"**
- **Realtà**: yjs NON è in package.json. `npx tsc --noEmit` passa pulito.
- **Classificazione**: FALSO POSITIVO ✅

#### FP03. **"KEY_ENCRYPTION_KEY non validato"**
- **Realtà**: `config.go:57` FATAL se mancante. Test confermano.
- **Classificazione**: FALSO POSITIVO ✅

#### FP04. **"Nessuna documentazione"**
- **Realtà**: README.md 305 linee. Architettura documentata. Funzioni Go hanno doc comment. Docs/ ha manuale-tecnico.md, API.md, CHANGELOG.md, CI-CD-README.md, release-checklist.md.
- **Classificazione**: FALSO POSITIVO ✅

#### FP05. **"Nessun test"**
- **Realtà**: 89 Go _test.go file, 17 frontend test file, 811 test case totali. `go test -race -count=1 ./...` passa.
- **Classificazione**: FALSO POSITIVO ✅

#### FP06. **"Nessuna migration"**
- **Realtà**: 26 file (13 UP + 13 DOWN) in migrations/. DuckDB (7) + Postgres (6+).
- **Classificazione**: FALSO POSITIVO ✅

#### FP07. **"Sicurezza ignorata"**
- **Realtà**: .gitignore con .env. CI con gitleaks. encryption key documentata. CSRF, SecurityHeaders, CSP presenti. auth middleware presente.
- **Classificazione**: FALSO POSITIVO ✅

---

## PARTE 3: RACCOMANDAZIONI PRIORITARIE

### Bloccanti (settimana 1)
1. Fix SQL injection in query.go (C01) — usare sanitizeIdentifier()
2. Fix sandbox allowlist bypass (C02) — bloccare --interactive/--tty, restringere PATH
3. Fix Decision Engine Plan-Act disconnect (C04) — far eseguire Act() i PlannedStep
4. Fix nil Provider degraded mode (C05) — passare provider valido
5. Fix ChatSession panico su plan==nil (C06) — nil check
6. Fix API key cookie → session token (C07)

### Alti (settimana 2)
7. Sandbox container isolation (C09)
8. Postgres DSN hardening (C08)
9. Dual Reflect unification (C03)
10. DuckDB TX deadlock fix (H10)
11. HealthChecker context leak (C12)
12. Rate limiter map growth (C13)
13. Alertmanager receivers (C14)
14. LLM provider timeout (H14)

### Testing (settimana 2-3)
15. ChatSession PAORA tests (C06)
16. Decision Engine tests (engine_test.go)
17. Sandbox tests
18. Contract tests build tag fix (H25)
19. CI pipefail fix (H22)

### Frontend (settimana 3)
20. AbortController in App.tsx data loading (H01, H03)
21. useOntologyActions auth (H02)
22. Empty catch blocks (H05)
23. Consolidare handleError (H08)
24. Zod schemas per Scenario/ToolAnomaly (M09)

---

*Documento generato il 1 Maggio 2026 da Sisyphus. Basato su analisi codebase tramite 5 explore agent + verifica manuale.*
