# Coverage-80 Aggressive Implementation Plan

> **For agentic workers:** Each task is a self-contained unit. Execute sequentially. Verify after each task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Go ≥85% + Frontend ≥80% statement coverage with zero mock interfaces.

**Architecture:** Go tests use `newTestServer()` (DuckDB :memory: + httptest) from existing `internal/integration/`. Frontend tests use a shared `test-utils.tsx` mock module.

**Tech Stack:** Go `testing`+`testify`+`httptest`+DuckDB :memory: | Frontend `vitest`+`@testing-library/react`+`jsdom` (pool=forks)

---

## Phase A: Enablers

### Task A1: Create Frontend test-utils.tsx

**Files:**
- Create: `frontend/src/test-utils.tsx`

```tsx
import { vi } from 'vitest'
import { render, RenderOptions } from '@testing-library/react'
import React from 'react'

// ---- i18n mock ----
export function mockI18n(overrides?: Record<string, string>) {
  const defaults: Record<string, string> = {
    'agents.create': 'Nuovo Agente',
    'agents.edit': 'Modifica Agente',
    'agents.form.name': 'Es: Analista Finanze',
    'agents.form.model': 'Es: gpt-4o-mini o llama3.2',
    'agents.form.apiKey': 'Inserisci solo per sovrascrivere la chiave esistente (facoltativo)',
    'agents.form.baseUrl': 'Es: https://api.openai.com/v1',
    'agents.form.systemPrompt': "Definisci il ruolo dell'agente",
    'skills.create': 'Crea Skill',
    'skills.edit': 'Modifica Skill',
    'skills.form.name': 'Es: Analista Finanze',
    'skills.form.description': 'Descrivi la capacità di questa skill...',
    'skills.form.nameRequired': 'Il nome è obbligatorio',
    'confirmDialog.cancel': 'Annulla',
    'common.search': 'Cerca...',
    'common.save': 'Salva',
    'common.delete': 'Elimina',
    'common.loading': 'Caricamento...',
    ...overrides,
  }
  vi.mock('../../i18n', () => ({
    t: (key: string) => defaults[key] ?? key,
  }))
}

// ---- store mock ----
export function mockStore(state: Record<string, unknown> = {}) {
  const getState = vi.fn(() => state)
  vi.mock('../store/useStore', () => ({
    useStore: Object.assign(
      vi.fn((sel: (s: typeof state) => unknown) => sel(getState())),
      { subscribe: vi.fn(() => vi.fn()), getState }
    ),
  }))
}

// ---- features mock ----
export function mockFeatures(enabled: string[] = []) {
  vi.mock('../config/features', () => {
    const flags: Record<string, boolean> = {}
    for (const f of enabled) flags[f] = true
    return {
      isEnabled: vi.fn((key: string) => !!flags[key]),
      ...Object.fromEntries(Object.keys(flags).map(k => [k, flags[k]])),
    }
  })
}

// ---- nuqs mock ----
export function mockNuqs(value: string = '') {
  vi.mock('nuqs', () => ({
    useQueryState: vi.fn(() => [value, vi.fn()]),
  }))
}

// ---- jsdom polyfills ----
export function jsdomPolyfills() {
  Element.prototype.scrollTo = vi.fn() as unknown as typeof Element.prototype.scrollTo
  class MockIntersectionObserver {
    observe = vi.fn()
    unobserve = vi.fn()
    disconnect = vi.fn()
  }
  Object.defineProperty(window, 'IntersectionObserver', {
    writable: true, configurable: true, value: MockIntersectionObserver,
  })
  class MockResizeObserver {
    observe = vi.fn()
    unobserve = vi.fn()
    disconnect = vi.fn()
  }
  Object.defineProperty(window, 'ResizeObserver', {
    writable: true, configurable: true, value: MockResizeObserver,
  })
  window.HTMLElement.prototype.scrollIntoView = vi.fn()
}

// ---- render wrapper ----
export function renderWithProviders(ui: React.ReactElement, options?: RenderOptions) {
  return render(ui, { ...options })
}
```

