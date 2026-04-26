# HANDOFF — Aleph-v2 Session 26 Aprile 2026

## Stato Generale

**Piano**: `plans/piano-finale-aleph-26-apr.md` (101 task, 16 wave)
**Progressione**: W0 ✅ → W0.5 ✅ → W1 ✅ → W2 ✅ → W3 parziale (danneggiata) → **BLOCCATO su recovery build**
**Build**: ❌ ROTTO — 9 errori in `routes.go` e `app.go` (metodi mancanti su ToolHandler/QueryHandler dopo corruzione W3-04)
**Git**: Lavoro W0-W2+parziale W3 presente in working tree (166 file modified/untracked), stash@{0} disponibile come backup

---

## Waves Completate

### W0 ✅ — Critical Fixes (SQLi, SSRF, Y.js, CORS, config)
- `duckdb.go`: scopeQuery parametrized, info_schema → `?` params in query.go/project.go/ingestion/engine.go
- `query.go`: Cleanup() double call rimosso
- Embedding hardcode: OllamaBaseURL via QueryHandler
- toolFailCount: da globale a ToolHealthMonitor field
- Config.Validate(): 8 field checks
- retry_provider.go: 3 retries + 5min TTL cache
- Registry test: SetMaxOpenConns(1) per DuckDB `:memory:`

### W0.5 ✅ — Build Fixes (context, schema, ToolRecord, auth)
- `duckdb.go`/`context.go`: ContextWithSchema/SchemaFromContext/scopeQuery/EnsureProjectSchema ricostruiti
- `auth_middleware.go`: pulito (ProjectIDFromContext duplicato rimosso)
- ToolRecord esteso: Category, Version, HealthStatus, SourceType, UpdateHealthStatus
- GetToolCode: signature → `(ctx context.Context, toolID string)`
- backupMu aggiunto a DuckDB struct

### W1 ✅ — TypeScript Build (43→0 errori) + Frontend
- tsconfig.json paths + vitest/globals
- uiSlice esteso: ToastMessage, confirmDialog, showGuide, toastMessages, addToast, removeToast
- copilotSlice esteso: splitView, bookmarkedIds, toggleBookmark, chatSearchQuery, onlyBookmarks
- types.ts: ToolIntel esteso con metadata/description/usage
- SlideOverContent + 'tool-intelligence' aggiunto a useStore.ts
- handleError esportato standalone da utils.ts
- 6 slice test corretti (StateCreator 3-arg), 3 hook test corretti (mock pattern)
- TerminalLine esportato, 116/116 vitest pass

### W2 ✅ — God Method Extraction + DuckDB Transactions
- **`internal/decision/`** (6 nuovi file): decision.go, engine.go, planner.go (buildToolDefinitions + validateToolName), observer.go, reflector.go, admitter.go
- **`query.go`**: QueryHandler con `decision.DecisionEngine` field, Chat() delegato, streaming/cronologia/ollama preservato
- **`tool_executor.go`**: ToolExecutor implementation (search_data/analyze_sentiment/get_trust_score)
- **`duckdb.go`**: TX type con BeginTX/BeginReadTX (semaphore + lock + SET schema), Commit/Rollback idempotent

### W3 Parziale ❌ — Danneggiata da agente ctx audit
- W3-01/02 ✅: `nlp/predict.py` e `nlp/convert_onnx.py` eliminati, `nlp/nlp_pb2_grpc.py` fixato
- W3-07 parziale: test fixes parziali (CreateTool 8-col INSERT, ListTools 8-col SELECT, fmt import)
- **W3-04 CORRUPTION**: Agente ctx audit ha corrotto `query.go`, `tool.go`, `nlp.go`, `registry_handler.go` — `connect.Request[T]` → `connect.Request[*T]`, metodi cancellati, duplicati aggiunti

---

## Stato Build Corrente (9 errori)

```
internal/routes/routes.go:58: QueryHandler non implementa GetChecksum
internal/routes/routes.go:76-81: ToolHandler.ServeHTTP/HandleVerify/HandleHealthHistory mancanti
internal/app/app.go:100: BrierObserver.Observe signature sbagliata (ctx in eccesso)
internal/app/app.go:151: QueryHandler.GetChecksum mancante
```

**Causa radice**: L'agente W3-04 (ctx audit) ha cancellato metodi da ToolHandler e cambiato la signature di BrierObserver.Observe aggiungendo context.Context. `git checkout HEAD` ha ripristinato i file ma ha perso le modifiche W2 su query.go.

---

## Problema Recovery

Il working tree corrente ha 166 file modificati/non-tracciati che contengono:
- ✅ Tutte le modifiche W0-W2 (buone)
- ✅ W3-01/02 (buone)
- ⚠️ W3-04 parziale (alcune corrupt changes potrebbero essere presenti in query.go, ast.go)

Lo stash@{0} è un backup del work precedente.

File specifici che necessitano attenzione:
1. **`internal/api/handler/query.go`** — Ha modifiche W2 (DecisionEngine) mescolate con possibile corruzione W3-04. Verificare che `connect.Request[T]` non sia `connect.Request[*T]`
2. **`internal/dsl/ast.go`** — Corrotto dall'agente, aggiunto duplicati ToolParamDef/HandlerDef/DepDef
3. **`internal/repository/metadata.go`** — W0.5 ToolRecord extension + ctx params
4. **`internal/api/handler/tool_executor.go`** — Creato in W2, verificare integrità

---

## Task Rimasti

### W1 Residuali (mai indirizzati)
- W1-04: NotificationService.Stop() — manca `close(s.jobs)` o quit channel, non wired in app.Close()
- W1-05: tool_suggest ctx fix — storePending passa request ctx a goroutine
- W1-06/W1-07/W1-08: Già risolti o non necessari

