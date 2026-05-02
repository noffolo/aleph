# 🔴 Aleph-v2 Audit Completo — Report Consolidato

**7 auditor completati** (6 explore + 1 security-red-team) | **~150 findings total** | **2 maggio 2026**

---

## CLUSTER 1: AUTHENTICATION & AUTHORIZATION 🔴

| ID | Severity | Finding |
|----|----------|---------|
| R1 | CRITICAL | `AuthService` RPCs bypassano auth — `CreateApiKey`, `RevokeApiKey` in `authSkipSet` |
| R2 | CRITICAL | RBAC mai applicato — `RequireRole`/`IsAdmin` definite ma mai chiamate in nessun handler |
| R3 | CRITICAL | `ALEPH_API_KEY_SECRET_BACKEND` env backdoor — chiunque con accesso env può impersonare il backend |
| R4 | HIGH | IDOR: `projectID` dal request body sovrascrive auth context — furto cross-tenant |
| R5 | HIGH | Brute-force auth endpoint — 500 req/min senza lockout/rate limiting su login |
| R6 | HIGH | SSE bypassa TUTTI i middleware — no auth, no rate limit, no telemetry |
| R7 | HIGH | API key nei proto messages in chiaro — Agent API key mostrata in risposte `ListAgents` |
| R8 | HIGH | JWT senza validazione `aud`/`sub` — token accettato senza verificare audience o subject |
| R9 | HIGH | Tool API senza project scoping o auth — tool eseguibili senza verificare appartenenza al progetto |
| R10 | MEDIUM | Role fallback a env var — se token JWT non ha role, prende `ALEPH_DEFAULT_ROLE` |

---

## CLUSTER 2: INJECTION ATTACKS 🟠

| ID | Severity | Finding |
|----|----------|---------|
| I1 | CRITICAL | SQL injection via double-escaped DSN — `ReplaceAll` + `QuoteStringLiteral` = triple-quoting |
| I2 | HIGH | SQL injection: dynamic table names in `memory.go` — `fmt.Sprintf` con table names (mitigato da safeident ma difettoso) |
| I3 | HIGH | SQL injection in `GetDataStats` — limit non range-validated |
| I4 | HIGH | Search text LIKE injection — input utente direttamente in LIKE pattern |
| I5 | MEDIUM | DSL injection in tool code — user-generated code eseguito senza sanitization completa |
| I6 | MEDIUM | Prototype pollution via chat input — oggetti utente iniettati senza sanitization |

---

## CLUSTER 3: SANDBOX E CODE EXECUTION ISOLATION 🔴

| ID | Severity | Finding |
|----|----------|---------|
| S1 | CRITICAL | `curl` e `pip` in `CommandAllowlist` — SSRF gateway + supply chain via `curl` e `pip install` |
| S2 | CRITICAL | Python sandbox: `urllib` non blocklisted — e `apiConnectorPythonTemplate` lo importa |
| S3 | CRITICAL | `ExecSandbox` zero isolamento reale — no network namespace, chroot, cgroup; solo code validation bypassabile |
| S4 | CRITICAL | `runDynamic` esegue codice Go senza isolamento — blocklist manca `os`, `plugin`, `runtime`, `database/sql` |
| S5 | CRITICAL | Python blocklist bypassabile via `importlib` — `importlib.import_module("s"+"ubprocess")` aggira il check |
| S6 | HIGH | Python blocklist manca: `os.exec*`, `pickle`, `code`, `compile`, `eval` nativo, `exec` |
| S7 | HIGH | Go blocklist manca: `os`, `plugin`, `runtime`, `database/sql`, `syscall` |
| S8 | HIGH | Sandbox Python fallback a processo senza container — se Docker non disponibile |
| S9 | MEDIUM | Blocklist inconsistenti Go vs Python — Python ha `ctypes`, Go no |
| S10 | MEDIUM | Sandbox Genesis non validato — file di init non verificati |

---

## CLUSTER 4: DATA LEAKAGE & SECRETS MANAGEMENT 🔴

| ID | Severity | Finding |
|----|----------|---------|
| L1 | CRITICAL | Email credentials in `/proc/environ` — password passata come `ALEPH_EMAIL_PASS` env var al subprocess Python |
| L2 | CRITICAL | `window.__ALEPH_STORE__` in produzione — l'intero store Zustand (incl. `apiKey`) esposto a qualsiasi script nella pagina |
| L3 | HIGH | `apiKey` ancora in memoria Zustand nonostante migrazione a httpOnly cookie |
| L4 | HIGH | Agent API key leak — primi 8 caratteri mostrati in UI agent list |
| L5 | HIGH | `PATH` env var leakato al sandbox — variabili d'ambiente del processo host visibili |
| L6 | HIGH | Email IMAP host senza SSRF validation — user-controlled host, no checking |
| L7 | HIGH | `SendWebhook` senza URL validation o SSRF protection |
| L8 | MEDIUM | Webhook URL unchecked lato frontend — SSRF possibile |
| L9 | MEDIUM | `sessionStorage` per command history — accessibile da JS, non encrypted |
| L10 | MEDIUM | API key masking mostra ultimi 4 chars — facilita brute-force parziale |