---

## Phase B: Go Backend Tests

### Wave B1: Handler Tests via newTestServer() (125 funcs at 0%)

Use the existing `newTestServer()` from `internal/integration/integration_test.go`.

**Files:**
- Modify: `internal/integration/integration_test.go` (append test functions)

#### B1-01: Agent handler tests

- [ ] Test `ListAgents` — POST agent, GET list, verify agent present
- [ ] Test `CreateAgent` — POST agent, verify 201 + agent returned  
- [ ] Test `DeleteAgent` — POST agent, DELETE, verify 204, re-list verify gone
- [ ] Test `AgentLimit` — create N agents up to limit, verify limit enforced

#### B1-02: Skill handler tests

- [ ] Test `ListSkills` — POST skill, GET list, verify skill present
- [ ] Test `CreateSkill` — POST skill, verify 201 + skill returned
- [ ] Test `UpdateSkill` — POST skill, PATCH update, verify changed
- [ ] Test `DeleteSkill` — POST skill, DELETE, verify 204

#### B1-03: Tool handler tests

- [ ] Test `ListTools` — POST tool, GET list, verify tool present
- [ ] Test `CreateTool` — POST tool, verify 201 + tool returned
- [ ] Test `UpdateTool` — POST tool, PATCH update, verify changed
- [ ] Test `DeleteTool` — POST tool, DELETE, verify 204

#### B1-04: Auth + Session handler tests

- [ ] Test `ListApiKeys` — POST API key, list, verify
- [ ] Test `CreateApiKey` — POST API key, verify
- [ ] Test `DeleteApiKey` — POST, delete, verify 204
- [ ] Test `CreateSession` — POST session with valid API key, verify JWT cookie
- [ ] Test `ValidateSession` — GET validate with valid JWT
- [ ] Test `InvalidSession` — GET validate with invalid JWT, verify 401

#### B1-05: Project handler tests

- [ ] Test `ListProjects` — POST project, GET list, verify
- [ ] Test `CreateProject` — POST project, verify 201
- [ ] Test `DeleteProject` — POST, delete, verify 204

#### B1-06: Library handler tests

- [ ] Test `ListAssets` — GET /api/v1/library/assets
- [ ] Test `DeleteAsset` — POST asset, DELETE, verify 204

#### B1-07: Notification handler tests

- [ ] Test `ListChannels` — GET /api/v1/notifications/channels
- [ ] Test `CreateChannel` — POST channel, verify 201

#### B1-08: Ingestion handler tests

- [ ] Test `ListTasks` — GET /api/v1/ingestion/tasks
- [ ] Test `CreateTask` — POST task, verify 201

#### B1-09: NLP handler tests

- [ ] Test `AnalyzeSentiment` — POST /api/v1/nlp/analyze, verify response

#### B1-10: Query handler tests

- [ ] Test `Chat` — POST /api/v1/chat, verify response
- [ ] Test `GetDataStats` — GET /api/v1/data/stats

#### B1-11: SSE handler tests

- [ ] Test `StreamEvents` — GET /api/v1/sse/events, verify SSE headers
- [ ] Test `StreamUnauthenticated` — GET /api/v1/sse/events no auth, verify 401

#### B1-12: Registry handler tests

- [ ] Test `ListComponents` — POST component, GET list, verify
- [ ] Test `SearchComponents` — POST component, GET search, verify

#### B1-13: Codeflow handler tests

- [ ] Test `GetVisualization` — GET /api/v1/codeflow/viz
- [ ] Test `GetTracker` — GET /api/v1/codeflow/tracker

### Wave B2: Ingestion Engine Tests

**Files:**
- Create: `internal/ingestion/engine_supplement_test.go`

