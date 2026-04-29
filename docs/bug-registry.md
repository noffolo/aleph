# Bug Registry — Aleph-v2 Go Backend

> **Generated**: 2026-04-29 | **Codebase**: 25,210 lines across 151 non-test Go files | **Total functions**: 1,965 (253 exported, 160 unexported) | **Test functions**: 734 across 82 test files

---

## Section 1: Cyclomatic Complexity Leaderboard (Top 60)

> Threshold: ≥5 is "concerning", ≥10 is "refactor candidate", ≥15 is "critical"

| Rank | Complexity | Function | Package | File |
|------|-----------|----------|---------|------|
| 1 | **9** | `TestMCPConnectivity` (test) | mcp | connectivity_test.go:351 |
| 2 | **9** | `(*Watcher).Start` | service/watcher | watcher.go:25 |
| 3 | **9** | `(*ToolRegistry).RegisterAll` | tools | registry.go:77 |
| 4 | **9** | `(*DuckDB).cleanOldBackups` | storage | duckdb_backup.go:147 |
| 5 | **9** | `(*DuckDB).AutoBackup` | storage | duckdb_backup.go:86 |
| 6 | **9** | `looksLikeNonDecimalPart` | ssrf | validator.go:243 |
| 7 | **9** | `WriteEvent` | api/sse | sse.go:243 |
| 8 | **9** | `CheckGoFormat` | sandbox | validation.go:190 |
| 9 | **9** | `Retry` | middleware | retry.go:79 |
| 10 | **9** | `(*ToolHealthMonitor).checkOnce` | mcp | health.go:279 |
| 11 | **9** | `(*Engine).runPostgresLoad` | ingestion | engine.go:663 |
| 12 | **9** | `(*HealthChecker).checkMCPHealth` | health | checker.go:217 |
| 13 | **9** | `(*ToolSuggestHandler).HandleApprove` | api/handler | tool_suggest.go:199 |
| 14 | **9** | `(*ToolSuggestHandler).discoverMCPTool` | api/handler | tool_suggest.go:273 |
| 15 | **9** | `(*ToolExecuteHandler).HandleExecuteTool` | api/handler | tool_exec.go:162 |
| 16 | **9** | `(*QueryHandler).resolveAgent` | api/handler | query.go:488 |
| 17 | **9** | `(*ChatSession).executeAndStreamTool` | api/handler | chat_session.go:211 |
| 18 | **9** | `(*Trainer).step` | gnn | trainer.go:105 |
| 19 | **9** | `(*SentimentAnalysisFinTool).Execute` | tools/finance | sentiment_analysis_fin.go:57 |
| 20 | **9** | `ComputeDiversityScore` | ethics | bias.go:270 |
| 21 | **9** | `SecurityScan` | dsl | compiler_tool.go:647 |
| 22 | **9** | `RootCauseAnalysis` | diagnostic | patterns.go:111 |
| 23 | **9** | `(*GNNLinkPredictor).TrainFromGraph` | decision | gnn_adapter.go:64 |
| 24 | **9** | `(*AnalysisStage).Execute` | tools/adaptation | pipeline.go:212 |
| 25 | **8** | `(*DuckDB).Restore` | storage | duckdb_backup.go:48 |
| 26 | **8** | `StreamEvents` | api/sse | sse.go:298 |
| 27 | **8** | `(*ExecSandbox).RunSkill` | sandbox | exec_sandbox.go:129 |
| 28 | **8** | `(*MetadataRepository).ListAgentsCursor` | repository | metadata.go:268 |
| 29 | **8** | `ClassifyToolError` | repair | repair.go:168 |
| 30 | **8** | `addImportToBlock` | repair | repair.go:709 |
| 31 | **8** | `(*RepairEngine).AnalyseAndPlan` | repair | repair.go:316 |
| 32 | **8** | `(*ErrorHandlerInterceptor).apiErrorCodeToConnectCode` | middleware | error_handler.go:112 |
| 33 | **8** | `(*DiscoveryEngine).checkServerHealth` | mcp | discovery.go:284 |
| 34 | **8** | `validateCode` | ingestion | engine.go:834 |
| 35 | **8** | `(*Engine).registerViews` | ingestion | engine.go:328 |
| 36 | **8** | `(*ToolUsageTracker).GetTopUsers` | tools/humanecosystems | pattern_tracker.go:195 |
| 37 | **8** | `(*HealthChecker).checkAll` | health | checker.go:165 |
| 38 | **8** | `populateDefaultRegistry` | api/handler | tool_exec.go:56 |
| 39 | **8** | `ParsePagePagination` | api/handler | pagination.go:13 |
| 40 | **8** | `CheckDataBalance` | ethics | bias.go:89 |
| 41 | **8** | `Decrypt` | crypto | aesgcm.go:59 |
| 42 | **8** | `(*ExecutionTracker).RecordDependency` | tools/codeflow | tracker.go:61 |
| 43 | **8** | `(*AlephApp).watchSidecar` | app | app.go:435 |
| 44 | **8** | `extractPythonImports` | tools/adaptation | pipeline.go:287 |
| 45 | **7** | `stripComments` | sandbox | validation.go:234 |
| 46 | **7** | `(*MetadataRepository).ListAgents` | repository | metadata.go:240 |
| 47 | **7** | `(*RepairEngine).executeRegenerate` | repair | repair.go:574 |
| 48 | **7** | `(*AuditInterceptor).logAuditEvent` | middleware | audit.go:50 |
| 49 | **7** | `(*Embedder).Embed` | memory | embed.go:56 |
| 50 | **7** | `(*DiscoveryEngine).extractTools` | mcp | discovery.go:181 |
| 51 | **7** | `(*Engine).runDynamic` | ingestion | engine.go:774 |
| 52 | **7** | `timeOfDay` | tools/humanecosystems | pattern_tracker.go:52 |
| 53 | **7** | `(*PatternClassifier).queryPatterns` | tools/humanecosystems | he_pattern_classifier.go:46 |
| 54 | **7** | `NewChatSession` | api/handler | chat_session.go:49 |
| 55 | **7** | `(*ToolHandler).HandleHealthHistory` | api/handler | tool.go:122 |
| 56 | **7** | `(*QueryHandler).resolveProject` | api/handler | query.go:60 |
| 57 | **7** | `(*QueryHandler).GetDataLineage` | api/handler | query.go:355 |
| 58 | **7** | `(*QueryHandler).GetChecksum` | api/handler | query.go:389 |
| 59 | **7** | `(*QueryHandler).ConfirmAction` | api/handler | query.go:77 |
| 60 | **7** | `(*AgentHandler).ListAgents` | api/handler | agent.go:27 |

