# Aleph-v2 UltraBrain Audit — Piano d'Azione v2.2

> **Audit autor**: Audit completo dello stato attuale di aleph-v2
> **Data**: 2026-04-26
> **Basi analizzate**:
> - Reconciliation plan (1577 righe, W0-W6)
> - Hardening v2.1 (198 righe, W1-W2)
> - LAM Pivot v3 (387 righe, F1-F7)
> - Codice sorgente: 173 file backend+frontend
>
> **Principio**: Build fixes prima di tutto. Infrastruttura prima della logica. Hardening prima delle nuove feature.

---

## 1. ANALISI DELLO STATO REALE

### Build Status

| Componente | Stato |
|---|---|
| `go build ./...` | Pulisce |
| `go test ./...` | MCP 14/14 pass, Memory 10/10 pass, Handler parziale (query_test.go Chat loop NON testato) |
| `npx tsc --noEmit` | **10 errori TypeScript in store test** — `StateCreator<T>` richiede 3 argomenti, i test ne passano 1 |
| `npx vitest run` | 117/117 test passano (ma TS rotto) |
| `npx playwright test` | NON VERIFICATO — Piano W6-10 richiede 21/21 |

> **Bug critico scoperto**: `vitest run` passa (JavaScript runtime non controlla argomenti), ma `tsc --noEmit` fallisce. Se pusha senza tsc check, il CI si rompe.

---

### Cosa manca? (Gap Analysis)

#### Architettura
1. **Decision Loop (F3) INCOMPLETO** — scheletro, non LAM
   - `plannedTool` manca `rationale`, `expected`, `fallback` (F3-01)
   - Observe = string match "Errore", non valutazione LLM (F3-02)
   - Re-plan non esiste come fase separata — solo contatore failure max 2 (F3-02)
   - AdmitFailure = string trimming minimale (F3-03)
   - Decision trace spans NON tracciano alternative scartate (F3-04)

2. **Telemetry traces mancanti** — telemetry/ ha `telemetry.go` e `middleware.go` ma zero `traces.go` per decision spans. Chat() crea spans generiche ma mancano: tool alternativi, scores, divergenza.

3. **NotificationService.Stop() NON collegato** — metodo esiste ma NON chiamato da `app.Close()` (W1-01 parziale).

4. **tool_suggest cleanup bug** — goroutine cleanup (riga 322) usa `r.Context()` (richiesta) anziche `app.ctx` (server). Su shutdown veloce il context si annulla subito, cleanup non eseguito.

5. **Ingestion ctx propagation** — Engine inizializzato con `a.ctx` ma `eng.Close()` chiude storage, non cancella context interno. Nessun `Stop()`.

6. **DuckDB backup locking** — `AutoBackup()` usa `Lock()` esclusivo in `duckdb_backup.go`. Blocca RLock in lettura durante backup lungo.

#### Feature
7. **Vitest config mancante** — `vitest.config.ts` non esiste (W1-07 non fatto)
8. **Frontend store tests falliscono a TypeScript** — 6 test con `StateCreator` type mismatch
9. **SlideOver null safe** — `VIEW_REGISTRY[content.type]` restituisce undefined per tipo sconosciuto → crash
10. **NLP dead code** — piano W2-01 richiede delete ma codice probabilmente ancora esiste
11. **gRPC proto fix** — `nlp_pb2_grpc.py` import errato (W2-02 non fatto)
12. **Honest responses** — `analyze_sentiment` potrebbe restituire score fittizio se modello non allenato
13. **ToolCodeWriter senza ctx** — interfacce `codeflow/` senza `context.Context` (W1-05)
14. **Hook tests** — 15 hook file, zero test (W2-06)
15. **Component tests** — solo 4 form test esistenti (W2-08)

