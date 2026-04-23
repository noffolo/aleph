# Piano Assemblea Cooperativa — Aleph-v2 (Sintesi Dialettica)

> **Principio**: Tesi e antitesi sono state confrontate. Le duplicazioni sono state fuse preservando le fonti. I conflitti sono stati risolti con sintesi esplicite. Il piano è organizzato per ONDA con dipendenze, stime di sforzo, e marcatori di rischio.

---

## Legenda

| Simbolo | Significato |
|---------|------------|
| **S/M/L/XL** | Sforzo: Small (< 4h), Medium (4-16h), Large (1-3gg), XL (3-5gg) |
| ⚠️ | Rischio: incertezza architetturale o dipendenza esterna |
| 🔗 | Dipendenza da item precedente |
| 🔄 | Conflitto risolto con sintesi |

**Fonti**: P1 (Fullstack Dev), P2 (Debug/RE), P3 (Analyst), P4 (Filosofo Epistemologico), P5 (Filosofo Euristico), P6 (UX/Microtipografia), P7 (Art Director), P8 (Go+Python), P9 (React+API), P10 (End User), Lib (Librarian Go/React), PF (Piano Frontend esistente), GLM (Piano GLM-5.1)

**Severità**: 🔴 CRITICA (blocca deploy/sicurezza), 🟠 ALTA (blocca sviluppo/UX), 🟡 MEDIA (degrada qualità), 🟢 BASSA (polish)

---

## ONDA 0 — Sicurezza Critica 🔴 ⛔ BLOCCA TUTTO

> Nessun sviluppo prosegue finché W0 non è completato.

### W0-01 — Iniezione SQL in query e ingestion (FUSO) 🔄
- **Fonte**: P1, P2 (originale W0-01 + W0-11)
- **Severità**: 🔴 CRITICA
- **Sforzo**: L
- **File**: `handler/query.go:128-136, 167-168, 228, 246-257`; `engine.go` 6+ siti ingestion
- **Descrizione**: `fmt.Sprintf` con nomi tabella/colonna derivati dall'utente, sia nelle query (W0-01 originale) sia nell'ingestion (W0-11 originale). Lo stesso vettore in due punti — fuso per evitare fix parziale.
- **Criteri di accettazione**:
  - [ ] Funzione `sanitizeIdentifier()` con regex `^[a-zA-Z_][a-zA-Z0-9_]{0,62}$` + quoting DuckDB
  - [ ] Zero `fmt.Sprintf` per costruire SQL dinamico con input utente (grep verify)
  - [ ] Test penetration con payload di iniezione su ogni endpoint query e ingestion
  - [ ] `query_test.go`: test con nomi tabella malevoli (`DROP TABLE; --`, `' OR 1=1`, etc.)

### W0-02 — Sandbox senza isolamento
- **Fonte**: P1, P2
- **Severità**: 🔴 CRITICA
- **Sforzo**: XL ⚠️
- **File**: `exec_sandbox.go`, sidecar Python
- **Descrizione**: Sandbox Go blocca solo 4 import. Sandbox Python senza restrizioni. PATH dell'host.
- **Rischio**: L'isolamento completo (container, seccomp) richiede cambi infrastrutturali.
- **Criteri di accettazione**:
  - [ ] Go: blocklist estesa a `{syscall, net, os/exec, os/user, crypto/*, database/sql, io/ioutil, archive/*, compress/*, encoding/json, encoding/xml, plugin, reflect}`
  - [ ] Python: restrizioni di import comparabili
  - [ ] Esecuzione in container con network disabled (Docker `network_mode: none`)
  - [ ] Test: tentativi di import bloccati ritornano errore esplicito
  - [ ] README aggiornato con modello di threat

### W0-03 — Segreti hardcoded in docker-compose
- **Fonte**: P1
- **Severità**: 🔴 CRITICA
- **Sforzo**: S
- **File**: `docker-compose.yml`
- **Criteri di accettazione**:
  - [ ] Tutti i segreti letti da `${ENV_VAR}` con `.env` file
  - [ ] `.env.example` con valori placeholder, `.env` in `.gitignore`
  - [ ] Nessun segreto in chiaro in file versionati (grep verify)

### W0-04 — Chiavi API in chiaro + leak nella risposta proto (FUSO)
- **Fonte**: P1, P4 (originale W0-04 + W1-11)
- **Severità**: 🔴 CRITICA
- **Sforzo**: M
- **File**: `handler/agent.go:37-43`; schema proto `Agent`
- **Descrizione**: Le chiavi API sono in chiaro in DB E nella risposta gRPC. Fuso perché il leak nella risposta è conseguenza diretta della stessa vulnerabilità.
- **Criteri di accettazione**:
  - [ ] Chiavi API memorizzate con AES-256-GCM (KMS o env key)
  - [ ] Campo `apiKey` rimosso dal messaggio proto `Agent` o mascherato (`****`)
  - [ ] Test: risposta serializzata non contiene chiave API leggibile
  - [ ] Rotazione chiavi supportata senza downtime

### W0-05 — Confusione entrypoint duale
- **Fonte**: P1, P8 (conferma GLM T1.1-T1.3)
- **Severità**: 🔴 CRITICA (blocca deploy)
- **Sforzo**: S
- **File**: `main.go` vs `cmd/aleph-server/main.go`
- **Criteri di accettazione**:
  - [ ] `cmd/aleph-server/main.go` eliminato
  - [ ] Singolo entrypoint `main.go` alla radice
  - [ ] Dockerfile e Makefile aggiornati
  - [ ] `go build ./...` passa