**Key Finding**: No function exceeds complexity 9. Maximum complexity in the codebase is moderate. The highest-risk areas cluster in: `ingestion/engine.go`, `api/handler/tool_suggest.go`, `api/handler/query.go`, and `tools/adaptation/pipeline.go`.

---

## Section 2: Exported API Surface

**Total**: 253 exported functions (excluding generated proto/v1connect code)

### By Package

| Package | Exported | Lines of Code | Test Files |
|---------|----------|--------------|------------|
| tools | 36 | 5,275 | 6 |
| api | 26 | 4,152 | 9 |
| middleware | 22 | 1,375 | 10 |
| sandbox | 19 | 1,366 | 5 |
| telemetry | 18 | 516 | 1 |
| errors | 16 | 205 | 1 |
| mcp | 15 | 1,478 | 6 |
| dsl | 12 | 1,113 | 3 |
| health | 7 | 429 | 2 |
| ethics | 7 | 412 | 1 |
| storage | 6 | 635 | 4 |
| gnn | 6 | 609 | 1 |
| decision | 6 | 1,071 | 1 |
| diagnostic | 5 | 277 | 1 |
| service | 4 | 419 | 2 |
| routes | 4 | 245 | 0 |
| repair | 4 | 1,007 | 2 |
| registry | 4 | 169 | 1 |
| genesis | 4 | 247 | 1 |
| cursor | 4 | 76 | 1 |
| workflow | 3 | 219 | 1 |
| ssrf | 3 | 258 | 1 |
| repository | 3 | 835 | 3 |
| migrate | 3 | 219 | 1 |
| memory | 3 | 169 | 0 |
| crypto | 3 | 110 | 1 |
| predict | 2 | 79 | 1 |
| llm | 2 | 468 | 0 |
| ingestion | 2 | 1,036 | 4 |
| auth | 2 | 89 | 1 |
| config | 1 | 113 | 1 |
| app | 1 | 473 | 0 |

### Notable Exported Constructors (high coupling risk)

