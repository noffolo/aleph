# Ontology Value Flow — Aleph-v2

**Generated**: 2026-04-29 | **Source**: Phase 0 Production Gate

## 1. Overview

In Aleph-v2, the **ontology** is a domain-specific schema definition written in the `.aleph` DSL format, stored at `<project>/ontologies/core.aleph`. It is the single source of truth that bridges raw data (parquet/CSV/JSON) and the AI layer. The ontology defines:

- **Objects** — business entities with properties, ID fields, filters, and aggregates
- **Relations** — foreign-key-style joins between objects
- **Datasets** — versioned data source bindings
- **Actions** — parameterized operations bound to objects
- **Tools** — custom tool definitions with inputs, outputs, and handlers

The DSL is parsed by `internal/dsl/parser.go` (using `participle`) into a `Program` AST (`internal/dsl/ast.go`), then consumed by 5 distinct pipeline entry points.

**Ontology lifecycle:**
1. **Emerge** — auto-generated from DuckDB `information_schema` columns (`ProjectHandler.EmergeOntology`)
2. **Edit** — user edits raw DSL text in `OntologyView.tsx`
3. **Save** — persisted to disk atomically with `.bak` backup (`ProjectHandler.SaveOntology`)
4. **Parse** — compiled on-demand with program cache (`QueryHandler.resolveProject`)
5. **Invalidate** — cache TTL of 30 minutes in `programCache`

## 2. Entry Point 1: Query/NLP Enrichment

### Input
The NLP sidecar (`nlp/main.py`) receives `StreamPredictionsRequest` with:
- `context_id` — maps to a DuckDB table name (project-scoped)
- `ontology_query` — a string (currently `"*"` from frontend)

### Processing
- The Python service loads time-series data from DuckDB via `load_history_from_duckdb(context_id, duckdb_path)`
- If `ontology_query` is present, it computes a **query_signal** feature: `int(hashlib.sha256(ontology_query.encode()).hexdigest()[:8], 16) % 100 / 100.0`
- This signal is blended into the `PredictiveEnsemble.predict_probs(features)` call alongside `drift_detected` and `market_prob`
- The ensemble produces probability estimates (`STABLE_TREND`, `ACTION_REQUIRED`) calibrated by market data

### Output
- `StreamPredictionsResponse` stream with `entity_id`, `probability`, `predicted_state`, `explanation`, `is_synthetic`
- Frontend `OracleView.tsx` renders these as prediction cards with confidence levels

### Risk
- **Ontology is effectively unused**: the `ontology_query` is just hashed into a float — no structural understanding of object types, properties, or relations. If ontology is missing, `query_signal` is simply omitted from features with no fallback.

## 3. Entry Point 2: DSL Compiler (Data Access Layer)

### Input
`QueryHandler.resolveProject(projectID)` reads and parses `core.aleph`:
```go
ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
content, err := os.ReadFile(ontPath)
prog, err = dsl.Parse(string(content))
```
The parsed `Program` AST is cached per-project in `programCache` (LRU, max 64 entries, 30min TTL).

### Processing
Three compilation paths:

**a) `CompileObject(objName)`** — transforms an ObjectDefinition into SQL:
- Maps `from dataset <name>` to `read_parquet('<dataRoot>/<name>/latest/*.parquet')`
- Maps `property X type Y from Z` to `"objName"."Z" AS "X"`
- Applies `map X to Y` as CASE WHEN expressions
- Adds `predict` placeholder columns (`0.0 AS "prop_probability"`)
- Adds `factor` placeholder columns (`0.0 AS "_factor_name"`)
- Resolves `relation` definitions into `LEFT JOIN` on parquet files
- Applies `filter` as WHERE clauses with sanitization
- Applies `aggregate` as GROUP BY + SUM/COUNT/AVG/MIN/MAX

**b) `CompileActions()`** — transforms ActionDefinitions into LLM tool schemas
**c) View Registration** — during ingestion (`Engine.registerViews`): creates `CREATE OR REPLACE VIEW "<projectID>_<objectName>" AS <compiledSQL>` in DuckDB

### Risk
- **If ontology is missing/wrong**: `resolveProject` returns `CodeNotFound` error — the entire query pipeline fails.
- **No fallback to raw table names** if ontology is absent.
- **Cache invalidation**: If data files change but the program cache hasn't expired, views may reference stale schemas.

## 4. Entry Point 3: Decision Engine (Plan→Act→Observe→Reflect→Admit)