### W3 Rimasti
- W3-03: Honest responses (analyze_sentiment confidence)
- W3-04: Codeflow interfaces ctx audit — **RI-SCOPARE a singolo package per volta, MAI tutto internal/**
- W3-05/06: Hook + store slice tests frontend
- W3-07: Go test fixes (parziale, needs completion)

### W4-W10 + Wave Speciali
- **W4**: Decision Loop Planner/Observe/Reflect (12 task)
- **W5**: Decision Loop Reflect/Admit + Tool Intelligence (8 task)
- **W6**: Tool Interfaces + Residuali (6 task)
- **W7**: Frontend Coverage + Playwright (4 task)
- **W8**: Multi-Agent A2A (6 task)
- **W9**: Security F5 (6 task)
- **W10**: Evaluation F7 (7 task)
- **W-ERR, W-A11Y, W-PERF, W-DEPLOY, W-DOCS**: Wave speciali

---

## Procedura di Recovery (Passi Immediati)

1. **Verifica integrità file critici** nel working tree:
   ```bash
   # Controlla che query.go non abbia connect.Request[*T]
   grep -n 'connect.Request\[' internal/api/handler/query.go

   # Controlla che ast.go non abbia duplicati
   grep -n 'type ToolParamDef' internal/dsl/ast.go
   grep -n 'type HandlerDef' internal/dsl/ast.go

   # Controlla metadata.go per ToolRecord fields
   grep -n 'Category\|Version\|HealthStatus\|SourceType' internal/repository/metadata.go
   ```

2. **Se query.go è corrotto**: Ripristinare da git HEAD, poi ri-applicare manualmente le modifiche W2 (DecisionEngine + Chat refactor)

3. **Se ast.go ha duplicati**: Rimuovere i duplicati aggiunti dall'agente, tenere solo la definizione originale

4. **Fix build errors**: Ripristinare GetChecksum, ServeHTTP, HandleVerify, HandleHealthHistory su ToolHandler; correggere BrierObserver.Observe signature (rimuovere ctx)

5. **Verifica**: `go build ./...` → 0 errors

6. **Verifica**: `cd frontend && npx vite build` → 0 errors

7. **Procedere con W3 rimasti**

---

## Architettura Chiave (Per Referenza)

### Backend (Go)
- **Decision Engine**: `internal/decision/` — 6 file, interface Plan→Act→Observe→Reflect→Admit
- **QueryHandler**: `internal/api/handler/query.go` — delega a DecisionEngine, streaming ConnectRPC
- **ToolExecutor**: `internal/api/handler/tool_executor.go` — search_data/analyze_sentiment/get_trust_score
- **DuckDB TX**: `internal/storage/duckdb.go` — BeginTX/BeginReadTX con semaphore
- **Context**: `internal/storage/context.go` — ContextWithSchema/SchemaFromContext/scopeQuery
- **LLM**: `internal/llm/retry_provider.go` — 3 retries + 5min TTL cache
- **Config**: `internal/config/config.go` — Validate() con 8 checks

### Frontend (React)
- **React 18.3.1**, Zustand 4.5.2 (6 slices), lucide-react, Tailwind CSS
- **Niente React Router** — view switching via Zustand navigationSlice
- **App.tsx** ~2030 linee, SlideOverContent function
- 116/116 vitest pass (ultimo check)

### Tool Infrastructure
- `internal/tools/` — ToolRecord con Category/Version/HealthStatus/SourceType
- `internal/mcp/` — MCP discovery engine
- `internal/health/` — Health checker
- `internal/sandbox/` — Exec sandbox, verification, security

---

## Lezioni Apprese (CRITICHE per prox sessione)

1. **MAI dare scope "tutto internal/" a un agente** — L'agente W3-04 ha corrotto 4 file in 1h22min. Scope sempre a singolo package.
2. **Verificare sempre dopo ogni agente** — La corruzione W3-04 è passata inosservata fino ai build errori
3. **`connect.Request[T]` non `connect.Request[*T]`** — ConnectRPC usa generics value types, non pointer
4. **BrierObserver.Observe non prende context.Context** — La signature originale è `Observe(*AlephPrediction, float32)`
5. **Git stash + working tree** — Prima di operazioni rischiose, assicurarsi che lo stash sia pulito
6. **Frontend vitest 116/116** — Gli unici test che passano sono i 6 slice + 3 hook test. Gli altri test sono skippati o mancanti.
7. **Pre-existing errors** — `dsl/compiler_tool.go` e `adaptation/pipeline.go` hanno errori preesistenti non causati da noi

---

## Sessioni Agenti Precedenti

| Agente | Category | Task ID | Stato | Descrizione |
|--------|----------|---------|-------|-------------|
| Sisyphus-Junior | deep | bg_2dfafbef | ✅ | DecisionEngine + Chat refactor W2 |
| Sisyphus-Junior | deep | bg_949fedbc | ✅ | DuckDB transactions W2-03/04 |
| Sisyphus-Junior | deep | bg_e062b397 | ✅ | NLP dead code cleanup W3-01/02 |
| Sisyphus-Junior | deep | bg_2c8cbf10 | ✅ (CORROTTO) | W3-04 codeflow ctx audit |
| Sisyphus-Junior | deep | bg_4ff01d04 | ❌ timeout | W3-07 fix Go test failures |

---

## File Piano
- `plans/piano-finale-aleph-26-apr.md` — Piano master (101 task, 16 wave)

*Handoff creato il 26 Aprile 2026 — Sessione interrotta per recovery build*