#### Cosa non funziona?
16. **TypeScript build rottura**: Il codice JS esegue ma il type check fallisce → CI/CD rotto
17. **Observe phase bug**: Se resultStr contiene "Errore" nel contenuto normale (JSON con campo "errore"), falso positivo failure
18. **AdmitFailure leak**: `h.db.Cleanup()` chiamato in loop dopo ogni tool e poi di nuovo in AdmitFailure — double cleanup
19. **Memory embedding hardcode**: `NewEmbedder("http://localhost:11434", "")` in query.go riga 792 — nessun config injection
20. **Tool registry race**: `toolRegistry.RegisterAll()` senza lock in app.go (righe 265-317) ma `ExecuteContext` prende RLock — registrazione concorrente potenzialmente unsafe

#### Cosa ci siamo dimenticati?
21. **MCP discovery non cancellabile** — `a.discoveryEngine.Start(a.ctx)` lanciata in goroutine ma se fallisce subito, errore solo WARN log
22. **Health checker start senza stop** — `healthChecker.Start(a.ctx)` in goroutine ma manca stop sequence nel Close appropriato (Stop() chiamato ma non attende)
23. **SSE broker close** — `sseBroker.Close()` chiamato in Close() ma non prima del server shutdown (riga 393-398) — client potrebbero ricevere EOF anziche chiusura graceful
24. **W5 e W6 reconciliation** — Segnano 16/16 e 16/16 completati ma piani non presenti nel reconciliation (solo nel riepilogo). W5 e W6 sono stati eseguiti ma NON documentati nel piano come item individuali.
25. **Cursor pagination** — W6-06 completato ma da verificare se usato in frontend (useInfiniteQueries usa offset, non cursor?)

---

## 2. PIANO D'AZIONE DETTAGLIATO

> **Ordine**: Build fixes → Hardening W1/W2 → Fase 3 (Decision Loop) → Fase 4+ (Multi-Agent, Security, Evaluation)
>
> **Regole wave**:
> - Ogni wave sequenziale (wave N+1 inizia dopo wave N)
> - Task indipendenti dentro wave = parallelizzabili (PP)
> - Dipendenze esplicite per ogni task

---

### WAVE 1: Stop the Bleeding (Build Fixes + Hardening W1)

> **Obiettivo**: `go build`, `go test`, `tsc --noEmit`, `vitest run` TUTTI puliti.
> **Tema**: Build integrity. Se il build non passa, nulla serve.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W1-01 | **Fix TypeScript store tests** — Cambiare tipo slice creators da `StateCreator<T>` a `SliceCreator<T> = (set: any) => T` (custom type con 1 arg). Oppure passare `set, ()=>{}, {}` nei test. | `frontend/src/store/*Slice.ts`, `__tests__/*Slice.test.ts` | S | Si | — | W1-02,W1-03 |
| W1-02 | **Creare vitest.config.ts** — Coverage reporter, jsdom, timeout 10s, test patterns. | `frontend/vitest.config.ts` (nuovo) | S | Si | — | W1-03,W1-04 |
| W1-03 | **Fix SlideOver null-safe** — Aggiungere `if (!ViewComponent) return null` prima del render. | `frontend/src/App.tsx` riga 468 | S | Si | W1-01 | W1-04 |
| W1-04 | **Verifica build frontend** — `npx tsc --noEmit` deve passare. `npx vitest run` deve passare con coverage report. | — (verifica) | S | No | W1-01,W1-02,W1-03 | W2-01 |

**Build check W1**:
```bash
npx tsc --noEmit     # deve passare
npx vitest run       # deve passare
npx vite build       # deve passare
```

---

### WAVE 2: Hardening W1 (Integrita Strutturale)

