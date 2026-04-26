# aleph-v3: LAM Pivot

> **Prerequisito**: `a-v2.1-hardening-2.md` completato (W1 + W2)
> **Base**: LAM paradigm (Large Action Model) — percezione → decisione → azione → osservazione → verifica
> **Review**: Momus ✅ | Aleph ✅ (4 condizioni) | Metis ✅ (blind spot corretti)
> **Principio**: Single-agent LAM funzionante PRIMA di multi-agente

---

## Condizioni Aleph (vincolanti)

1. **Multi-agente SOLO DOPO** Decision Loop (Fase 3) e Memoria (Fase 3) funzionanti
2. **Judge Model (Fase 5) solo ADVISORY** — mai blocco. Blocco richiede policy engine (Fase 5) come gate separato
3. **AdmitFailure esplicito** — ogni Decision Loop deve poter dire "Non posso" invece di inventare successo
4. **Multimodal e Computer-use (Fase 6) come SPIKE opzionali** — non gate per ship

---

## Fasi

```
Fase 1: MCP Completo ──→ Fase 2: Memoria ──→ Fase 3: Decision Loop ──→ Fase 4: Multi-Agente ──→ Fase 5: Sicurezza ──→ Fase 6: Multimodale ──→ Fase 7: Valutazione
                                                                                                            (spike)
```

Le fasi sono SEQUENZIALI e indipendenti. Ogni fase produce un sistema shipabile.

---

## Fase 1: MCP Completo (8 task)
> *Il protocollo tool deve parlare JSON-RPC 2.0 su STDIO prima che qualsiasi LAM possa usarlo.*

### F1-01: MCP JSON-RPC 2.0 envelope
- **File**: `internal/mcp/schemas.go`, `internal/mcp/transport.go`
- **Cosa**: Implementare envelope JSON-RPC 2.0: `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}` con error codes standard (-32700 parse error, -32600 invalid request, -32601 method not found, -32603 internal error)
- **Verifica**: `go test ./internal/mcp/` — valid JSON-RPC request/response round-trip
- **Stima**: Medium (~4h)

### F1-02: MCP STDIO transport
- **File**: `internal/mcp/transport_stdio.go`
- **Cosa**: STDIO transport per tool locali: stdin read (line-delimited JSON), stdout write (JSON-RPC response), stderr per log. Subprocess lifecycle (start, health, stop).
- **Verifica**: Tool MCP STDIO funzionante da CLI: `echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./tool-mcp-server`
- **Stima**: Medium (~4h)

### F1-03: MCP tool discovery via `tools/list`
- **File**: `internal/mcp/discovery.go`
- **Cosa**: Implementare `tools/list` che restituisce strumenti registrati con nome, descrizione, parametri (JSON Schema), capabilities. Unificare con discovery engine esistente.
- **Verifica**: `tools/list` restituisce lista completa tool → test PASS
- **Stima**: Medium (~3h)

### F1-04: MCP tool execution via `tools/call`
- **File**: `internal/mcp/handler.go`
- **Cosa**: Implementare `tools/call` con validazione parametri, timeout, error standard. Unificare con tool_exec.go.
- **Verifica**: `tools/call` con parametri validi → risultato. Parametri invalidi → JSON-RPC error.
- **Stima**: Medium (~3h)

### F1-05: Tool registry — unico punto di registrazione
- **File**: `internal/tools/registry.go` (new)
- **Cosa**: Estrarre la registrazione tool da app.go e handler sparsi in un unico ToolRegistry. Ogni tool si registra con nome, categoria, parametri, STDIO command (se locale), HTTP endpoint (se remoto).
- **Verifica**: `go build ./...` — tutti i tool registrati via registry, non via app.go diretto
- **Stima**: Short (~1h)

### F1-06: Tool registry REST API
- **File**: `internal/api/handler/tool_exec.go`, `internal/routes/` (da v2.1)
- **Cosa**: Esporre ToolRegistry via REST: `GET /api/v1/tools` (list), `GET /api/v1/tools/{name}` (detail), `POST /api/v1/tools/call` (execute). Sostituire l'attuale dispatcher basato su switch/case.
- **Verifica**: `curl /api/v1/tools/call -d '{"tool":"finance.prophet_forecast","params":{}}'` → result o error
- **Stima**: Short (~1h)

