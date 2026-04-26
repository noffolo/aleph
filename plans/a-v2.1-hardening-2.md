# aleph-v2.1: Secondo Hardening

> **Base**: 20-agent audit findings (non toccati da hardening v1) + 5 explore agent su aree residue
> **Review**: Momus ✅ | Aleph ✅ urgenti | Metis ✅ "Plan A"
> **Stima**: ~2 settimane
> **Principio**: Onestà prima di tutto. Nessuna architettura LAM — solo ciò che è non-negoziabile.

---

## Wave 1: Integrità Strutturale (9 task)

> *Stop the bleeding. Fix goroutine leak, God Object, test infra.*

### W1-01: NotificationService — Stop() + close channel
- **Fonte**: G2 — 3 worker() goroutines immortali in `internal/service/notification/notification.go`
- **Cosa**: Aggiungere `Stop()` method che chiude channel e fa terminare worker. Chiamare da shutdown sequence di `app.go`
- **Verifica**: `go test -race ./internal/service/notification/` + goroutine count pre/post
- **Stima**: Short (~30min)

### W1-02: tool_suggest.go — time.Sleep(5min) → select ctx.Done + ticker
- **Fonte**: G4 — cleanup goroutine non interruptibile
- **File**: `internal/api/handler/tool_suggest.go`
- **Cosa**: Sostituire `time.Sleep(5 * time.Minute)` con `select { case <-ticker.C: cleanup(); case <-ctx.Done(): return }`
- **Verifica**: `go test -race ./internal/api/handler/`
- **Stima**: Quick (~20min)

### W1-03: ingestion.RunTask + enrichPredictiveMetadata — ctx cancellation
- **Fonte**: G9, G10 — fire-and-forget senza cancellazione su Close()
- **File**: `internal/ingestion/ingestion.go`, `internal/ingestion/engine.go`
- **Cosa**: Passare context con cancellazione, interrompere su Close()
- **Verifica**: `go test -race ./internal/ingestion/`
- **Stima**: Short (~30min)

### W1-04: app.go God Object decomposition
- **Fonte**: Architecture debt — 498 linee, 20 campi, 84-line constructor, 248-line Serve()
- **File**: `internal/app/app.go`
- **Cosa**: Estrarre in 4 package:
  - `internal/wire/` — dependency injection (`NewServer`, `NewApp`)
  - `internal/server/` — Serve() lifecycle (start, shutdown, health)
  - `internal/routes/` — route registration (raw HTTP + Connect RPC + middleware)
  - `internal/health/` — startup checks (DuckDB, Postgres, NLP)
- **⚠️ AI Failure Point**: Dipendenze implicite tra campi app. Usare un solo file per package, testare ogni estrazione con `go build`.
- **Verifica**: `go build ./...` + `go test ./...` invariato
- **Stima**: Medium (4-6 ore)

### W1-05: ToolCodeWriter/ToolCodeReader — add context.Context
- **Fonte**: Architecture debt — interfacce senza ctx = timeout/annullamento impossibile
- **File**: `internal/api/handler/tool.go`, `internal/tools/codeflow/`
- **Cosa**: Aggiungere `ctx context.Context` a tutti i metodi delle interfacce. Aggiornare implementazioni e callers.
- **Verifica**: `go build ./...`
- **Stima**: Quick (~20min)

### W1-06: DuckDB backup lock — non bloccare tutte le read
- **Fonte**: Architecture debt — `AutoBackup()` tiene exclusive Lock bloccando tutte le RLock
- **File**: `internal/storage/duckdb_backup.go`, `internal/tools/humanecosystems/duckdb_layer.go`
- **Cosa**: Usare snapshot-read o lock separato per backup. Documentare gerarchia lock.
- **Verifica**: `go test -race ./internal/storage/` + `go test -race ./internal/tools/humanecosystems/`
- **Stima**: Short (~1h)