| Function | Creates | Dependencies |
|----------|---------|-------------|
| `NewAlephApp` | *Root orchestrator* | Config, embed.FS, 15+ handlers |
| `NewQueryHandler` | Query API | DuckDB, projectsRoot, MetaRepo, NLPHandler, Registry |
| `NewChatSession` | Chat session | 7 params including tools, engine, providers |
| `NewEngine` (ingestion) | Ingestion engine | projectsRoot, MetaRepo, DuckDB, NLPAnalyzer |
| `NewDiscoveryEngine` | MCP discovery | Logger, MetaRepo, DiscoveryConfig |
| `NewShadowbroker` | OSINT broker | ShadowbrokerConfig (8 fields) |
| `NewSynthesisEngine` | Tool synthesis | 6 dependencies |

---

## Section 3: HTTP Handler Registration Map

Source: `/Users/ff3300/Desktop/aleph-v2/internal/routes/routes.go`

### Health & Probes (unauthenticated)

| Route | Method | Handler | Auth |
|-------|--------|---------|------|
| `/readyz` | GET | Inline (drain check) | No |
| `/livez` | GET | Inline (always 200) | No |
| `/api/v1/healthz` | GET | Inline (always 200) | No |
| `/metrics` | GET | `telemetry.MetricsHandler()` | No |

### Connect RPC Services (12 services)

| Route Prefix | Handler | Auth |
|-------------|---------|------|
| `/aleph.v1.QueryService/` | `QueryHandler` | Via interceptors |
| `/aleph.v1.ProjectService/` | `ProjectHandler` | Via interceptors |
| `/aleph.v1.AgentService/` | `AgentHandler` | Via interceptors |
| `/aleph.v1.SkillService/` | `SkillHandler` | Via interceptors |
| `/aleph.v1.ToolService/` | `ToolHandler` | Via interceptors |
| `/aleph.v1.LibraryService/` | `LibraryHandler` | Via interceptors |
| `/aleph.nlp.v1.NLPService/` | `NLPHandler` | Via interceptors |
| `/aleph.v1.NotificationService/` | `NotificationHandler` | Via interceptors |
| `/aleph.v1.AuthService/` | `AuthHandler` | Via interceptors |
| `/aleph.v1.IngestionService/` | `IngestionHandler` | Via interceptors |
| `/aleph.v1.SandboxService/` | `SandboxHandler` | Via interceptors |
| `/aleph.v1.RegistryService/` | `RegistryHandler` | Via interceptors |

### Session Management (unauthenticated — validates credentials)

| Route | Method | Handler | Auth |
|-------|--------|---------|------|
| `/api/v1/auth/session` | POST | `SessionHandler.HandleCreateSession` | No (credentials-in) |
| `/api/v1/auth/session` | DELETE | `SessionHandler.HandleDeleteSession` | No |

### Raw HTTP Routes (AuthMiddleware protected)

| Route | Handler | Purpose |
|-------|---------|---------|
| `/api/v1/tools/intelligence` | `ToolHandler.ServeHTTP` | Tool intelligence data |
| `/api/v1/tools/recommendations` | `ToolHandler.ServeHTTP` | Tool recommendations |
| `/api/v1/tools/health` | `ToolHandler.ServeHTTP` | Tool health status |
| `/api/v1/tools/verify` | `ToolHandler.HandleVerify` | Verify tool integrity |
| `/api/v1/tools/` | `ToolHandler.HandleHealthHistory` | Tool health history |
| `/api/v1/tools` | `ToolHandler.ServeHTTP` | Tool listing |
| `/api/v1/tools/suggest` | `SuggestPipeline` | Tool suggestion workflow |
| `/api/v1/tools/suggest/approve` | `SuggestPipeline` | Tool suggestion approval |
| `/api/v1/tools/categories` | `ToolExecHandler.HandleListCategories` | List tool categories |
| `/api/v1/tools/execute/{category}/{name}` | `ToolExecHandler.ServeHTTP` | Execute tool by name |
| `/api/v1/tools/call` | `ToolExecHandler.HandleCallTool` | Call tool (JSON body) |
| `/api/v1/tools/register` | `ToolExecHandler.HandleRegister` | Register custom tool |
| `/api/v1/codeflow/graph` | `CodeFlowHandler.HandleGetGraph` | Execution graph |
| `/api/v1/codeflow/metrics` | `CodeFlowHandler.HandleGetMetrics` | Execution metrics |
| `/api/v1/codeflow/executions` | `CodeFlowHandler.HandleListExecutions` | Execution history |
| `/api/v1/codeflow/engines` | `CodeFlowHandler.HandleListEngines` | Engine listing |
| `/api/v1/diagnostic/patterns` | Inline (DiagnosticMonitor) | Error patterns |

### Streaming & Static