### F1-07: MCP health check — tool liveliness
- **File**: `internal/mcp/health.go`
- **Cosa**: STDIO tool health check: ping/pong su subprocess, restart su crash. Integrare con health checker esistente.
- **Verifica**: Kill subprocess → health checker rileva e restart → ok
- **Stima**: Short (~1h)

### F1-08: MCP test suite
- **File**: `internal/mcp/mcp_test.go`
- **Cosa**: Test suite completa: JSON-RPC envelope valid/invalid, STDIO trasporto, tool discovery, call, errori, subprocess lifecycle, timeout
- **Verifica**: `go test -v -race ./internal/mcp/` → coverage ≥ 80%
- **Stima**: Short (~1h)

**Build check Fase 1**:
```bash
go build ./...
go test -v -race ./internal/mcp/
npx playwright test    # invariato
```

---

## Fase 2: Memoria (4 task)
> *Prima di decidere, il sistema deve ricordare.*

### F2-01: Backend selection + vector store
- **File**: `internal/memory/store.go` (new)
- **Cosa**: Scegliere backend (pgvector se Postgres disponibile, altrimenti SQLite-vec). Creare interfaccia `MemoryStore { Insert, Search, Delete, Namespace }`. Implementare per backend scelto.
- **Criteri**: Zero nuove runtime deps (SQLite-vec se Go+nome, pgvector se già Postgres). No Python deps.
- **Verifica**: `go test ./internal/memory/` — insert + nearest neighbor search funziona
- **Stima**: Medium (~4h)

### F2-02: Embedding pipeline
- **File**: `internal/memory/embed.go`
- **Cosa**: Chunking (recursive character split, 512 token, 128 overlap) → embedding API call (Ollama via llm provider esistente) → store. Pipeline per document ingestion.
- **⚠️ Scope boundary**: NO chunking semantico (richiederebbe NLP). NO multi-lingua. Solo text splitting.
- **Verifica**: Testo in → chunk + embedding + stored → retrieve
- **Stima**: Medium (~4h)

### F2-03: Namespace isolation (forward-compat)
- **File**: `internal/memory/store.go`
- **Cosa**: Namespace per isolamento (progetto, agente, sessione). Hooks per policy engine futuro. Forward-compatibile per W4 multi-agente namespaces.
- **Verifica**: Insert con namespace A → Search in namespace B → 0 risultati
- **Stima**: Short (~1h)

### F2-04: Retrieval in Chat() context
- **File**: `internal/api/handler/query.go` (Chat)
- **Cosa**: Prima di Plan, retrieve memory pertinente dalla conversazione corrente e dal progetto. Iniettare nel system prompt come contesto.
- **Verifica**: Chat ricorda informazioni da conversazioni precedenti nello stesso progetto
- **Stima**: Short (~2h)

**Build check Fase 2**:
```bash
go build ./...
go test -v -race ./internal/memory/
npx vitest run --coverage   # invariato da v2.1
```

---

## Fase 3: Decision Loop (4 task)
> *Il cuore del LAM: plan→act→observe→reflect→admit.*

### F3-01: Planner stage
- **File**: `internal/api/handler/query.go` (Chat)
- **Cosa**: Prima di eseguire tool, LLM call per decomporre intent in tool sequence con dipendenze:
  1. Intent parsing
  2. Tool selection con motivazione
  3. Piano: ordered steps con attesi output intermedi
  4. AdmitFailure: se nessun tool adatto, rispondere subito "Non posso"
- **Formato piano**: JSON strutturato: `{steps: [{tool, params, expected, fallback}], rationale: "..."}`
- **Verifica**: Chat con richiesta multi-tool → piano visibile in decision trace (F3-04)
- **Stima**: Medium (~6h)

### F3-02: Observe + Reflect stages
- **File**: `internal/api/handler/query.go`
- **Cosa**: Dopo ogni Act (tool execution):
  - **Observe**: LLM valuta output tool vs expected. Se divergenza → register in decision trace.
  - **Reflect**: Ricalibrare piano se necessario. Max 2 re-plan prima di AdmitFailure.