### W0-06 — Autenticazione chat fallisce sempre
- **Fonte**: P2 (conferma GLM T1.2)
- **Severità**: 🔴 CRITICA
- **Sforzo**: S
- **File**: `query.go:314, 421`
- **Criteri di accettazione**:
  - [ ] `Chat()` confronta `sha256(inputKey)` con hash memorizzato
  - [ ] Test: autenticazione con chiave valida → successo
  - [ ] Test: autenticazione con chiave invalida → fallimento
  - [ ] Test: chiave vuota/missing → skip (per AuthService interno)

### W0-07 — Y.js sicurezza room
- **Fonte**: P1, P2, PF (FASE 2.4)
- **Severità**: 🔴 CRITICA
- **Sforzo**: M ⚠️
- **Rischio**: Richiede cambiamento backend per generazione JWT.
- **Criteri di accettazione**:
  - [ ] Eliminare `simpleHash(apiKey)` come nome room
  - [ ] Backend genera token JWT per autenticazione room (endpoint `/api/v1/collab-token`)
  - [ ] Signaling auth con token JWT
  - [ ] Test collision per il nuovo schema di naming
  - [ ] Frontend: `sessionStorage` per API key (non `localStorage`) 🔗 PF FASE 2.5

### W0-08 — SSRF bypass in engine.go
- **Fonte**: P2
- **Severità**: 🔴 CRITICA
- **Sforzo**: M
- **File**: `engine.go:60-83`
- **Criteri di accettazione**:
  - [ ] DNS resolution dopo validazione (anti-rebinding)
  - [ ] Blocco: IPv6 loopback, rappresentazioni ottali, `0.0.0.0`, `127.x.x.x`
  - [ ] Test per ogni vettore: `[::1]`, `0177.0.0.1`, `0.0.0.0`, DNS rebinding
  - [ ] Stessa correzione per webhook endpoint

### W0-09 — Data leakage cross-project DuckDB
- **Fonte**: P5
- **Severità**: 🔴 CRITICA
- **Sforzo**: M
- **File**: `storage/duckdb.go`
- **Criteri di accettazione**:
  - [ ] Ogni progetto usa schema DuckDB isolato `project_{id}`
  - [ ] Query sempre scoped allo schema del progetto autenticato
  - [ ] Test: utente progetto A non legge dati progetto B
  - [ ] Migration: migrazione dati esistenti in schemi separati

### W0-10 — DuckDB: metodo DB() bypassa concorrenza
- **Fonte**: P1, P2
- **Severità**: 🟠 ALTA
- **Sforzo**: S
- **File**: `duckdb.go:102`
- **Criteri di accettazione**:
  - [ ] Metodo `DB()` reso privato (`db`)
  - [ ] Tutte le query passano da `QueryContext` o `ExecContext`
  - [ ] `grep -r '\.DB()' codebase` → zero risultati

### W0-11 — CORS permissivo
- **Fonte**: P1
- **Severità**: 🟠 ALTA
- **Sforzo**: S
- **File**: `app.go:182-187`
- **Criteri di accettazione**:
  - [ ] CORS ristretto a `ALLOWED_ORIGINS` da env var
  - [ ] Dev: `localhost:5173`; prod: dominio configurato
  - [ ] Credenziali e metodi HTTP espliciti
  - [ ] Content Security Policy header in produzione 🔗 PF FASE 2.6

### W0-12 — Slash command allow-list e sanitization (MANCANTE DA ORIGINALE)
- **Fonte**: PF FASE 2.1-2.3
- **Severità**: 🔴 CRITICA
- **Sforzo**: M
- **File**: `slashCommands.ts`, `TerminalPrompt`
- **Descrizione**: Assente dal piano originale ma critico per sicurezza frontend. Comandi arbitrari eseguibili, output agente non sanitized.
- **Criteri di accettazione**:
  - [ ] Allow-list: solo comandi in `slashCommands.ts` sono eseguibili
  - [ ] Comandi mutanti (`/agent create`, `/skills run`) richiedono conferma
  - [ ] Output agente LLM: plain text escaped, nessun HTML rendering
  - [ ] Se agente scrive `/explore` nel suo output → NON interpretato come comando

---

## ONDA 1 — Indurimento Architetturale 🔗 Dopo W0 completo

### W1-01 — Decomporre monolite Zustand
- **Fonte**: P1, P9, PF (FASE 1), Codemem
- **Severità**: 🟠 ALTA (blocca sviluppo frontend parallelo)
- **Sforzo**: L ⚠️
- **Rischio**: Change grande, tocca ogni componente che usa lo store.
- **Dipendenze**: 🔗 W0-07 (Y.js auth), W0-12 (slash commands)
- **File**: Frontend store (~60 campi, 330+ righe)
- **Criteri di accettazione**:
  - [ ] Store decomposto in 5+ slices: `useAuthStore`, `useProjectStore`, `useDataStore`, `useAgentStore`, `useUIStore`
  - [ ] Ogni slice ha interfaccia tipizzata propria
  - [ ] Nessun re-render cross-slice non necessari
  - [ ] `activeTab` marcato `@deprecated`, rimosso da `SYNCED_KEYS`, one-time `yMap.delete('activeTab')`
  - [ ] Campi nuovi: `slideOverContent`, `sandboxResult`, `sandboxInput`, `terminalMode: 'copilot'`

### W1-02 — Aggiungere migrazioni database
- **Fonte**: P1, GLM (infra)
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] `golang-migrate` o `goose` integrato
  - [ ] Tutte le modifiche schema sono migrate versionate (up/down)
  - [ ] Test: migrate up → migrate down → migrate up (roundtrip)
  - [ ] Documentazione su come creare nuove migrazioni

### W1-03 — Estrarre logica provider LLM in interfaccia
- **Fonte**: P1, P8
- **Severità**: 🟠 ALTA
- **Sforzo**: L
- **Criteri di accettazione**:
  - [ ] Interfaccia `LLMProvider` con metodi `Complete(ctx, req)`, `Stream(ctx, req)`, `Embed(ctx, req)`
  - [ ] Implementazioni: `OllamaProvider`, `OpenAIProvider`, `AnthropicProvider`
  - [ ] Factory pattern per selezione runtime
  - [ ] Nessuna logica provider-specific nel codice chiamante