> **Obiettivo**: Goroutine leak fix, God Object parziale, cleanup interruptibile, DuckDB lock.
> **Tema**: Integrita strutturale. Codice esistente robusto.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W2-01 | **NotificationService Stop wiring** — Chiamare `notificationSvc.Stop()` in `app.Close()` prima di chiudere server. | `internal/app/app.go` riga 379-407 | S | Si | W1-04 | W2-05 |
| W2-02 | **tool_suggest cleanup ctx fix** — Sostituire `r.Context()` con context persistente (app.ctx) nella goroutine cleanup. | `internal/api/handler/tool_suggest.go` riga 322 | S | Si | W1-04 | W2-05 |
| W2-03 | **Ingestion ctx propagation** — Aggiungere `Stop()` al Engine che cancella il context. Chiamare da `app.Close()`. | `internal/ingestion/ingestion.go`, `app.go` | S | Si | W1-04 | W2-05 |
| W2-04 | **app.go God Object parziale** — Estrarre `routes.RegisterRoutes` gia esiste (bene). Verificare che `internal/routes/` sia completo e non ci siano route sparsi in app.go. Aggiungere commento TODO per futuro split. | `internal/app/app.go`, `internal/routes/routes.go` | M | No | W1-04 | W2-05 |
| W2-05 | **DuckDB backup lock fix** — Cambiare `Lock()` in `RLock()` + snapshot-read per backup. Documentare gerarchia lock. | `internal/storage/duckdb_backup.go` | S | Si | W2-01,W2-02,W2-03,W2-04 | W2-06 |
| W2-06 | **Build check W2** | — | — | No | W2-05 | W3-01 |

---

### WAVE 3: Hardening W2 (Onesta + Frontend Tests)

> **Obiettivo**: Onesta profonda, test frontend, fix NLP.
> **Tema**: Onesta. Nessun output fabbricato. Test effettivi.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W3-01 | **DELETE NLP dead code** — Eliminare `ensemble.py`, `calibration.py`, `predict.py`, `convert_onnx.py`, `*.onnx`. Tenere solo gRPC server, `nlp_pb2.py`, base requirements. | `nlp/` | S | Si | W2-06 | W3-05 |
| W3-02 | **Fix gRPC proto import** — Check-in .proto, rigenerare pb2, fixare import path in `nlp_pb2_grpc.py`. | `nlp/*.proto`, `nlp_pb2_grpc.py` | S | Si | W2-06 | W3-05 |
| W3-03 | **Honest responses** — Aggiungere check: se `nlpHandler` non restituisce score con confidenza, rispondere "NLP prediction confidence unavailable — model not trained" invece di 0.5. | `internal/api/handler/query.go` (analyze_sentiment path), `internal/nlp_adapter/adapter.go` | S | Si | W2-06 | W3-05 |
| W3-04 | **SlideOverPanel triage + registry map** — Convertire switch-case in registry map gia fatto (VIEW_REGISTRY). Pulire stub/commenti. Documentare 17 casi. | `frontend/src/App.tsx` | S | Si | W2-06 | W3-05 |
| W3-05 | **Hook unit tests** — Testare useAppActions, useChat, useSSE, useToolActions, useViewActions, useSlideOver con mock store. | `frontend/src/hooks/__tests__/*.test.ts` | M | Si | W3-01,W3-02,W3-03,W3-04 | W3-06 |
| W3-06 | **Store slice tests completi + coverage gate** — Verificare che ogni slice abbia coverage >= 90%. Correggere eventuali edge case mancanti. | `frontend/src/store/__tests__/` | S | Si | W3-05 | W3-07 |
| W3-07 | **Build check W3** | — | — | No | W3-06 | W4-01 |

---

### WAVE 4: Decision Loop — Planner Raffinato (F3-01)

> **Obiettivo**: Il Decision Loop ha un vero Planner con intent parsing, rationale, expected, fallback.
> **Tema**: Fondazione LAM. Senza questo, F4-F7 sono costruiti su sabbia.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W4-01 | **Redefinire plannedTool struct** — Aggiungere campi: `Rationale string`, `ExpectedOutput string`, `FallbackTool string`, `FallbackParams map[string]interface{}`. | `internal/api/handler/query.go` (type plannedTool riga 909) | S | Si | W3-07 | W4-02 |
| W4-02 | **Intent parsing ricorsivo in Plan** — LLM call per decomporre intent complesso. Prompt richiede JSON con `steps[].{tool,params,expected,rationale,fallback}`. | `internal/api/handler/query.go` (Plan phase riga 915-962) | M | Si | W4-01 | W4-03 |
| W4-03 | **AdmitFailure precoce** — Se il planner restituisce "[]" (nessun tool adatto) o nessun tool disponibile, rispondere subito "Non posso completare questa richiesta perche [motivo]" senza eseguire tool. | `internal/api/handler/query.go` (dopo Plan phase) | S | Si | W4-02 | W4-04 |
| W4-04 | **Build + test Planner** | `internal/api/handler/query_test.go` (aggiungere test Chat planner) | S | No | W4-03 | W5-01 |