- **Verifica**: Tool produce output inaspettato → sistema ricalibra → seconda risposta + admit se fallisce
- **Stima**: Medium (~4h)

### F3-03: AdmitFailure esplicito
- **File**: `internal/api/handler/query.go`
- **Cosa**: Dopo 2 re-plan falliti o nessun tool applicabile, rispondere: "Non posso completare questa richiesta perché [motivo specifico]." MAI output parziale come "successo". MAI score fabbricati.
- **Verifica**: Richiesta impossibile → AdmitFailure con motivo → test PASS
- **Stima**: Quick (~30min)

### F3-04: Decision trace spans
- **File**: `internal/api/handler/query.go`, `internal/telemetry/traces.go`
- **Cosa**: OTEL span per ogni step: Plan (intent, piano, rationale), Act (tool, params, output, latency), Observe (divergenza), Reflect (re-plan decisione). Tool alternativi scartati + perché.
- **Verifica**: `go test -v ./internal/telemetry/` — trace contiene Plan→Act→Observe→Reflect→Admit
- **Stima**: Short (~2h)

**Build check Fase 3**:
```bash
go build ./...
go test -v -race ./internal/api/handler/...  # chat loop test
go test -v -race ./internal/telemetry/...    # trace test
npx playwright test
```

---

## Fase 4: Multi-Agente (5 task)
> *Solo dopo che single-agent LAM funziona (Fase 3).*

### F4-01: A2A Protocol — RFC + spec
- **Fonte**: LAM gap — nessuna comunicazione agente→agente
- **File**: `docs/a2a-protocol.md` (new), `internal/a2a/spec.go` (new)
- **Cosa**: PRIMA di implementare, scrivere RFC interna:
  - Agent identity (name, version, capabilities[], trust_level)
  - Message envelope (from, to, intent_id, payload_type, context_refs[], trace_id)
  - Task lifecycle (request → accept → progress → complete / fail / reject)
  - Capability discovery e negoziazione
  - **Revisione**: Almeno 2 persone/gruppi devono revieware prima di codice
- **Verifica**: RFC approvata → start implementation
- **Stima**: Medium (~3h spec + 3h revisione)

### F4-02: A2A transport (HTTP/JSON)
- **File**: `internal/a2a/transport.go`
- **Cosa**: Implementare A2A su HTTP esistente (stessa porta, routing `/a2a/`). JSON envelope. Timeout 10s per hop.
- **Verifica**: 2 agent processi si scambiano messaggi A2A via HTTP
- **Stima**: Medium (~4h)

### F4-03: Agent capability registry
- **File**: `internal/a2a/registry.go`
- **Cosa**: Ogni agente si registra con capabilities dichiarate al boot. Registry centrale per routing. Policy engine (future F5) può bloccare azione fuori scope.
- **Verifica**: 2 agent con capabilities diverse → registry le distingue → routing corretto
- **Stima**: Short (~2h)

### F4-04: AgentOrchestrator
- **File**: `internal/orchestrator/orchestrator.go` (new)
- **Cosa**: Refactor Chat() → orchestrator:
  - Inbound intent → analyze → decompose → route a agenti specializzati
  - Merge output da agenti multipli
  - Rilevamento e risoluzione conflitti (risposte contraddittorie)
  - Shared memory access scoped (namespace F2-03)
- **⚠️ Scope boundary**: Orchestrator NON implementa planner — chiama Decision Loop (F3) di ogni agente
- **Verifica**: 3 agent completano task collaborativo → output unificato corretto
- **Stima**: **Large** (~3-5gg)

### F4-05: Multi-agent test suite
- **File**: `internal/orchestrator/orchestrator_test.go`
- **Cosa**: Scenari: 2 agent collaborativi, 2 agent con conflitto, 3 agent con merge. Verifica context merge, memory isolation, decision trace.
- **Verifica**: `go test -v -race ./internal/orchestrator/`
- **Stima**: Medium (~4h)

**Build check Fase 4**:
```bash
go build ./...
go test -v -race ./internal/a2a/
go test -v -race ./internal/orchestrator/
npx playwright test    # invariato
```

---

## Fase 5: Sicurezza (6 task)
> *Policy e Judge. Solo ADVISORY per Judge, BLOCK per policy engine.*