### W1-04 — Goroutine staccate + context.Background()
- **Fonte**: P2
- **Severità**: 🟠 ALTA
- **Sforzo**: S
- **File**: `ingestion.go:95-97`
- **Criteri di accettazione**:
  - [ ] Goroutine usano contesti derivati da richiesta/app cancellabile
  - [ ] Shutdown graceful: `WaitGroup` + timeout per goroutine in corso
  - [ ] Nessun `context.Background()` in business logic (grep verify)

### W1-05 — Leak connessione gRPC NLP + error mapping
- **Fonte**: P1, P2
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **File**: `app.go:232-253`
- **Criteri di accettazione**:
  - [ ] Connessione gRPC chiusa in `Close()` dell'applicazione
  - [ ] Errori sidecar mappati a codici gRPC appropriati (non tutti `Unavailable`)
  - [ ] `enrichPredictiveMetadata` propaga errori al chiamante
  - [ ] Test: chiusura connessione graceful sotto carico

### W1-06 — Chat streaming: abort su disconnessione
- **Fonte**: P2
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] Context cancellation detect con timeout 5s
  - [ ] Abort streaming LLM su disconnessione client
  - [ ] Test con simulated disconnect

### W1-07 — Mappa programmi senza bound (memory leak)
- **Fonte**: P2
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] LRU eviction o TTL per programmi nella mappa
  - [ ] Limite massimo configurabile (default 100)
  - [ ] Metriche su dimensione mappa esposte a monitoring

### W1-08 — Agent ListModels senza timeout
- **Fonte**: P2
- **Severità**: 🟡 MEDIA
- **Sforzo**: S — 🔗 con W3-07 (timeout budgets)
- **Criteri di accettazione**:
  - [ ] `http.Client{Timeout: 10 * time.Second}`
  - [ ] Context con cancel per ogni richiesta
  - [ ] Test con server lento che verifica timeout

### W1-09 — Concurrency DuckDB: semplificare o aggiungere fairness 🔄
- **Fonte**: P2 (starvation writer), P5 (premature optimization)
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Descrizione**: P5 classifica la triple concurrency come premature optimization. P2 identifica starvation writer. Sintesi: semplificare a write-preferring RWMutex E aggiungere fairness; se il benchmark mostra che la triple concurrency è giustificata, mantenerla con fix.
- **Criteri di accettazione**:
  - [ ] Benchmark prima e dopo semplificazione
  - [ ] Se semplificato: write-preferring RWMutex singolo + pool
  - [ ] Se mantenuto: fairness policy per writer
  - [ ] Writer completa entro timeout ragionevole (200ms) sotto carico read

### W1-10 — PRAGMA DuckDB specifici per SQLite
- **Fonte**: P2
- **Severità**: 🟢 BASSA
- **Sforzo**: S
- **File**: `storage/duckdb.go`
- **Criteri di accettazione**:
  - [ ] PRAGMA SQLite-specifici rimossi
  - [ ] Sostituiti con PRAGMA DuckDB appropriati per concurrency
  - [ ] Documentazione PRAGMA DuckDB supportati

### W1-11 — Architettura esagonale (piano, non esecuzione)
- **Fonte**: Lib (Go)
- **Severità**: 🟡 MEDIA (guida strutturale)
- **Sforzo**: S (piano) / XL (esecuzione)
- **Rischio**: ⚠️ Migration incrementale, non big-bang
- **Descrizione**: Questo item è un PIANO, non un'implementazione immediata. La ristrutturazione va fatta gradualmente.
- **Criteri di accettazione**:
  - [ ] Piano documentato: `docs/architecture-migration.md`
  - [ ] Target: `cmd/aleph-server/`, `internal/{handler,service,repository}/`, `pkg/`
  - [ ] Nessuna dipendenza circolare tra layer
  - [ ] Piano per fase, non big-bang refactor

---

## ONDA 2 — Integrità Epistemica 🔗 Dopo W1 (parziale — può iniziare in parallelo con W1 items indipendenti)

### W2-01 — Feature sentiment fantasma (sempre 0.0)
- **Fonte**: P4
- **Severità**: 🔴 CRITICA (epistemologica)
- **Sforzo**: M
- **File**: `engine.go:275-279`
- **Criteri di accettazione**:
  - [ ] Se sentiment implementato: test end-to-end con output non-zero
  - [ ] Se placeholder: rimuovere e documentare come futuro enhancement
  - [ ] Mai mostrare "sentiment analysis" nella UI se il valore è sempre 0

### W2-02 — Dati sintetici di fallback non etichettati
- **Fonte**: P4
- **Severità**: 🔴 CRITICA (epistemologica)
- **Sforzo**: S
- **File**: `main.py:168` (sidecar)
- **Criteri di accettazione**:
  - [ ] Flag `is_synthetic` nei risultati predittivi
  - [ ] UI: badge "sintetico" quando dati sono di fallback
  - [ ] Documentazione chiara sulla natura dei dati

### W2-03 — Stringhe ragionamento LLM fabbricate
- **Fonte**: P4
- **Severità**: 🔴 CRITICA (epistemologica)
- **Sforzo**: M
- **File**: `query.go:522`
- **Criteri di accettazione**:
  - [ ] Se LLM disponibile: usare output LLM reale con attribuzione
  - [ ] Se LLM non disponibile: mostrare "Ragionamento non disponibile" (non testo fittizio)
  - [ ] Mai generare testo che simula ragionamento senza fonte LLM reale