**Build check W4**: `go test ./internal/api/handler/ -run TestChatPlanner`

---

### WAVE 5: Decision Loop — Observe + Reflect + Re-plan (F3-02)

> **Obiettivo**: Observe valuta davvero l'output. Reflect ricalibra. Re-plan loop strutturato.
> **Tema**: Autenticita decisionale. Niente string match.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W5-01 | **Observe vera (LLM eval)** — Dopo ogni tool execution, LLM call che confronta output con `expected`. Se divergenza significativa, lo registra. Prompt: "Confronta output con expected output. Restituisci JSON {diverges: bool, reason: string, confidence: 0-1}". | `internal/api/handler/query.go` (Observe phase riga 1053-1081) | M | Si | W4-04 | W5-02 |
| W5-02 | **Reflect + Re-plan** — Se divergenza, ricalibrare piano: chiedere a LLM nuova sequenza tool con motivazione. Max 2 re-plan. | `internal/api/handler/query.go` (dopo Observe) | M | Si | W5-01 | W5-03 |
| W5-03 | **AdmitFailure strutturato** — Dopo max 2 re-plan falliti, rispondere con motivo specifico: "Non posso completare perche [divergenza motivo]" + trace delle decisioni. NON inventare successo. | `internal/api/handler/query.go` (AdmitFailure phase riga 1083-1107) | S | Si | W5-02 | W5-04 |
| W5-04 | **Test Observe/Reflect/Re-plan** — Test con mock LLM che simula output inaspettato → sistema ricalibra → secondo tentativo → admit se fallisce. | `internal/api/handler/query_test.go` | M | No | W5-03 | W6-01 |

**Build check W5**: `go test ./internal/api/handler/ -run TestChatObserveReflect`

---

### WAVE 6: Decision Trace + Telemetry Spans (F3-04)

> **Obiettivo**: Ogni decisione del LAM e tracciabile: tool considerati, scartati, motivi, divergenza.
> **Tema**: Trasparenza. Senza trace, il LAM e una scatola nera.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W6-01 | **Creare internal/telemetry/traces.go** — Span builder specializzato per decision loop: `StartDecisionSpan`, `AddToolCandidate(name, score, selected bool)`, `AddDivergence(expected, actual, reason)`, `AddReplanAttempt(attempt, newPlanJSON)`. | `internal/telemetry/traces.go` (nuovo) | M | Si | W5-04 | W6-02,W6-03 |
| W6-02 | **Instrumentare Chat() con decision spans** — Sostituire spans generiche attuali con spans strutturate. In Plan: tool candidates + rationale. In Act: tool eseguito + params. In Observe: divergenza. In Reflect: replan decisione. In Admit: motivo + trace completo. | `internal/api/handler/query.go` | M | Si | W6-01 | W6-04 |
| W6-03 | **Test decision trace spans** — Verificare che trace contenga Plan -> Act -> Observe -> Reflect -> Admit completo con tool alternatives. | `internal/telemetry/telemetry_test.go` (estendere) | S | Si | W6-01 | W6-04 |
| W6-04 | **Build check W6** |

---

### WAVE 7: Tool Interfaces + Onesta Finale (v2.1 residue)