### Input
`DecisionEngine.Plan()` receives `ontContent []byte` (currently **nil** in `ChatSession.Run()`)

### Processing
- `extractObjectReferences(msg)` — returns `nil` (stub)
- `buildToolDefinitions()` — produces `search_data` tool (object_name, limit)
- `inferToolsFromMessage()` — keyword-based heuristic
- `PlanWithProvider()` — LLM call with tools + system prompt

### Output
- `PlanResult` with `Intent`, `PlannedStep[]`, confidence
- `search_data` tool dispatches to `ExecuteQuery()` → `CompileObject()`

### Risk
- **No ontology awareness in tool planning**: `extractObjectReferences` returns nil. LLM infers object names from system prompt alone.
- **`ontContent` is nil in ChatSession**: `PlanWithProvider` never receives ontology content.

## 5. Entry Point 4: Genesis/Memory (Auto-Improvement Cycle)

### Input
`GenesisEngine.Suggest()` receives `ProjectID`, `AgentID`, chat history, tool usage (not wired)

### Processing
- `Suggester.Analyze()` — **stub**, returns `[]Suggestion{}`
- `MemoryStore` — **stub**, only provides `Close()`

### Risk
- **Complete dead code**: No auto-improvement cycle exists.
- No path for: learning object query patterns, suggesting ontology definitions from usage, embedding ontology for semantic search.

## 6. Entry Point 5: Frontend (Ontology Management & Visualization)

### Input
- `ontologyRaw` — string in Zustand `workspaceSlice` (default `""`)
- `availableObjects` — string array (default `[]`)

### Processing
- **OntologyView.tsx** — DSL editor + visual glossary (regex-based: `line.match(/^object\s+(\w+)/)`)
- **Emerge**: scans DuckDB `information_schema` → auto-generates DSL
- **Save**: writes atomically with backup

### Risk
- **No client-side validation**: User can save invalid DSL. Parse errors surface at query time.
- **No preview**: No way to test if ontology compiles before saving.
- **No diff**: No comparison between emerged and current version.
- **Glossary parser is brittle**: Regex expects exact `^object \w+` format.

## 7. Dependency Chain

```
core.aleph (disk)
  │
  ├── resolveProject() ──────── cache (LRU 64, TTL 30min)
  │     ├── ExecuteQuery() ──── CompileObject() ──── SQL → DuckDB → frontend
  │     ├── GetDataStats() ──── CompileObject() ──── column statistics SQL
  │     └── Chat() ────── ReadFile(core.aleph) ──── System Prompt + full ontology text
  │                        └── ChatSession.Run() ── LLM call → search_data → ExecuteQuery
  │
  ├── registerViews() ──────── CompileObject() per object ──── CREATE OR REPLACE VIEW
  └── enrichPredictiveMetadata() ─ Parse ontology ──── find primaryKey from object ID
        └── NLP sidecar.AnalyzeSentiment() on text columns
```

## 8. Risks and Failure Modes

| # | Risk | Severity | Impact | Status |
|---|------|----------|--------|--------|
| R1 | Missing ontology file — `resolveProject()` fails with `CodeNotFound` | CRITICAL | All query/chat/data endpoints return 404 | Placeholder created on project creation. No runtime fallback. |
| R2 | Parse error in ontology — `dsl.Parse()` returns error | HIGH | Cache bypassed, view registration silently skips | No user-facing validation feedback |
| R3 | Stale program cache — data changes but cache hasn't expired | MEDIUM | Queries return outdated schema, views reference missing columns | 30min TTL, no invalidation on ingestion |
| R4 | `extractObjectReferences()` returns nil | MEDIUM | LLM infers object names from raw system prompt | Degraded mode: 0.5 confidence |
| R5 | NLP sidecar uses ontology only as hash | LOW | No semantic understanding from ontology structure | Stub implementation |
| R6 | Genesis/Memory are dead code | MEDIUM | No ontology evolution based on usage patterns | Fully stubbed |
| R7 | No client-side ontology validation | MEDIUM | Parse errors surface at query time, not save time | No parse-then-validate in SaveOntology |
| R8 | Emerge ontology is structural, not semantic | LOW | Auto-generated DSL has no domain knowledge | Expected behavior |
| R9 | Full ontology text in LLM prompt | LOW | Large ontologies consume context tokens | Could be optimized |
| R10 | View registration order matters | LOW | Self-healing at read time | Logged as warning |