### W2-04 — Brier score e Trust score: persistenza o rimozione (FUSO)
- **Fonte**: P4, P5 (originale W2-04 + W2-05)
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Descrizione**: P4: Brier score calcolato ma mai persistito. P5: è premature optimization. Sintesi: decidere tra persistenza e rimozione, ma non lasciare codice fantasma.
- **Criteri di accettazione**:
  - [ ] Decisione documentata: persistere O rimuovere (non entrambi)
  - [ ] Se persistere: `WriteTrustScore` nel registro DuckDB + API endpoint per lettura storica
  - [ ] Se rimuovere: eliminare codice morto + commenti che suggeriscono funzionalità
  - [ ] Grep: nessun riferimento a Brier/trust functionality fantasma rimasto

### W2-05 — Provenienza dati su ingestion
- **Fonte**: P4
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Ogni record ingerito ha metadata: `source`, `ingested_at`, `transform_version`, `quality_score`
  - [ ] API per interrogare lineage di un dato
  - [ ] UI mostra provenienza quando disponibile (🔗 W5-09)

### W2-06 — Feedback: pozzo nero (write-only)
- **Fonte**: P4
- **Severità**: 🟠 ALTA
- **Sforzo**: L ⚠️
- **Rischio**: Richiede design di pipeline di feedback — può essere implementato in fasi.
- **Criteri di accettazione**:
  - [ ] Fase 1: feedback consumato per aggiornare trust score (🔗 W2-04)
  - [ ] Fase 2: feedback influisce su pesi del modello
  - [ ] Se non consumato ora: disabilitare raccolta O etichettare "contributo futuro"

### W2-07 — Sigmoid non calibrata
- **Fonte**: P4
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **File**: `ensemble.py:38-48`
- **Criteri di accettazione**:
  - [ ] Platt Scaling o Isotonic Regression su dati di validazione
  - [ ] Se non ci sono dati: usare media semplice senza sigmoid
  - [ ] Documentare metodo e metriche di calibrazione

### W2-08 — UI: probabilità come deterministiche senza incertezza (FUSO con W5-09)
- **Fonte**: P4, P10, PF
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **File**: Frontend terminale
- **Descrizione**: Fuso con W5-09 del piano originale — stesso problema da prospettive diverse.
- **Criteri di accettazione**:
  - [ ] Intervallo di confidenza visibile (es. "72% ±8%")
  - [ ] Indicatore livello incertezza con colore/icone
  - [ ] Badge "sintetico" per dati di fallback (🔗 W2-02)
  - [ ] Citazioni ai dati sorgente
  - [ ] Chain-of-thought reale visibile (🔗 W2-03) — non fabbricato

### W2-09 — Troncamento JSON distrugge semantica
- **Fonte**: P4
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **File**: `query.go:542-543`
- **Criteri di accettazione**:
  - [ ] Troncare a livello di oggetto JSON completo
  - [ ] Se messaggio troppo lungo: ultimo oggetto JSON + `…[truncated]`
  - [ ] Test con payload di varie dimensioni

### W2-10 — GNN addestrato solo su link positivi
- **Fonte**: P4
- **Severità**: 🟡 MEDIA
- **Sforzo**: L ⚠️
- **Rischio**: Richiede dati negativi che potrebbero non esistere.
- **Criteri di accettazione**:
  - [ ] Aggiungere negative sampling al dataset di training
  - [ ] Metriche su link prediction (AUC, MRR) con set negativo
  - [ ] Se non implementabile ora: documentare limitazione + warning nella UI

### W2-11 — StreamPredictions manca recordSuccess() + Circuit breaker valutazione 🔄
- **Fonte**: P2 (W2-12), P5 (W6-01)
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Descrizione**: Fuso — `StreamPredictions` non chiama `recordSuccess()` è un bug che rende il circuit breaker permanentemente aperto. P5 dice il circuit breaker stesso è premature optimization. Sintesi: fix il bug OR semplifica a retry.
- **Criteri di accettazione**:
  - [ ] Opzione A (fix): `StreamPredictions` chiama `recordSuccess()`, test ciclo open→half-open→closed
  - [ ] Opzione B (semplifica): rimpiazzare circuit breaker con retry + backoff (🔗 W3-07)
  - [ ] Decisione documentata con giustificazione

### W2-12 — Errori json.Unmarshal inghiottiti
- **Fonte**: P2
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **File**: `query.go:471, 480`
- **Criteri di accettazione**:
  - [ ] Tutti gli errori `json.Unmarshal` loggati con contesto
  - [ ] Errori critici ritornati al chiamante
  - [ ] Test con payload malformati

### W2-13 — Watcher service no-op
- **Fonte**: P2
- **Severità**: 🟢 BASSA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] Se necessario: implementare logica di aggiunta directory
  - [ ] Se non necessario: rimuovere codice morto
  - [ ] Decisione documentata

### W2-14 — Claim "Decision Intelligence" disonesto
- **Fonte**: P4
- **Severità**: 🟠 ALTA (epistemologica)
- **Sforzo**: S
- **File**: README, UI
- **Descrizione**: P4: il sistema claima "Decision Intelligence" ma è "Data Query + LLM Chat + Prediction Decoration". Sintesi: qualificare il claim, non rimuoverlo.
- **Criteri di accettazione**:
  - [ ] UI: "Decision Intelligence (beta)" con disclaimer onestà
  - [ ] Ogni funzione predittiva ha metriche di accuratezza visibili
  - [ ] Feature placeholder etichettate come tali nella UI
  - [ ] Disclaimer: "Le predizioni sono stime con livelli di incertezza indicati"

---

## ONDA 3 — Developer Experience & Infrastruttura 🔗 Può iniziare in parallelo con W2

### W3-01 — Pipeline CI/CD
- **Fonte**: P1
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] GitHub Actions: lint Go + lint Frontend + test Go + test Frontend + build Docker
  - [ ] Branch protection: review required + CI verde
  - [ ] Deploy automatico su merge a main (staging)