---

## CLUSTER 5: DATABASE LAYER 🟠

| ID | Severity | Finding |
|----|----------|---------|
| D1 | CRITICAL | DuckDB `RWMutex` serializza TUTTE le letture — distrugge la concorrenza, mutex globale |
| D2 | CRITICAL | `DeleteProjectCascade` NON atomico tra DuckDB + PostgreSQL — possibile stato inconsistente |
| D3 | CRITICAL | `DuckDBRegistry` zero protezione accesso concorrente — nessun mutex/lock |
| D4 | HIGH | `MemoryStore.Store` DELETE+INSERT non atomico — race condition tra delete e insert |
| D5 | HIGH | Zero `NOT NULL` constraints su PostgreSQL — colonne critiche nullable senza motivo |
| D6 | HIGH | Zero foreign key constraints — integrità referenziale non enforced |
| D7 | HIGH | Missing indexes su FK-like columns — ogni JOIN è table scan potenziale |
| D8 | HIGH | `QueryRowContext` ritorna `nil` senza error su semaphore exhaustion — silent failure |
| D9 | HIGH | `ToolCache` nessun size bound — crescita illimitata, memory leak |
| D10 | MEDIUM | Duplicate schema definitions in 3 location diverse — drift risk |
| D11 | MEDIUM | `ExportDatabase` `context.Background` — no timeout/cancellation |
| D12 | MEDIUM | DuckDB backup `Lock()` esclusivo blocca letture durante backup |

---

## CLUSTER 6: API SECURITY & VALIDATION 🟡

| ID | Severity | Finding |
|----|----------|---------|
| A1 | HIGH | CSRF bypass quando Origin E Referer mancanti entrambi — doppio fallback = nessun check |
| A2 | HIGH | Nessun rate limiting su endpoint auth — brute-force JWT/API key possibile |
| A3 | HIGH | CSP permette `unsafe-inline` per styles |
| A4 | HIGH | CSP hardcoded `ws://localhost` dev origin in produzione |
| A5 | MEDIUM | CSP allow `unsafe-inline` per scripts |
| A6 | MEDIUM | No HSTS header |
| A7 | MEDIUM | No Permissions-Policy header |
| A8 | MEDIUM | Paginazione ignorata in `ListAgents`/`ListTools` — proto fields `After`/`Limit` non rispettati |
| A9 | MEDIUM | Error response inconsistente — raw HTTP vs ConnectRPC errori mescolati |
| A10 | LOW | No request ID in error responses — debugging difficile |

---

## CLUSTER 7: FRONTEND QUALITY & SECURITY 🟡

| ID | Severity | Finding |
|----|----------|---------|
| F1 | HIGH | `useStore.getState()` chiamato 50+ volte — bypassa React subscriptions, anti-pattern |
| F2 | HIGH | Race condition in `App.tsx` data fetches — nessun `AbortController`, component unmount non gestito |
| F3 | HIGH | Nessun focus trap in `SlideOverPanel` — accessibilità rotta per keyboard users |
| F4 | MEDIUM | `console.error` in produzione — 7 file, 9 occorrenze, leak di dettagli interni |
| F5 | MEDIUM | `assertType<T>` type lie — finger-crossed type assertion senza runtime check |
| F6 | MEDIUM | `confirm()`/`alert()` per UI — blocca event loop, non accessibile |
| F7 | MEDIUM | Form senza `<form>` elements — niente submit su Enter, niente validation nativa |
| F8 | MEDIUM | Sistemi adapter duplicati — da proto e da API, divergenti |
| F9 | LOW | Hardcoded placeholder dashboard stats |
| F10 | MEDIUM | 9 test vitest falliti — `useStore.getState()` non è funzione in test context |

---

## CLUSTER 8: CODE QUALITY & ARCHITECTURE 🟡

| ID | Severity | Finding |
|----|----------|---------|
| Q1 | HIGH | `ToolRegistry.Execute` usa `context.Background()` — bypassa request context, no cancellation |
| Q2 | HIGH | `json.Unmarshal` errori ingoiati in `agent.go` — silent failures |
| Q3 | HIGH | Notification service droppa errori silenziosamente — nessun retry, nessun alert |
| Q4 | HIGH | No request context propagation in HTTP clients — nessun timeout, tracing rotto |
| Q5 | HIGH | DI via `Set` dopo costruzione non thread-safe — race condition su inizializzazione |
| Q6 | HIGH | Chat streaming no per-iteration timeout — loop infinito possibile |
| Q7 | MEDIUM | `ToolCache` TTL senza cleanup goroutine — entry scadute mai rimosse |
| Q8 | MEDIUM | Panic in `init` per CIDR invalidi — crash all'avvio |
| Q9 | MEDIUM | Error wrapping inconsistente — `fmt.Errorf` vs `%w` usati a caso |
| Q10 | MEDIUM | `context.Background()` usato in produzione — no propagation |
| Q11 | MEDIUM | Mixed Italian/English error messages |
| Q12 | MEDIUM | Duplicate schema definitions (migration side) |
| Q13 | LOW | Dead imports in `compiler_tool.go` |
| Q14 | LOW | `ValidateStrictIdentifier` è solo un alias — false sense of security |