### W1-07: Frontend test infra — vitest.config.ts + CI gate 15%
- **Fonte**: Frontend coverage gap — Vitest installato MAI configurato
- **File**: `frontend/vitest.config.ts` (new)
- **Cosa**: Config con coverage reporter, jsdom environment, test patterns, test timeout 10s
- **Verifica**: `npx vitest run --coverage` exit 0 con report
- **Stima**: Quick (~20min)

### W1-08: Store slice unit tests — 8 Zustand slices
- **Fonte**: Frontend coverage gap — 0 store test (pure functions, highest ROI)
- **File**: `frontend/src/store/__tests__/*.test.ts`
- **Store reali**: `authSlice`, `copilotSlice`, `healthSlice`, `navigationSlice`, `uiSlice`, `workspaceSlice`, `types.ts`, `useStore.ts`
- **Cosa**: Testare ogni slice: reducer, selectors, edge cases (stato nullo, errori, loading)
- **Verifica**: `npx vitest run --coverage` coverage store ≥ 90%
- **Stima**: Medium (3-4 ore)

### W1-09: SlideOver triage — registry map + stub cleanup
- **Fonte**: Architecture debt — 18 switch cases in `SlideOverContent`, 1 stub
- **File**: `frontend/src/components/SlideOverPanel.tsx` (non SlideOverContent.tsx)
- **Cosa**: Convertire switch case in `Record<string, Component>` registry. Rimuovere stub default (null). Documentare 17 casi reali.
- **Verifica**: `npx tsc --noEmit` + `npx vite build`
- **Stima**: Quick (~20min)

### Build check W1
```bash
go build ./...          # ✅
npx tsc --noEmit        # ✅
npx vitest run          # ✅
npx vite build          # ✅
```

---

## Wave 2: Onestà e Test (8 task)

> *NLP finto muore. Frontend ottiene copertura. Decisioni tracciabili.*

### W2-01: DELETE NLP dead code
- **Fonte**: NLP — CalibrationWrapper dead, predict_probs fittizio, sentiment fake
- **File**: `nlp/ensemble.py`, `nlp/calibration.py`, `nlp/predict.py`, `nlp/convert_onnx.py`, `nlp/*.onnx`
- **⚠️ Decisione**: DELETE — tutti e 3 i reviewer concordano
- **Cosa eliminare**:
  - `ensemble.py`: CalibrationWrapper, righe 260-287 (predict_probs fittizio), `load_validation_data` no-op
  - `calibration.py`: intero file (Platt Scaling, Isotonic — mai chiamati)
  - `predict.py`: `train_link_prediction`, `LinkPredictor` — mai importati
  - `convert_onnx.py`: mai importato
  - `*.onnx`: modello binario
  - `requirements.txt`: `xgboost`, `sentencepiece`, `torch`, `torch-geometric` (mai importati)
- **Cosa tenere**: gRPC server (serve le request), `nlp_pb2.py`, `nlp_pb2_grpc.py` (protocollo necessario), `requirements.txt` base
- **Verifica**: `cd nlp && python -c "from ensemble import PredictiveEnsemble"` fallisce → OK. `go build ./...` ✅
- **Stima**: Short (~1h)

### W2-02: Fix gRPC proto — check in .proto + fix import path
- **Fonte**: NLP — gRPC import path `aleph.nlp.v1.nlp_pb2` non esiste
- **File**: `nlp/nlp_pb2_grpc.py` (import), `nlp/*.proto` (new)
- **Cosa**: Check in .proto file, generare pb2 con path corretto, allineare import Go
- **Verifica**: `python -c "from nlp_pb2_grpc import *"` exit 0
- **Stima**: Quick (~30min)

### W2-03: Honest responses — mai fabricare score
- **Fonte**: NLP — predict_probs restituisce `sigmoid(feature_mean)` come "prediction"
- **File**: `internal/api/handler/query.go` (Chat), `internal/nlp_adapter/adapter.go`
- **Cosa**: Ogni path che produce score fittizi deve restituire errore esplicito "NLP prediction confidence unavailable — model not trained" invece di score fabbricato. **W0.5 epistemic integrity principle**.
- **Verifica**: Nessun path che restituisce 0.5 come score "predittivo"
- **Stima**: Short (~30min)