### W3-02 — Linting e formattazione
- **Fonte**: P1
- **Severità**: 🟠 ALTA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] `golangci-lint` configurato e verde in CI
  - [ ] `eslint` + `prettier` configurati e verdi in CI
  - [ ] Pre-commit hooks con linting
  - [ ] Zero warning iniziali (fixati o suppressi con commento)

### W3-03 — Unit test per moduli critici
- **Fonte**: P1, GLM (Track 7)
- **Severità**: 🟠 ALTA
- **Sforzo**: L
- **Criteri di accettazione**:
  - [ ] `auth_service_test.go`: validazione chiave, hash, errori
  - [ ] `query_test.go`: injection prevention, parametri, edge cases
  - [ ] `chat_test.go`: autenticazione, streaming, disconnessione
  - [ ] `circuit_breaker_test.go`: stati open/closed/half-open (🔗 W2-11)
  - [ ] `parser_test.go` e `compiler_test.go`: filtri, aggregazioni, errori (🔗 GLM T4.4)
  - [ ] Copertura minima 50% per moduli critici

### W3-04 — OpenTelemetry e logging strutturato
- **Fonte**: P1
- **Severità**: 🟡 MEDIA
- **Sforzo**: L
- **Criteri di accettazione**:
  - [ ] OpenTelemetry SDK integrato (traces + metrics)
  - [ ] Logging strutturato con `slog`
  - [ ] Endpoint tracing configurabile (Jaeger/OTLP)
  - [ ] Frontend: error boundary con contesto inviato a monitoring 🔗 PF FASE 12

### W3-05 — Standardizzare messaggi errore
- **Fonte**: P1
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] Audit di tutti i messaggi errore Go
  - [ ] Errori tecnici (gRPC, log): inglese
  - [ ] Messaggi UI utente: italiano (con termini tecnici in inglese)
  - [ ] Glossario di traduzione per messaggi utente 🔗 W6-06

### W3-06 — Air hot reload per Go backend
- **Fonte**: P1
- **Severità**: 🟢 BASSA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] `air` configurato con `.air.toml`
  - [ ] Restart automatico su modifica file Go
  - [ ] Documentazione nel README

### W3-07 — Timeout budgets, retry, bulkhead (FUSO)
- **Fonte**: P5, P2 (W1-06, W1-08 parziali)
- **Severità**: 🟠 ALTA
- **Sforzo**: L
- **Criteri di accettazione**:
  - [ ] Timeout: DB 5s, LLM 30s, NLP 10s, HTTP esterno 15s
  - [ ] Retry con exponential backoff per operazioni idempotenti
  - [ ] Fallback per PostgreSQL down (cache locale per metadata)
  - [ ] Bulkhead: pool separato per dominio (query, ingestion, chat)

### W3-08 — Audit logging
- **Fonte**: P5
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Middleware audit per operazioni mutanti (create, update, delete)
  - [ ] Log strutturati: `user_id`, `action`, `resource_type`, `resource_id`, `timestamp`, `diff`
  - [ ] Tabella `audit_log` in PostgreSQL

### W3-09 — Checksum dati su ingestion
- **Fonte**: P5
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] SHA-256 checksum per ogni file ingerito
  - [ ] Verifica checksum su lettura
  - [ ] API per checksum di un dataset

### W3-10 — Testing: testify + dockertest + mockery
- **Fonte**: Lib (Go)
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] `testify` per assertions e suite
  - [ ] `dockertest` per integration test con PostgreSQL e DuckDB reali
  - [ ] `mockery` per mock generation
  - [ ] CI esegue unit + integration test

### W3-11 — Connect RPC: error handling strutturato
- **Fonte**: Lib (Go)
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Structured `APIError` type con codice, messaggio, dettagli
  - [ ] Middleware chain per error wrapping
  - [ ] Nessun `fmt.Errorf` in handler — errori wrappati con contesto

---

## ONDA 4 — Trasformazione UX/UI (Terminal-First) 🔗 Dopo W1-01 (Zustand decomposition)

### W4-01 — Design system tokens mancanti
- **Fonte**: P7, P6
- **Severità**: 🟠 ALTA (design system)
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] `design-tokens.json`: elevation (4 livelli), shadow (3), transition (3), border (3 tier)
  - [ ] Tailwind config esteso con tutti i token
  - [ ] `design-system.styles.ts` eliminato (🔗 W5-13)
  - [ ] Nessun valore hardcoded di spacing/shadow/transition nel CSS

### W4-02 — Intento tipografico + densità terminale (FUSO)
- **Fonte**: P6 (W4-01 + W4-05 originale)
- **Severità**: 🟠 ALTA (design system)
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Font body: JetBrains Mono 13px / line-height 1.25
  - [ ] Font meta: JetBrains Mono 11px
  - [ ] `font-variant-numeric: tabular-nums` per alignment dati numerici
  - [ ] Griglia 8px uniforme per spaziatura verticale
  - [ ] `font-variant-ligatures: none` per output terminale
  - [ ] Max-width container per area output (+20% densità caratteri)
  - [ ] WCAG AA contrast verificato su sfondo `#080810` 🔗 W4-08

### W4-03 — Select → Command palette
- **Fonte**: P6
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Tutti i `<select>` sostituiti con command palette con ricerca fuzzy
  - [ ] Navigazione tastiera: frecce, Enter, Escape
  - [ ] Highlight match nella ricerca

### W4-04 — border-radius e stile componenti terminali
- **Fonte**: P6, P7
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] Componenti terminale: `border-radius: 0`
  - [ ] Card/slide-over: `border-radius: 8px` (compound: esterno = interno + padding)
  - [ ] Regola P7: raggio esterno = raggio interno + padding

