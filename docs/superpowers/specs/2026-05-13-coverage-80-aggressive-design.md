# Coverage-80 — Zero-Mock Strategy

**Date:** 13 May 2026
**Status:** Approved

## Goal

Bring Go backend to ≥85% and Frontend to ≥80% statement coverage using zero mock interfaces — all tests exercise real code paths against DuckDB `:memory:` databases and real HTTP servers.

## Architecture

```
setupTestApp(t) — already exists in internal/integration/
├── DuckDB :memory: → storage.DuckDB (analytic queries)
├── DuckDB :memory: → repository.MetadataRepository (tools, skills, agents, API keys)
├── DuckDB :memory: → registry.DuckDBRegistry (component catalog)
├── httptest.Server → all routes + middleware chain
├── real auth → argon2id hashing + JWT
└── t.TempDir() → filesystem operations

Frontend test-utils.tsx (new shared module)
├── mockI18n() → map-based Italian translations
├── mockStore(state) → Zustand with subscribe/getState
├── mockFeatures(enabled) → feature flag overrides
├── mockNuqs(value) → URL query state
└── jsdomPolyfills() → scrollTo, IntersectionObserver, ResizeObserver
```

## Go Strategy

### Phase A: Enable all packages with DuckDB :memory:

| Package | Enabler |
|---------|---------|
| `handler/*` (125 funcs at 0%) | Already wired in `newTestServer()` — just send httptest requests |
| `ingestion/engine.go` | Already has `NewEngine(db, meta, ingestService)` — just needs DuckDB tables |
| `sandbox/verification.go` | Pure AST analysis — zero Docker needed |
| `sandbox/validation.go` | Import/path checking — zero Docker |
| `middleware/*` (streaming wrappers) | httptest with real handler chain |
| `service/notification` | Format checks — HTTP calls are side effects |
| `health/checker.go` | Individual check functions testable without MCP |
| `routes/routes.go` | Route registration verification via httptest |

### Phase B: Write tests per package

| Target | Current | Goal | Method |
|--------|---------|------|--------|
| `handler/*` | 50.4% | 85% | httptest requests via `newTestServer()` |
| `ingestion/engine.go` | 31.5% | 70% | Run tasks in :memory: |
| `sandbox/*` | 58.4% | 80% | AST analysis + validation |
| `middleware/*` | 66.3% | 80% | httptest with real chains |
| `service/*` | 36.7% | 75% | Format + structure tests |
| `health/*` | 55.5% | 75% | Individual checker functions |
| `routes/*` | 18.7% | 70% | httptest route wiring |
| `api/sse/*` | 68.2% | 75% | Broker struct management |
| `migrate/*` | 36.7% | 50% | SQL string comparison |
| `ingestion/sources/*` | 8.7% | 50% | Parse logic sans network |

### Accepted caps (truly infrastructure-bound):
- `app/Serve(port)` — needs real server startup
- `service/watcher` — needs Docker
- `sandbox/exec_sandbox` — needs Docker

## Frontend Strategy

### Phase A: Create shared test utilities
File: `frontend/src/test-utils.tsx` — exports all mock helpers.

### Phase B: Write tests per component group

| Group | Files | Tests |
|-------|-------|-------|
| Form components | AgentForm, DataSourceForm, SkillForm, AgentFormSlideOver, DataSourceFormSlideOver, SkillFormSlideOver, ToolFormSlideOver, ComponentFormSlideOver | ~60 |
| Views | App.tsx, LibraryView, OracleView, ScenarioComparisonView | ~30 |
| Hooks | useInfiniteQueries, NavigationStateSync, useCursorPagination | ~15 |
| Slices | navigationSlice, workspaceSlice supplements | ~10 |

## Targets

| | Current | Target |
|---|---------|--------|
| Go | ~60% | **≥85%** |
| Frontend | 54.91% | **≥80%** |
| Python | 83.6% | 83.6% ✅ |

## Constraints

- Zero mock interfaces — all tests use real implementations
- DuckDB `:memory:` for all database dependencies
- httptest.Server for all HTTP handler tests
- No new dependencies added to any package

## Risks

| Risk | Mitigation |
|------|-----------|
| Integration tests OOM (single-file run) | Run integration tests one file at a time |
| Frontend form test i18n complexity | Shared mock module eliminates duplicated mock code |
| DuckDB dialect differences from PostgreSQL | All tests use DuckDB-compatible SQL |