| Route | Handler | Purpose |
|-------|---------|---------|
| `/api/v1/events` | `SSEHandler.Stream` | Server-Sent Events |
| `/swagger.json` | Inline (ServeFile) | API docs |
| `/` | Inline SPA fallback | Frontend serving |

**Total**: 4 health + 12 RPC + 2 session + 18 API + 3 static = **39 registered routes**

---

## Section 4: Top 30 Longest Functions (by line count)

| Rank | Lines | Function | File |
|------|-------|----------|------|
| 1 | **175** | `(*SynthesisEngine).GetUnifiedToolIntel` | tools/synthesis/synthesis.go |
| 2 | **172** | `(*AlephApp).Serve` | app/app.go |
| 3 | **167** | `Handle__NAME__` (generated) | dsl/compiler_tool.go |
| 4 | **150** | `(*RepairEngine).ExecutePlan` | repair/repair.go |
| 5 | **135** | `RegisterRoutes` | routes/routes.go |
| 6 | **129** | `(*Engine).runEmailFetch` | ingestion/engine.go |
| 7 | **125** | `(*Compiler).CompileObject` | dsl/compiler.go |
| 8 | **116** | `(*QueryHandler).GetDataStats` | api/handler/query.go |
| 9 | **114** | `(*ToolAnalyzer).DetectAnomalies` | tools/codeflow/analyzer.go |
| 10 | **105** | `(*QueryHandler).ExecuteQuery` | api/handler/query.go |
| 11 | **105** | `(*MCPHealthChecker).CheckServer` | mcp/health.go |
| 12 | **102** | `InitTelemetry` | telemetry/telemetry.go |
| 13 | **102** | `ComputeDemographicParity` | ethics/bias.go |
| 14 | **99** | `(*SynthesisEngine).GetCrossContextRecommendations` | tools/synthesis/synthesis.go |
| 15 | **96** | `(*OpenAIProvider).Complete` | llm/openai.go |
| 16 | **96** | `(*ToolSuggestHandler).HandleSuggest` | api/handler/tool_suggest.go |
| 17 | **94** | `RunPostgresMigrations` | migrate/migrate.go |
| 18 | **94** | `RunDuckDBMigrations` | migrate/migrate.go |
| 19 | **92** | `(*Engine).enrichPredictiveMetadata` | ingestion/engine.go |
| 20 | **91** | `(*Engine).runURLFetch` | ingestion/engine.go |
| 21 | **90** | `(*ToolHealthMonitor).checkOnce` | mcp/health.go |
| 22 | **88** | `(*TestingStage).Execute` | tools/adaptation/pipeline.go |
| 23 | **88** | `(*Engine).RunTask` | ingestion/engine.go |
| 24 | **87** | `(*AnthropicProvider).Complete` | llm/anthropic.go |
| 25 | **84** | `(*RegistrationStage).Execute` | tools/adaptation/pipeline.go |
| 26 | **83** | `(*ExecSandbox).ExecuteTool` | sandbox/exec_sandbox.go |
| 27 | **82** | `(*Engine).runPrecompiled` | ingestion/engine.go |
| 28 | **81** | `(*ToolIntel).ScanTool` | tools/osint/tool_intel.go |
| 29 | **80** | `(*AuditRepository).QueryAuditLog` | repository/audit.go |
| 30 | **80** | `(*DiscoveryEngine).Discover` | mcp/discovery.go |

### Longest Source Files (by total lines)

| Rank | Lines | File |
|------|-------|------|
| 1 | 998 | `internal/ingestion/engine.go` |
| 2 | 880 | `internal/repair/repair.go` |
| 3 | 783 | `internal/dsl/compiler_tool.go` |
| 4 | 707 | `internal/tools/adaptation/pipeline.go` |
| 5 | 637 | `internal/repository/metadata.go` |
| 6 | 587 | `internal/api/handler/query.go` |
| 7 | 473 | `internal/app/app.go` |
| 8 | 450 | `internal/decision/engine.go` |
| 9 | 421 | `internal/mcp/health.go` |
| 10 | 412 | `internal/ethics/bias.go` |

---

## Section 5: Bug Markers (TODOs, FIXMEs, HACKs, BUGs, XXXs)

**Total**: 4 markers found (all in non-generated code)

| Marker | File:Line | Context | Severity |
|--------|-----------|---------|----------|
| `TODO` | `repair/repair.go:814` | Auto-inserted comment: "consider adding caching/connection pooling for performance" | Info |
| `TODO` | `repair/repair.go:837` | Auto-prepended TODO comment block | Info |
| `TODO` | `repair/repair_test.go:396` | "// TODO: implement" | Low — test gap |
| `TODO` | `dsl/compiler_tool.go:158` | Generated code: "zero value, TODO: implement" | Low — template |