### W4-05 — Animazione cursore e effetti terminale
- **Fonte**: P6, P7, P10 🔄
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Descrizione**: P6 raccomanda CRT stepped, P7 spring per transazioni strutturali, P10 dice effetti causano affaticamento. Sintesi: stepped deterministic per terminale, spring per slide-over, off di default.
- **Criteri di accettazione**:
  - [ ] Cursore: animazione `steps(2, end)` (non linear)
  - [ ] Slide-over/modali: `cubic-bezier(0.16, 1, 0.3, 1)` (spring)
  - [ ] Output terminale: NO bounce/spring (deterministico)
  - [ ] Scanlines: opacity 0.02-0.04, off di default, `prefers-reduced-motion` rispettato
  - [ ] Toggle setting per effetti terminale 🔗 W5-06

### W4-06 — Command Mode vs Input Mode
- **Fonte**: P6
- **Severità**: 🟠 ALTA (UX)
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Indicatore visivo della modalità (prefisso `:` per command mode)
  - [ ] Transizione chiara tra modalità
  - [ ] Escape da command mode con `Escape`

### W4-07 — Palette dark — shift calore
- **Fonte**: P7
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Sfondo base: `#080810` (warm brown-black)
  - [ ] Surface: `#0e0e18`, Surface-alt: `#141420`
  - [ ] WCAG AA contrast verificato per tutti i testi
  - [ ] Migrare 11 view a dark palette 🔗 PF FASE 7

### W4-08 — Glassmorphism panels
- **Fonte**: P7
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] SlideOverPanel con `backdrop-filter: blur(12px)` + sfondo semi-trasparente
  - [ ] Card interne: sfondo solido per leggibilità
  - [ ] Fallback per browser senza backdrop-filter

### W4-09 — Layer per volatilità CSS
- **Fonte**: P7
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] Static: nessuna animazione (layout, sfondo)
  - [ ] Structural: transizione su mount (fade-in, slide-in, 250ms)
  - [ ] Interactive: transizione su hover/focus (glow, color shift, 150ms)
  - [ ] Signal: animazione su evento (pulse, staggered entry, 50ms)
  - [ ] Documentazione del layer system

### W4-10 — Sistema icone + stati vuoti/errore (FUSO)
- **Fonte**: P7 (W4-12 + W4-13 + W4-14 originale)
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Icone body: 16px/stroke 1.5px; header/sidebar: 20px/stroke 2px (Lucide o Phosphor)
  - [ ] Lista vuota: ghost command prompt `aleph-v2 ❯ _` con suggerimento contestuale
  - [ ] Errori inline: `border-l-4 border-danger bg-danger/5` + 4px left border
  - [ ] Errori toast: icona + messaggio + azione "Riprova"

### W4-11 — Navigazione sidebar ridotta
- **Fonte**: P7
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] Sidebar: icona + label, nessun badge decorativo
  - [ ] Attivo: 2px left border, colore primario
  - [ ] Hover: `bg-surface-alt`, nessuna ombra
  - [ ] Densità: gap-1 items, gap-0.5 sezioni

### W4-12 — App.tsx riscrittura radicale 🔗
- **Fonte**: PF (FASE 3)
- **Severità**: 🟠 ALTA
- **Sforzo**: L
- **Dipendenze**: 🔗 W1-01 (Zustand decomposition), W0-12 (slash commands)
- **Criteri di accettazione**:
  - [ ] Solo CopilotView import statico
  - [ ] Tutte le altre viste `React.lazy`
  - [ ] `renderMain()` eliminato
  - [ ] `prefetchView(viewId)` utility per hover prefetch
  - [ ] Vite manual chunks configurati, budget 150KB gzipped entry

### W4-13 — Modali → SlideOverPanel 🔗
- **Fonte**: PF (FASE 8)
- **Severità**: 🟠 ALTA
- **Sforzo**: L
- **Dipendenze**: 🔗 W4-12 (App.tsx rewrite)
- **Criteri di accettazione**:
  - [ ] 6 viste migrate: Agents, Skills, Tools, DataSources, Library, Components
  - [ ] SlideOverPanel: prop `fullscreen?` + pulsante ⛶
  - [ ] Animazione `max-w-2xl` → `max-w-full` con spring cubic-bezier

### W4-14 — Sidebar + StatusBar refactor 🔗
- **Fonte**: PF (FASE 4-5)
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Dipendenze**: 🔗 W1-01 (store decomposition)
- **Criteri di accettazione**:
  - [ ] Prop `activeTab` rimossa da Sidebar e StatusBar
  - [ ] Sidebar click → `store.setInput('/explore')` + auto-submit
  - [ ] StatusBar: `ALEPH │ {projectID || 'NO PROJECT'} │ {slideOverContext || 'READY'}`

---

## ONDA 5 — Completamento Funzionalità 🔗 Dopo W4-12 (App.tsx rewrite)

### W5-01 — Form creazione agenti mancanti ⚠️
- **Fonte**: P10
- **Severità**: 🔴 CRITICA (blocca utente)
- **Sforzo**: L
- **File**: Frontend `AgentsView`
- **Rischio**: ⚠️ Dipende da schema backend e API
- **Criteri di accettazione**:
  - [ ] Form completo: tipo, nome, descrizione, modello, API key (mascherata)
  - [ ] Validazione client-side e server-side
  - [ ] Integrato con SlideOver panel 🔗 W4-13
  - [ ] Feedback: successo → lista aggiornata, errore → toast con "Riprova"

### W5-02 — Form creazione data source mancanti
- **Fonte**: P10
- **Severità**: 🟠 ALTA
- **Sforzo**: L
- **Criteri di accettazione**:
  - [ ] Form multi-step: upload file / connessione DB / URL con validazione
  - [ ] Integrato con SlideOver panel 🔗 W4-13