### W2-04: Decision Loop — Plan stage in Chat()
- **Fonte**: LAM gap — Chat() ha 5-iteration loop ma nessun planning
- **File**: `internal/api/handler/query.go` (line 705, Chat)
- **Cosa**: Aggiungere fase Plan prima di Act:
  1. **Plan**: LLM call per decomporre intent in tool sequence
  2. **Act**: Eseguire tool plan
  3. **Observe**: Valutare output tool vs intent
  4. **Reflect**: Ricalibrare o consolidare
  5. **AdmitFailure**: Se dopo 2 tentativi il tool fallisce, rispondere "Non posso completare questa richiesta perché [motivo]" — mai inventare successo
- **Verifica**: `go test -v ./internal/api/handler/query_test.go` + decision trace spans
- **Stima**: Medium (4-6 ore)

### W2-05: Decision trace spans — OpenTelemetry per tool selection
- **Fonte**: LAM gap — nessuna traccia di decisione (perché tool X e non Y?)
- **File**: `internal/api/handler/query.go`, `internal/telemetry/`
- **Cosa**: Aggiungere span OTEL per ogni decisione: tool considerati + score + rationale + alternativi scartati
- **Verifica**: Span visibili in console exporter
- **Stima**: Short (~1h)

### W2-06: Hook unit tests — 15 hook file
- **Fonte**: Frontend coverage gap — 15 hook file, 0 test
- **File**: `frontend/src/hooks/__tests__/*.test.ts`
- **Cosa**: Testare useAppActions, useProject, useSlideOver, useSSE, useChat, useToolActions con mock store
- **Verifica**: `npx vitest run --coverage` hooks ≥ 70%
- **Stima**: Medium (3-4 ore)

### W2-07: Hook integration tests — hook + API client
- **Fonte**: Frontend coverage gap — hook mai testati con API
- **File**: `frontend/src/hooks/__tests__/*.integration.test.ts`
- **Cosa**: Test hook + API adapter + error boundary + loading state. Mockare fetch/WebSocket.
- **Verifica**: `npx vitest run` integration suite
- **Stima**: Medium (3-4 ore)

### W2-08: Component tests — 10 componenti critici
- **Fonte**: Frontend coverage gap — 40+ component, 0 test
- **File**: `frontend/src/components/__tests__/*.test.tsx`
- **Cosa**: AgentForm, SkillForm, ToolForm, ChatPanel, SlideOverPanel, DataHealthView, SettingsView, StatusBar, CopilotView, ToolIntelligenceView
- **Verifica**: `npx vitest run --coverage` components ≥ 40%
- **Stima**: Medium (4-6 ore)

### Build check W2
```bash
go build ./...          # ✅
npx tsc --noEmit        # ✅
npx vitest run --coverage  # coverage ≥ 50% globale
npx vite build          # ✅
npx playwright test     # ✅ (21/21)
cd nlp && python -c "from nlp_pb2_grpc import *"  # ✅
```

---

## Riepilogo v2.1

| Wave | Task | Quick | Short | Medium |
|------|------|-------|-------|--------|
| W1: Integrità Strutturale | 9 | 2 | 3 | 2 |
| W2: Onestà e Test | 8 | 1 | 3 | 4 |
| **Totale** | **17** | **3** | **6** | **6** |

### Cosa NON è in questo piano (deferito a v3)
- Multi-agente (A2A, orchestrator)
- Long-term memory (vector store)
- MCP STDIO compliance
- Policy engine / Judge model
- Multimodal / Computer-use
- Benchmark framework
- Knowledge graph migration

### Vincoli
1. **Onestà prima di tutto**: Mai fabricare output. Errori reali, non score fittizi.
2. **Niente nuove dipendenze senza giustificazione documentata**.
3. **Build check obbligatorio** a ogni wave boundary.
4. **NLP DELETE è irreversibile** — se serve NLP reale, sarà un progetto separato.