### F5-01: Policy engine — YAML rules → compiled policy
- **File**: `internal/policy/engine.go`, `internal/policy/rules.go` (new)
- **Cosa**: Policy-as-code: YAML rules con condizione + azione (allow / deny / warn). Per-tool constraints (allowed params, rate limit, max calls). Per-agent constraints (scope, resource limits).
- **Verifica**: Policy rule "deny tool X per agente Y" → tool call bloccato
- **Stima**: Medium (~4h)

### F5-02: Judge Model — advisory review layer
- **File**: `internal/judge/judge.go` (new)
- **Cosa**: LLM call separata che reviewa:
  - Tool selection prima della risposta (advisory — logga warning, NON blocca)
  - Output per policy compliance (warn se violazione)
  - Traccia decisioni in OTEL span
- **⚠️ Condizione Aleph**: Solo advisory. Blocco = policy engine (F5-01), non judge model.
- **Verifica**: Judge warn su violazione → output passa comunque (advisory) → test PASS
- **Stima**: Medium (~4h)

### F5-03: Defense in depth — catena per-tool
- **File**: `internal/orchestrator/chat.go` (o query.go)
- **Cosa**: Ogni tool dispatch passa per: Policy Engine check (allow/deny) → Judge Model review (warn) → Execution → Output sanitization. Tutti registrati in decision trace.
- **Verifica**: Disattivare policy gate → test fallisce. Judge warn → test passa (advisory).
- **Stima**: Medium (~3h)

### F5-04: AST-based security scanner
- **File**: `internal/scanner/` (new o estendere sandbox/)
- **Cosa**: Sostituire regex scan (SecurityScan) con tree-sitter semgrep o equivalente. Pattern su struttura sintattica, non stringhe.
- **⚠️ Scope**: Solo Go e TypeScript. Python se già parsabile.
- **Verifica**: Bypass noto su regex → fallisce su AST → test PASS
- **Stima**: Medium (~4h)

### F5-05: Sandbox Verifier nel decision loop
- **File**: `internal/sandbox/verification.go`, `internal/orchestrator/chat.go`
- **Cosa**: Verifier PRIMA del tool dispatch (tool code validation). Se code malevolo → policy block + reject.
- **Verifica**: Tool code malevolo → Verifier block + policy deny → orchestration sceglie alternativa
- **Stima**: Short (~2h)

### F5-06: Policy compliance test suite
- **File**: `internal/policy/policy_test.go`
- **Cosa**: Input avversari contro ogni policy rule: tool dispatch fuori scope, rate limit bypass, prompt injection via argomenti, output bloccato
- **Verifica**: Ogni adversarial test → policy BLOCK
- **Stima**: Medium (~3h)

**Build check Fase 5**:
```bash
go build ./...
go test -v -race ./internal/policy/
go test -v -race ./internal/judge/
go test -v -race ./internal/scanner/
npx playwright test
```

---

## Fase 6: Multimodale (2 spike)
> *Opzionale. Non gate per ship.*

### F6-01: Image upload → OCR → context injection (SPIKE)
- **File**: `internal/input/` (new), frontend chat upload
- **Cosa**: SPIKE di 2 giorni per:
  - Endpoint upload immagine
  - OCR (tesseract CLI o API vision)
  - Structured context injection in Chat()
  - Frontend drag-and-drop
- **Output dello spike**: Prototipo funzionante O decisione di non procedere
- **Budget**: 2 giorni. Se non funziona entro 2gg → cancellare.
- **Stima**: **Large** (cap 2gg)

### F6-02: Computer-use sandbox → Playwright in container (SPIKE)
- **File**: `internal/browser/` (new)
- **Cosa**: SPIKE di 3 giorni per:
  - Container sandbox con Playwright
  - Tool: navigate, click, fill, screenshot
  - Rate limiting + timeout + confirmation gate
  - Output: screenshot + DOM snapshot
- **Output dello spike**: Prototipo funzionante O decisione di non procedere
- **⚠️ Security critical**: Container isolation, no network access esterno, max execution 30s
- **Budget**: 3 giorni. Se non funzionante → cancellare.
- **Stima**: **Large** (cap 3gg)