### W5-03 — Schermata di benvenuto e onboarding
- **Fonte**: P10
- **Severità**: 🟠 ALTA
- **Sforzo**: L
- **Criteri di accettazione**:
  - [ ] Welcome screen per nuovi utenti
  - [ ] SetupWizard multilingue (non solo italiano)
  - [ ] Dati demo precaricati (dataset "auto")
  - [ ] Agent di default preconfigurati
  - [ ] Guide contestuali per primi comandi

### W5-04 — Vista split, ricerca chat, esportazione
- **Fonte**: P10
- **Severità**: 🟡 MEDIA
- **Sforzo**: L
- **Criteri di accettazione**:
  - [ ] Vista split opzionale: query sinistra, risultati destra
  - [ ] Ricerca full-text nella cronologia chat
  - [ ] Esportazione risultati in CSV/JSON
  - [ ] Bookmark di query e risultati

### W5-05 — Esperienza errore migliorata
- **Fonte**: P10, P7 (W4-14 originale)
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Errori in linguaggio umano (italiano/inglese a seconda impostazione)
  - [ ] Toast con "Riprova" e 15s durata minima
  - [ ] Nessun fallimento silenzioso
  - [ ] Indicatori salute almeno 8px con tooltip
  - [ ] `AlephErrorBoundary` globale + per InlineRenderer + per SlideOver 🔗 PF FASE 12

### W5-06 — Effetti terminale toggle
- **Fonte**: P10, P6, P7 🔄
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] Toggle setting per scanlines, flicker, glow
  - [ ] Effetti OFF di default
  - [ ] `prefers-reduced-motion` rispettato
  - [ ] NESSUN bounce/spring/elastic per output terminale

### W5-07 — Command palette: slash commands + esecuzione
- **Fonte**: P10, PF (FASE 6)
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Slash commands integrati nel command palette
  - [ ] Comandi mutanti richiedono conferma 🔗 W0-12
  - [ ] Autocompletamento con `Tab`
  - [ ] Command history in `sessionStorage` (max 50, no API keys) 🔗 PF FASE 13

### W5-08 — Y.js collaboration migliorata
- **Fonte**: P10
- **Severità**: 🟡 MEDIA
- **Sforzo**: XL ⚠️
- **Rischio**: Richiede design di presenza e conflict resolution — complesso.
- **Criteri di accettazione**:
  - [ ] Presenza utenti online (avatar + cursore colorato)
  - [ ] Chat inline per discussione contestuale
  - [ ] Risoluzione conflitti per editing concorrente
  - [ ] Notifica quando altro utente modifica lo stesso elemento

### W5-09 — fromProto → mappers + Zod schemas
- **Fonte**: P1, P9, Codemem
- **Severità**: 🟠 ALTA
- **Sforzo**: L
- **Criteri di accettazione**:
  - [ ] Zod schemas per ogni tipo proto in arrivo
  - [ ] Mappers tipizzati con validation runtime
  - [ ] Nessun `any` nei tipi di ritorno
  - [ ] Test per ogni mapper con dati validi e invalidi

### W5-10 — TypeScript: eliminare `any`
- **Fonte**: P1, P9, Codemem
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Zero `any` nel codebase frontend
  - [ ] Type-safe adapters per ogni punto di contatto backend
  - [ ] `npm run typecheck` passa senza errori

### W5-11 — Ottimizzare GetDataStats
- **Fonte**: P1
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Ridurre a ≤5 query batch usando `INFORMATION_SCHEMA`
  - [ ] Tempo di risposta ≤200ms per dataset medio
  - [ ] Test con dataset 1M+ righe

### W5-12 — Error handling frontend centralizzato
- **Fonte**: PF (FASE 12), Lib (React)
- **Severità**: 🟠 ALTA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] `handleError` centralizzato: logga a monitoring + toast terminale
  - [ ] `AlephErrorBoundary` globale
  - [ ] `AlephErrorBoundary` per InlineRenderer (isola crash view lazy)
  - [ ] `AlephErrorBoundary` per SlideOverPanel

---

## ONDA 6 — Polish e Ottimizzazione 🔗 Dopo W5

### W6-01 — Eliminare codice morto e residui
- **Fonte**: P1, PF (W5-13, W6-04 originale)
- **Severità**: 🟢 BASSA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] `design-system.styles.ts` eliminato (riferimenti migrati a `design-tokens.json`)
  - [ ] `App.css` eliminato (stili migrati a Tailwind)
  - [ ] Import order corretto in `SetupWizard.tsx` e `LibraryView.tsx`
  - [ ] `cmd/aleph-server/main.go` eliminato 🔗 W0-05

### W6-02 — i18n: stringhe miste
- **Fonte**: PF (FASE 10), P1
- **Severità**: 🟢 BASSA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] UI utente: italiano (con termini tecnici in inglese)
  - [ ] Errori tecnici (gRPC, log): inglese
  - [ ] Traduzioni specifiche: "Visual Glossary" → "Glossario Visivo", etc.
  - [ ] `font-sans` → `font-mono` in LibraryView

### W6-03 — useViewActions refactor
- **Fonte**: PF (FASE 11)
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Ogni dominio ha proprio hook: `useExplorerActions`, `useAgentActions`, etc.
  - [ ] `useViewActions` facade compone i domini
  - [ ] `onRunSkill` → `setSlideOverContent({ type: 'skill' })`
  - [ ] Gestione errori centralizzata in `handleError` 🔗 W5-12

### W6-04 — Yjs cleanup e command history
- **Fonte**: PF (FASE 13)
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] `yMap.delete('activeTab')` one-time eseguito
  - [ ] `commandHistory` in `sessionStorage`, max 50, no API keys
  - [ ] `localStorage` audit: no API keys, tokens, o dati progetto