- [ ] Test `NewEngine(:memory:)` — create engine, verify tables created
- [ ] Test `RegisterViews` — register DuckDB views via engine
- [ ] Test `EnrichPredictiveMetadata` — run enrichment on :memory: tables
- [ ] Test `Close` — close engine, verify no panic on double close
- [ ] Test `UpdateProgress` — update task progress, verify stored
- [ ] Test `RunTask` — create and run a minimal task
- [ ] Test `ValidateSQLName` — valid/invalid SQL names
- [ ] Test `SanitizeIdentifier` — special chars, length limits
- [ ] Test `SanitizeFilePath` — path traversal, null bytes

### Wave B3: Sandbox Tests

**Files:**
- Create: `internal/sandbox/verification_supplement_test.go`

- [ ] Test `VerifyCode` — Go code syntax check (no runtime)
- [ ] Test `VerifyCode_invalid` — invalid Go code, verify error
- [ ] Test `CheckImports` — allowed imports pass, forbidden blocked
- [ ] Test `CheckHTTPCalls` — detect HTTP calls in code
- [ ] Test `CheckFileOps` — detect file operations in code

### Wave B4: Middleware Tests

**Files:**
- Create: `internal/middleware/streaming_supplement_test.go`

- [ ] Test `AuditMiddleware` — httptest request, verify audit header
- [ ] Test `CircuitBreakerMiddleware` — multiple fast failures, verify breaker opens
- [ ] Test `BulkheadMiddleware` — concurrent requests, verify queue behavior
- [ ] Test `RateLimitMiddleware` — rapid requests, verify 429 after limit
- [ ] Test `TimeoutMiddleware` — slow handler, verify 504 timeout
- [ ] Test `RetryMiddleware` — failing upstream, verify retries exhausted

### Wave B5: Service Tests

**Files:**
- Create: `internal/service/notification/notification_supplement_test.go`
- Create: `internal/service/tracker/tracker_supplement_test.go`

- [ ] Test `FormatChannel` — slack/email/webhook format generation
- [ ] Test `ValidateChannel` — valid/invalid channel configs
- [ ] Test `TrackerRecord` — record step, verify stored
- [ ] Test `TrackerGet` — record then retrieve step

### Wave B6: Routes Tests

**Files:**
- Create: `internal/routes/routes_supplement_test.go`

- [ ] Test `AllRoutesRegistered` — verify all expected routes return non-404
- [ ] Test `CORSMiddleware` — OPTIONS request, verify CORS headers
- [ ] Test `HealthEndpoint` — GET /healthz, verify 200
- [ ] Test `MetricsEndpoint` — GET /metrics, verify 200

### Wave B7: SSE Tests

**Files:**
- Create: `internal/api/sse/sse_supplement_test.go`

- [ ] Test `NewBroker` — create broker, verify channels
- [ ] Test `Subscribe` — subscribe client, verify message delivery
- [ ] Test `Unsubscribe` — subscribe then unsubscribe, verify cleanup
- [ ] Test `Broadcast` — broadcast message to multiple subscribers
- [ ] Test `Close` — close broker, verify channel cleanup

### Wave B8: Health Checker Tests

**Files:**
- Modify: `internal/health/checker_test.go` (append)

- [ ] Test `CheckToolHealth` — builtin checker against test tool
- [ ] Test `CheckToolHealth_failure` — failing tool, verify error recorded
- [ ] Test `GetHistory` — run check, verify history stored
- [ ] Test `ConsecutiveFailures` — multiple failures, verify count
- [ ] Test `UptimePercentage` — calculate uptime from history

### Wave B9: Migration Tests

**Files:**
- Create: `internal/migrate/migrate_supplement_test.go`

- [ ] Test `ExtractTableNames` — parse CREATE TABLE statements
- [ ] Test `ValidateMigrationOrder` — verify migration numbering sequence
- [ ] Test `MigrationConsistency` — duckdb + postgres have matching tables
- [ ] Test `NoDuplicateMigrations` — verify no repeated version numbers