**Assessment**: The codebase has remarkably few explicit bug markers. No FIXME, HACK, BUG, or XXX markers found. The 4 TODOs are in auto-generated/template code or test stubs, not in production logic.

### Known Architectural Issues (from Phase 0 audit, not grep-based)

| ID | Issue | Severity | Location |
|----|-------|----------|----------|
| W1-04 | SQL injection via string concatenation (24 sites, 3 critical) | CRITICAL | query.go, context.go, duckdb.go |
| W1-05 | Blocked imports not enforced at execution time | HIGH | exec_sandbox.go, compiler_tool.go |
| W1-06 | API key in sessionStorage (not httpOnly cookie) | HIGH | client.ts → auth.go |
| W1-07 | SHA-256 key hashing without salt | HIGH | auth.go:49-57 |
| W1-09 | SSRF fail-open on 7+ HTTP clients | HIGH | agent.go, ollama.go, openai.go, etc. |
| W1-10 | Email credentials in plaintext /proc | HIGH | engine.go:871-957 |
| W1.5-01 | Provider:nil hardcoded → 0.5 confidence | MEDIUM | app.go:226 |
| W1.5-02 | Ollama missing from docker-compose | MEDIUM | docker-compose.yml |

---

## Section 6: Test Coverage Summary

### Overview

| Metric | Count |
|--------|-------|
| Source `.go` files (non-test, non-proto, non-mock) | 151 |
| Test `_test.go` files | 82 |
| Test functions (`func Test*`) | 734 |
| Source-to-test file ratio | 1.85:1 (151 source / 82 test) |
| Packages with tests | 28 of 32 |

### Packages with ZERO Tests

| Package | Source Files | Lines of Code | Risk |
|---------|-------------|---------------|------|
| `app` | 1 | 473 | **HIGH** — Root orchestrator, `Serve()`, lifecycle |
| `llm` | 5 | 468 | **HIGH** — All LLM providers (Ollama, OpenAI, Anthropic) |
| `memory` | 2 | 169 | **MEDIUM** — Embedding + chunking |
| `routes` | 1 | 245 | **LOW** — Route wiring (covered by integration tests) |

### Top Test Files by Function Count

| Rank | Tests | File |
|------|-------|------|
| 1 | 41 | `mcp/discovery_test.go` |
| 2 | 32 | `repair/repair_test.go` |
| 3 | 31 | `ethics/bias_test.go` |
| 4 | 27 | `repository/metadata_test.go` |
| 5 | 25 | `mcp/jsonrpc_test.go` |
| 6 | 22 | `api/sse/sse_test.go` |
| 7 | 18 | `telemetry/telemetry_test.go` |
| 8 | 17 | `health/history_test.go` |
| 9 | 16 | `tools/synthesis/synthesis_test.go` |
| 10 | 16 | `mcp/health_test.go` |
| 11 | 16 | `tools/codeflow/codeflow_test.go` |
| 12 | 15 | `errors/errors_test.go` |
| 13 | 14 | `sandbox/validation_test.go` |
| 14 | 14 | `sandbox/security_test.go` |
| 15 | 14 | `mcp/handler_test.go` |

### Test Coverage Gaps (low test:source ratio)

| Package | Source Files | Test Files | Gap Assessment |
|---------|-------------|------------|----------------|
| `tools` | 5,275 LOC | 6 test files | Most subpackages tested |
| `api/handler` | 25 source | 9 test | Multiple handlers untested |
| `dsl` | 4 source | 3 test | Good coverage |
| `ingestion` | 2 source, 1036 LOC | 4 test | Adequate |
| `decision` | 8 source, 1071 LOC | 1 test | **GAP** — complex logic, 1 test file |
| `repair` | 2 source, 1007 LOC | 2 test | Adequate |
| `sandbox` | 7 source | 5 test | Adequate |
| `tools/adaptation` | pipeline.go 707 LOC | 1 test | **GAP** — pipeline.go has 88-line functions |

---

## Section 7: Recommendations for High-Priority Refactors

### Priority 1: Security (W1 items — block production)

1. **query.go SQL injection** — 3 critical string-concatenation sites (lines 362-363, 393-394). Migrate to parameterized queries or `resolveTableName` with double-quoted identifiers. Complexity 7-8 in handler functions, but the risk is injection, not complexity.