---

## CLUSTER 9: INFRASTRUCTURE & CI/CD 🟡

| ID | Severity | Finding |
|----|----------|---------|
| INF1 | CRITICAL | Plaintext secrets in `.env` con weak defaults — `MASTER_KEY`, password, API key |
| INF2 | HIGH | Main Dockerfile copia intero `site-packages` — immagine enorme, layer non ottimizzati |
| INF3 | HIGH | Frontend nginx HTTP porta 80 no TLS — tutto in chiaro |
| INF4 | HIGH | PostgreSQL esposto esternamente su 5432 |
| INF5 | HIGH | Ollama esposto esternamente su 11434 — accesso LLM non autenticato |
| INF6 | HIGH | No Docker build caching — rebuild completo ogni volta |
| INF7 | HIGH | Deploy no rollback strategy — se il deploy fallisce, manual recovery |
| INF8 | HIGH | Deploy no migration step — DB migration non automatizzata nel deploy |
| INF9 | MEDIUM | No resource limits su container — CPU/memory unbounded |
| INF10 | MEDIUM | Ollama pull models ad ogni start — avvio lentissimo |
| INF11 | MEDIUM | Prometheus/Grafana/Alertmanager no auth hardening |
| INF12 | MEDIUM | CI no timeout configuration — job possono runnare indefinitamente |
| INF13 | MEDIUM | CI duplica security scan — gitleaks + secrets scanning ridondanti |

---

## CLUSTER 10: GOROUTINE & CONCURRENCY 🟠

| ID | Severity | Finding |
|----|----------|---------|
| G1 | HIGH | Race condition su `nextID` in `ToolSuggestHandler` — contatore non atomico |
| G2 | HIGH | Goroutine lifecycle management gaps — `Start`/`Stop` non robusti, leak detection assente |
| G3 | HIGH | `RunTask` goroutine usa parent context dopo request completa — context scaduto ma goroutine viva |
| G4 | MEDIUM | `NotificationService.Stop()` mai chiamato in `app.Close()` — goroutine leak |
| G5 | MEDIUM | MCP discovery `Start()` in goroutine senza retry — fallisce silenziosamente |
| G6 | MEDIUM | `tool_suggest` cleanup usa `r.Context()` invece di `app.ctx` |

---

## 📊 SUMMARY STATISTICS

| Severity | Count |
|----------|-------|
| 🔴 CRITICAL | 22 |
| 🟠 HIGH | 38 |
| 🟡 MEDIUM | 39 |
| ⚪ LOW | 12 |
| **TOTAL** | **111** |

---

## 🎯 TOP 10 IMMEDIATE ACTIONS (da fixare oggi)

| # | Action |
|---|--------|
| 1 | Remove `CreateApiKey`/`RevokeApiKey` from `authSkipSet` |
| 2 | Fix DSN double-escaping in `engine.go:708` |
| 3 | Remove `curl`/`pip` from `CommandAllowlist` |
| 4 | Add `urllib`, `http.client`, `importlib`, `os.exec*`, `pickle`, `code` to Python blocklist |
| 5 | Add `os`, `plugin`, `runtime`, `database/sql`, `syscall` to Go blocklist |
| 6 | Remove `window.__ALEPH_STORE__` from production builds |
| 7 | Stop passing email credentials as env vars to Python subprocess |
| 8 | Fix CSRF middleware — reject when BOTH Origin AND Referer missing |
| 9 | Enforce RBAC: call `RequireRole`/`IsAdmin` in all mutating handlers |
| 10 | Fix DuckDB RWMutex: replace with `sync.RWMutex` or per-file locks |

---

## 📋 PROPOSTA PIANO DI REMEDIATION

| Wave | Focus | Items | Stima |
|------|-------|-------|-------|
| W1: Critical Security | Top 10 actions + varianti | ~15 | 1-2 gg |
| W2: Auth & Sandbox Hardening | RBAC, isolation reale, blocklist | ~10 | 1 gg |
| W3: Data Layer & Injection | DuckDB, atomic ops, SQL injection | ~8 | 1 gg |
| W4: Frontend & API Quality | Store leak, test fix, CSP, validation | ~10 | 1 gg |
| W5: Infrastructure & Polish | CI/CD, Docker, monitoring, errori minori | ~10 | 0.5 gg |

**Totale: ~53 items, ~5 giorni lavorativi.**