### Wave B10: Ingestion Sources Tests

**Files:**
- Create: `internal/ingestion/sources/sources_supplement_test.go`

- [ ] Test `ParseCSV` — parse valid/invalid CSV content
- [ ] Test `ParseJSON` — parse valid/invalid JSON content
- [ ] Test `ValidateURL` — valid/invalid source URLs
- [ ] Test `DetectFormat` — auto-detect csv/json/xml from content

---

## Phase C: Frontend Tests

### Wave C1: Shared test-utils.tsx (already written as Task A1)

### Wave C2: Form Component Tests

**File pattern:** `frontend/src/components/__tests__/<ComponentName>.test.tsx`
All use `mockI18n()`, `mockStore()`, `mockFeatures()`, `jsdomPolyfills()` from test-utils.tsx.

#### C2-01: SkillForm expansion

- [ ] Add tests: empty-name validation, description editing, tool checkbox toggle, save with API call mock, edit mode button text

#### C2-02: AgentForm expansion  

- [ ] Add tests: model input, baseUrl input, system prompt textarea, provider select options, validation error display

#### C2-03: DataSourceForm expansion

- [ ] Add tests: file mode fields, API mode URL field, DB mode connection string, mode switching, format selector, JSON config

#### C2-04: AgentFormSlideOver

- [ ] Test: renders with slide overlay, agent schema validation, save triggers handler, cancel closes slideover

#### C2-05: DataSourceFormSlideOver

- [ ] Test: step 1 name wizard, step 2 type selection, step 3 config, back/next navigation, cancel resets

#### C2-06: SkillFormSlideOver

- [ ] Test: create/edit mode, tool checkbox list, save button, cancel slideover

#### C2-07: ToolFormSlideOver

- [ ] Test: create/edit mode, code textarea, name/description fields, save with validation

#### C2-08: ComponentFormSlideOver

- [ ] Test: 15 fields rendering, JSON validation, category/status selects, create/edit mode

### Wave C3: View Tests

#### C3-01: App.tsx

- [ ] Test: renders without crash, view switching via store, SlideOverContent lazy loads views

#### C3-02: LibraryView

- [ ] Test: list assets, search filter, empty state, load state

#### C3-03: OracleView

- [ ] Test: scenario list, create scenario, prediction display, empty state

#### C3-04: ScenarioComparisonView

- [ ] Test: comparison table renders, scenario selection, diff highlighting

### Wave C4: Hook Tests

#### C4-01: useInfiniteQueries

- [ ] Test: initial fetch, load more pages, error handling, hasMore flag

#### C4-02: NavigationStateSync

- [ ] Test: URL→store sync, store→URL sync, initial load

#### C4-03: useSSE (supplement)

- [ ] Test: reconnection backoff, event parsing, error recovery

### Wave C5: Slice Tests

#### C5-01: navigationSlice supplement

- [ ] Test: setActiveView, setSlideOverItem, toggleSlideOver

#### C5-02: workspaceSlice supplement

- [ ] Test: setProject, clearWorkspace, updateSettings

---

## Verification Gates

### V1: Go

```bash
go build ./...                                    # Must pass
go test -race -count=1 ./internal/...              # Must pass, 46/46
go vet ./...                                       # Must pass
go test -count=1 ./internal/integration/ -tags=integration  # Fix the 8 failing tests
go test -count=1 -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out | grep -v ".pb.go" | tail -1  # Must show >= 85%
```

### V2: Frontend

```bash
cd frontend && npx tsc --noEmit                   # Must pass
cd frontend && npx vite build                      # Must pass
cd frontend && npx vitest run --pool=forks          # Must pass, 0 failures
cd frontend && npx vitest run --pool=forks --coverage  # Must show >= 80%
```

### V3: Python

```bash
cd nlp && python3 -m pytest tests/ --cov=. --cov-fail-under=80  # Must pass
```