2. **engine.go email credentials** — 129 lines in `runEmailFetch`, plaintext password in `/proc`. Replace with Go IMAP library or env-var credentials with zero temp file. This is the longest unguarded function at 129 lines.

3. **SSRF unguarded HTTP clients** — 7+ HTTP clients (agent.go, ollama.go, openai.go, anthropic.go, embed.go, notification.go, compiler_tool.go) lack `ValidateSSRF`. Unify on `mcp.ValidateSSRF` which includes DNS resolution.

4. **SHA-256 → argon2id** — `auth.go:49-57` and `middleware/auth.go:26-35`. No salt in current hash. Add `hash_algorithm` column, create `hasher.go` with argon2id + legacy SHA-256 fallback.

### Priority 2: Complexity & Length Reduction

5. **ingestion/engine.go** (998 lines, 5 functions in top-30 longest) — Extract `runEmailFetch` (129L), `runURLFetch` (91L), `runPrecompiled` (82L), `enrichPredictiveMetadata` (92L), `runPostgresLoad` (complexity 9) into separate files or a subpackage. Target: engine.go < 400 lines.

6. **tools/synthesis/synthesis.go** (354 lines, 2 functions 175L and 99L) — `GetUnifiedToolIntel` at 175 lines is the longest function in the codebase. Break into `gatherSecurityIntel`, `gatherHealthIntel`, `gatherUsageIntel`, `mergeIntel`.

7. **tools/adaptation/pipeline.go** (707 lines, 3 stages 88L/84L/38L) — Extract each pipeline stage into its own file under `adaptation/stages/`. This also has zero direct test coverage for the pipeline itself.

8. **app/app.go** (473 lines, `Serve` at 172L) — Extract route setup, dependency wiring, and graceful shutdown into separate functions/files. Zero test coverage is the highest risk here.

9. **api/handler/query.go** (587 lines, 3 functions in top-30) — `GetDataStats` (116L), `ExecuteQuery` (105L), `resolveAgent` (complexity 9). Decompose `GetDataStats` into stat collectors.

10. **mcp/health.go** (421 lines, 2 functions 105L/90L) — Extract `CheckServer` (105L) into `health/server_checker.go` and `checkOnce` (90L) into `health/monitor.go`.

### Priority 3: Test Coverage Gaps

11. **app package** — Zero tests for root orchestrator. Minimally: test `NewAlephApp` construction, `Serve` startup, `Close` cleanup. Mock all 15+ handler dependencies.

12. **llm package** — Zero tests for all 3 providers (Ollama, OpenAI, Anthropic). Each `Complete()` function is 87-96 lines. Create `provider_test.go` with table-driven tests using `httptest.Server` mocks.

13. **decision package** — 8 source files (1071 LOC) with only 1 test file (13 tests). The `gnn_adapter.go` `TrainFromGraph` has complexity 9. Add integration tests for `Plan` → `Act` → `Reflect` → `Admit` loop.

14. **memory package** — Zero tests for embedding + chunking. `Embed` has complexity 7. Test vector dimensions, error paths, and HTTP timeout behavior.

### Priority 4: Structural

15. **NewAlephApp coupling** — Takes `embed.FS` and creates 15+ handlers. Consider dependency injuction via a `HandlerFactory` interface or wire methods to reduce constructor arity.

16. **Connect RPC interceptor gaps** — All 12 Connect RPC services share the same `cfg.Interceptors`. Verify auth/authz interceptors are applied (currently only session endpoint is explicitly unauthenticated; RPC services may lack API key validation).

17. **OSINT shadowbroker** — `NewCircuitBreaker` and `NewRateLimiter` are redefined in `osint/shadowbroker.go` (shadow the middleware package). Consider importing from middleware or using distinct names.

---

## Appendix: Methodology

- **Cyclomatic complexity**: `gocyclo -over 1 ./internal/` filtered for non-test, non-proto, non-mock files
- **Function lengths**: awk-based line counting from `^func ` to next function or EOF
- **Exported API**: grep `^func [A-Z]` excluding `_test.go`, `vendor/`, `mocks/`, `proto/`, `v1connect/`
- **HTTP handlers**: Analysis of `routes.go` `RegisterRoutes` function
- **Bug markers**: grep for `(TODO|FIXME|HACK|BUG|XXX):` in all `.go` files
- **Test coverage**: `find` for `_test.go` files + grep for `^func Test` counts
- **Codebase totals**: `find + wc -l` across 151 source files = 25,210 lines