**Build check Fase 6**: Nessun build check obbligatorio (spike)

---

## Fase 7: Valutazione (5 task)
> *Prova che funziona. Ship con metriche.*

### F7-01: E2E benchmark framework
- **File**: `benchmark/suite/` (new directory)
- **Cosa**: Task suite con scoring automatico:
  - Single-agent: RAG query, tool execution, multi-step piano
  - Multi-agent: 3-agent coordinamento, conflitto, merge
  - Adversarial: policy bypass, prompt injection
  - Regression detection su CI
- **⚠️ Scope**: Benchmark framework, non adversarial research. Se adversarial diventa ricerca → stop.
- **Verifica**: `go test ./benchmark/suite/` produce report JSON
- **Stima**: **Large** (~3-5gg)

### F7-02: Decision trace audit tool
- **File**: `frontend/src/views/DecisionTraceView.tsx`, `internal/telemetry/replay.go`
- **Cosa**: Dashboard per replay decisioni: filtri per agente/tool/outcome, albero decisionale visuale, export JSON
- **Verifica**: Trace visibile in dashboard con Plan → Act → Observe → Reflect completo
- **Stima**: Medium (~4h)

### F7-03: Load test multi-agente
- **File**: `benchmark/load/`
- **Cosa**: 10+ sessioni concorrenti. Misurare: goroutine leak (count pre/post), latency P50/P95/P99, throughput tool calls/s, memory per sessione
- **Verifica**: Goroutine count ritorna a baseline, latency < target, zero leak
- **Stima**: Medium (~4h)

### F7-04: Frontend coverage target
- **Target**: Store 90%, Hooks 70%, Components 40%, Globale 60%
- **Cosa**: Completare test mancanti, rimuovere codice inaccessibile, refactor componenti non testabili
- **Verifica**: `npx vitest run --coverage` globale ≥ 60%
- **Stima**: Medium (~1gg)

### F7-05: Rollback + migration plan v2→v3
- **File**: `docs/plans/v2-to-v3-migration.md`
- **Cosa**: Documentare:
  - Schema DB changes (se memory store aggiunge tabelle)
  - API backward compatibility (MCP HTTP→STDIO, A2A routing)
  - Rollback procedure (quale commit, quale restore DB)
  - Feature flag strategy (disabilitare F3/F4 per tornare a v2 comportamento)
- **Verifica**: Documento approvato
- **Stima**: Short (~1h)

**Build check Fase 7**:
```bash
go build ./...
go test -v -race ./benchmark/...
npx vitest run --coverage   # ≥ 60%
npx playwright test         # 21/21
```

---

## Riepilogo v3

| Fase | Task | Quick | Short | Medium | Large |
|------|------|-------|-------|--------|-------|
| F1: MCP Completo | 8 | 0 | 3 | 5 | 0 |
| F2: Memoria | 4 | 0 | 2 | 2 | 0 |
| F3: Decision Loop | 4 | 1 | 1 | 2 | 0 |
| F4: Multi-Agente | 5 | 0 | 1 | 3 | 1 |
| F5: Sicurezza | 6 | 0 | 1 | 5 | 0 |
| F6: Multimodale (SPIKE) | 2 | 0 | 0 | 0 | 2 |
| F7: Valutazione | 5 | 1 | 0 | 3 | 1 |
| **Totale** | **34** | **2** | **8** | **20** | **4** |

### Cosa NON è in questo piano
- **Knowledge graph** → deferito a v4 (ontologia → KG è un progetto a sé)
- **NLP reale** → se serve, progetto separato (non aleph)
- **Frontend rewriting** → solo test, non riscrittura architetturale

### Vincoli Trasversali
1. **Niente nuove runtime deps senza rimozione documentata** di una dep inutilizzata
2. **Onestà**: Ogni AdmitFailure deve specificare il motivo — mai "errore generico"
3. **Judge advisory**: Mai blocco da Judge Model — policy engine ha l'unica authority di block
4. **Build check obbligatorio** a ogni fase boundary
5. **Ogni task produce test** — coverage gate cresce con ogni fase
6. **Nessuna fase dipende da una fase successiva** — ogni fase produce sistema shipabile