### W6-05 — shadcn/ui + Radix primitives
- **Fonte**: Lib (React)
- **Severità**: 🟡 MEDIA
- **Sforzo**: L
- **Criteri di accettazione**:
  - [ ] shadcn/iu per componenti base (Button, Input, Dialog, etc.)
  - [ ] Radix primitives per accessibilità (keyboard nav, ARIA)
  - [ ] CVA per varianti stilistiche
  - [ ] Migrazione graduale dei componenti esistenti

### W6-06 — Cursor-based pagination
- **Fonte**: Lib (React)
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] API supporta cursor-based pagination (`after` cursor, `limit`)
  - [ ] Frontend: TanStack Query per caching e infinite scroll
  - [ ] Nessun offset-based pagination per dataset grandi

### W6-07 — SSE per server→client streaming
- **Fonte**: Lib (React)
- **Severità**: 🟡 MEDIA
- **Sforzo**: M
- **Criteri di accettazione**:
  - [ ] Valutare dove SSE è più appropriato di gRPC streaming
  - [ ] Se adottato: endpoint SSE per notifiche e aggiornamenti stato
  - [ ] Frontend: `EventSource` o TanStack Query

### W6-08 — URL state per filtri condivisibili
- **Fonte**: Lib (React)
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] Filtri attivi riflessi nell'URL (es. `?view=explore&filter=active`)
  - [ ] URL condivisibili che ripristinano lo stesso stato
  - [ ] `nuqs` o simile per sincronizzazione URL↔store

### W6-09 — Bundle budget e performance
- **Fonte**: PF (FASE 12)
- **Severità**: 🟡 MEDIA
- **Sforzo**: S
- **Criteri di accettazione**:
  - [ ] Vite manual chunks configurati
  - [ ] Entry budget 150KB gzipped
  - [ ] CI fallisce se superato
  - [ ] Lighthouse CI integrato

### W6-10 — Playwright E2E suite
- **Fonte**: PF (FASE 12)
- **Severità**: 🟠 ALTA
- **Sforzo**: L
- **Dipendenze**: 🔗 Tutte le funzionalità W4/W5 completate
- **Criteri di accettazione**:
  - [ ] `parseCommand()` fuzzing con payload injection
  - [ ] `TerminalOutput` sanitization HTML/ANSI
  - [ ] `SlideOverPanel` apertura/chiusura/fullscreen
  - [ ] Typing commands → assert output
  - [ ] Security regression: XSS injection → assert testo non eseguito
  - [ ] Onboarding → Wizard → Terminale flow

### W6-11 — Bias di processo (META, non codice)
- **Fonte**: P5
- **Severità**: 🟢 BASSA (meta)
- **Sforzo**: S (documentazione)
- **Descrizione**: Non è un item di codice. È un principio di processo.
- **Criteri di accettazione**:
  - [ ] Documento `docs/development-bias-checklist.md` con i bias identificati
  -[ ] Code review template include bias checklist
  - [ ] Priorità: infrastruttura prima di nuove feature

---

## Dipendenze Critiche (Catene)

```
W0 (tutto) → W1
W0-07 (Y.js auth) → W1-01 (Zustand decomp) → W4-12 (App.tsx) → W4-13 (SlideOver) → W5-01 (Forms)
W0-12 (slash commands) → W1-01 (Zustand) → W4-14 (Sidebar ref)
W1-03 (LLM interface) → W2-03 (LLM reasoning)
W2-04 (Brier/Trust decision) → W2-06 (Feedback loop)
W4-07 (Dark palette) → PF FASE 7 (migrate 11 view)
W5-09 (Zod) → W5-10 (eliminate any)
W5 → W6-10 (E2E tests)
```

## Ordine di Esecuzione Consigliato

| Fase | Items | Tempo stimato | Bloccato da |
|------|-------|--------------|-------------|
| **W0** | W0-01 → W0-12 | 2-3 settimane | ⛔ Nessuno — priorità assoluta |
| **W1** | W1-01 → W1-11 | 3-4 settimane | W0 completo |
| **W2** | W2-01 → W2-14 | 2-3 settimane | W1 parziale (può iniziare in parallelo) |
| **W3** | W3-01 → W3-11 | 2-3 settimane | Può iniziare in parallelo con W2 |
| **W4** | W4-01 → W4-14 | 4-5 settimane | W1-01 (Zustand) |
| **W5** | W5-01 → W5-12 | 4-5 settimane | W4-12 (App.tsx), W4-13 (SlideOver) |
| **W6** | W6-01 → W6-11 | 3-4 settimane | W5 completo |

**Totale stimato**: 20-27 settimane (5-7 mesi)

## Sintesi delle Fusioni

| Originale | Fuso in | Motivo |
|-----------|---------|--------|
| W0-01 + W0-11 | W0-01 (SQL injection) | Stesso vettore, stessa fix |
| W0-04 + W1-11 | W0-04 (API key in chiaro + leak) | Stessa vulnerabilità, stessa fix |
| W1-09 + W6-02 | W1-09 (concurrency DuckDB) | P2/P5 conflitto risolto con benchmark |
| W2-04 + W2-05 | W2-04 (Brier/Trust persist o rimuovi) | Stesso problema: calcoli senza consumer |
| W2-08 + W5-09 | W2-08 (UI incertezza) | Prospettive diverse, stessa UI |
| W2-12 + W6-01 | W2-11 (circuit breaker fix o semplifica) | Bug + questione architetturale |
| W4-01 + W4-05 | W4-02 (tipografia + densità) | Stesso dominio |
| W4-12 + W4-13 + W4-14 | W4-12/13/14 | Stesso dominio ma rimasti separati per chiarezza |
| W4-12 + W4-13 + W4-14 | (icone, vuoto, errore) | Stesso dominio visivo |
| W5-13 + W6-04 + W6-05 | W6-01 (codice morto) | Stesso tipo di cleanup |
| Nuovo | W0-12 (slash commands + sanitization) | Critico ma mancante nel piano originale |