> **Obiettivo**: Completare gli item residuali del piano hardening v2.1.
> **Tema**: Completare il debito.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W7-01 | **Add context.Context a ToolCodeWriter/Reader** — Aggiungere `ctx context.Context` a tutti i metodi delle interfacce. Aggiornare implementazioni e caller. | `internal/tools/codeflow/*.go`, `internal/api/handler/tool.go` | S | Si | W6-04 | W7-02 |
| W7-02 | **Component tests — 10 componenti critici** | `frontend/src/components/__tests__/*.test.tsx` | M | Si | W6-04 | W7-03 |
| W7-03 | **Hook integration tests** | `frontend/src/hooks/__tests__/*.integration.test.ts` | M | Si | W7-02 | W7-04 |
| W7-04 | **Build check W7** — Frontend coverage globale >= 50%. Playwright 21/21. | — | — | No | W7-03 | W8-01 |

---

### WAVE 8: Multi-Agent A2A Protocol (F4)

> **Obiettivo**: Comunicazione agente-agente via A2A. Solo DOPO single-agent LAM funzionante.
> **Tema**: Espansione multi-agente. Dipende da F3 completo.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W8-01 | **A2A RFC + spec** — Scrivere `docs/a2a-protocol.md`: identity, message envelope, task lifecycle, capability discovery. Approvazione prima di codice. | `docs/a2a-protocol.md` | M | Si | W7-04 | W8-02 |
| W8-02 | **A2A transport** — HTTP/JSON envelope, routing `/a2a/`, timeout 10s per hop. | `internal/a2a/transport.go` | M | Si | W8-01 | W8-03 |
| W8-03 | **Agent capability registry** — Registrazione capabilities al boot, routing. | `internal/a2a/registry.go` | S | Si | W8-02 | W8-04 |
| W8-04 | **AgentOrchestrator** — Refactor Chat() in orchestrator: intent -> route ad agenti -> merge output -> rilevamento conflitti. | `internal/orchestrator/orchestrator.go` | L | No | W8-03 | W8-05 |
| W8-05 | **Multi-agent test suite** — 2 agent collaborativi, 2 agent con conflitto, 3 agent merge. | `internal/orchestrator/orchestrator_test.go` | M | No | W8-04 | W9-01 |

---

### WAVE 9: Sicurezza (F5)

> **Obiettivo**: Policy engine + Judge advisory + defense in depth.
> **Tema**: Sicurezza. Judge solo advisory. Policy engine authority di block.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W9-01 | **Policy engine** — YAML rules -> compiled policy. Per-tool constraints (allowed params, rate limit, max calls). | `internal/policy/engine.go`, `internal/policy/rules.go` | M | Si | W8-05 | W9-02,W9-05 |
| W9-02 | **Judge Model advisory** — LLM call separata che reviewa tool selection (warn, NON block). Traccia in OTEL span. | `internal/judge/judge.go` | M | Si | W9-01 | W9-03 |
| W9-03 | **Defense in depth chain** — Policy check -> Judge review -> Execution -> Output sanitization. Tutto in decision trace. | `internal/api/handler/query.go` (o orchestrator) | M | Si | W9-02 | W9-04 |
| W9-04 | **Policy compliance test suite** — Input avversari contro policy rules. | `internal/policy/policy_test.go` | M | Si | W9-03 | W9-05 |
| W9-05 | **Sandbox Verifier nel decision loop** — Valida tool code PRIMA del dispatch. Se malevolo -> policy block. | `internal/sandbox/verification.go`, `internal/orchestrator/` | S | Si | W9-01 | W9-06 |
| W9-06 | **Build check W9** |

---

### WAVE 10: Valutazione (F7)

> **Obiettivo**: Prova che funziona. Ship con metriche.
> **Tema**: Quality gates. Senza eval, e solo speranza.

| ID | Task | File | Effort | PP? | Dipende da | Blocca |
|---|---|---|---|---|---|---|
| W10-01 | **E2E benchmark framework** — Task suite con scoring automatico: RAG query, tool exec, multi-step plan, multi-agent, adversarial. | `benchmark/suite/` | L | Si | W9-06 | W10-02 |
| W10-02 | **Decision trace audit tool (backend)** — Replay decisioni con filtri per agente/tool/outcome. | `internal/telemetry/replay.go` | S | Si | W10-01 | W10-03 |
| W10-03 | **Decision trace dashboard (frontend)** — Albero decisionale visuale, export JSON. | `frontend/src/views/DecisionTraceView.tsx` | M | Si | W10-02 | W10-04 |
| W10-04 | **Load test multi-agente** — 10+ sessioni concorrenti. Metrics: goroutine leak, latency P50/P95, throughput, mem/session. | `benchmark/load/` | M | Si | W10-03 | W10-05 |
| W10-05 | **Frontend coverage target** | — | M | Si | W10-04 | W10-06 |
| W10-06 | **Rollback + migration plan v2->v3** — Schema DB changes, API backward compat, rollback procedure, feature flags. | `docs/plans/v2-to-v3-migration.md` | S | Si | W10-05 | W10-07 |
| W10-07 | **Build check W10** |

---

## 3. RIEPILOGO

| Wave | Tema | Task | Quick | Medium | Large | Stima Totale |
|---|---|---|---|---|---|---|
| W1 | Build Fixes | 4 | 4 | — | — | 0.5g |
| W2 | Hardening W1 | 6 | 4 | 1 | — | 1g |
| W3 | Hardening W2 + Tests | 7 | 3 | 3 | — | 2g |
| W4 | Planner (F3-01) | 4 | 2 | 1 | — | 1.5g |
| W5 | Observe/Reflect (F3-02) | 4 | 1 | 2 | — | 2g |
| W6 | Decision Trace (F3-04) | 4 | 1 | 2 | — | 1.5g |
| W7 | Residuali v2.1 | 4 | — | 2 | — | 1.5g |
| W8 | Multi-Agent (F4) | 5 | 1 | 2 | 1 | 4g |
| W9 | Security (F5) | 6 | 1 | 4 | — | 3g |
| W10 | Evaluation (F7) | 5 | 1 | 3 | 1 | ~4g |
| **Totale** | | **45** | **18** | **20** | **2** | **~20g** |

> **Nota**: Spike multimodale (F6) escluso — non gate per ship, budget 5gg opzionali.

---

## 4. CHECKLIST PRIORITA

### Prima di iniziare qualsiasi nuova feature:
- [ ] W1 passa (tsc pulito)
- [ ] W2 passa (goroutine leak fixed)
- [ ] W3 passa (test coverage >= 50%)

### Prima di F4 (Multi-Agente):
- [ ] F3 completo (W4+W5+W6 passate)
- [ ] Decision trace funzionante
- [ ] AdmitFailure testato

### Prima di ship v3:
- [ ] W9 passa (security)
- [ ] W10 passa (eval + benchmark)
- [ ] `go build ./...` pulito
- [ ] `go test ./...` tutti passano (con race)
- [ ] `npx tsc --noEmit` pulito
- [ ] `npx vitest run --coverage` >= 60%
- [ ] `npx playwright test` 21/21

---

## 5. DIPENDENZE GRAFICO

```
W1 (Build Fixes)
  |
W2 (Hardening W1)
  |
W3 (Hardening W2 + Tests)
  |
W4 (Planner F3-01) --+-- W6 (Trace F3-04)
  |                    |
W5 (Observe F3-02) ---+
  |
W7 (Residuali v2.1)
  |
W8 (Multi-Agent F4)
  |
W9 (Security F5)
  |
W10 (Evaluation F7)
```

**Regole**:
- Freccia = dipendenza (la destinazione non puo iniziare prima della sorgente)
- W4 e W6 possono procedere in parallelo dopo W3
- Fase 6 (Multimodal) e Fase 7 (Evaluation) sono opzionali — ship con F5

---

*Generato da UltraBrain Audit su aleph-v2. Stima ~20 giorni lavorativi (4 settimane) per 1 sviluppatore full-time. Con parallelizzazione e agent multipli: ~2-2.5 settimane.*
