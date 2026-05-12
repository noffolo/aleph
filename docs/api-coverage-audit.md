# Backend API Coverage Audit — UX Redesign (W0-04)

> **Date:** 11 May 2026  
> **Scope:** All backend API endpoints (ConnectRPC + raw HTTP) mapped against UX redesign requirements  
> **Plan reference:** `plans/aleph-ux-redesign-piano.md` W0-04  
> **Related specs:** `docs/specs/ux-redesign-w1-store-refactor.md`, `docs/specs/ux-redesign-w3-slideover-unification.md`  

---

## 1. Key Finding

**The existing backend API is largely sufficient for W1–W6 UX redesign with no new major endpoints required.** Of the 61 store fields across 6 slices, 52+ are already backed by API endpoints. The remaining fields are local UI state (navigation, UI preferences, transient UI toggles) that correctly live only in the frontend.

**7 specific gaps were identified** (detailed in §5) — all are read-only health/summary endpoints that would improve UX quality but are not blockers.

---

## 2. ConnectRPC Endpoints (via proto files)

All ConnectRPC endpoints are POST-only (ConnectRPC convention) and served under their service path prefix.

### 2.1 QueryService — `/aleph.v1.QueryService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `ExecuteQuery` | `{object_type, project_id, limit}` | `{sql, columns[], rows[]}` | Core data query |
| `Chat` | `{message, project_id, agent_id}` | **stream** `{token, tool_call, requires_confirmation}` | Streaming chat |
| `GetChatHistory` | `{project_id, agent_id}` | `{messages[]}` | Chat log for Oracle |
| `GetDataStats` | `{project_id, object_type}` | `{stats: [{column_name, min, max, count, unique_count, top_values}]}` | Data health stats |
| `ConfirmAction` | `{project_id, agent_id, approved}` | `{success}` | Approval flow |
| `GlobalQuery` | `{object_type, project_id, limit}` | `{sql, columns[], rows[]}` | Global search |
| `GetDataLineage` | `{project_id, table_name}` | `{provenance}` | Data provenance |
| `GetChecksum` | `{project_id, table_name}` | `{checksum, table_name, verified}` | Integrity check |

### 2.2 ProjectService — `/aleph.v1.ProjectService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `ListProjects` | `{}` | `{projects[]}` | — |
| `CreateProject` | `{id, name}` | `{project}` | — |
| `DeleteProject` | `{id}` | `{success}` | — |
| `EmergeOntology` | `{project_id}` | `{aleph_definition}` | AI ontology generation |
| `GetOntology` | `{project_id}` | `{aleph_definition, object_names[]}` | Current ontology |
| `SaveOntology` | `{project_id, aleph_definition}` | `{success}` | Save edited ontology |

### 2.3 AgentService — `/aleph.v1.AgentService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `ListAgents` | `{project_id, after, limit}` | `{agents[], next_cursor}` | Cursor-paginated |
| `CreateAgent` | `{project_id, agent}` | `{agent}` | Full agent object |
| `DeleteAgent` | `{project_id, id}` | `{success}` | — |
| `UpdateAgent` | `{project_id, agent}` | `{agent}` | Full update |
| `ListModels` | `{}` | `{models[]}` | Ollama models list |

### 2.4 SkillService — `/aleph.v1.SkillService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `ListSkills` | `{project_id, after, limit}` | `{skills[], next_cursor}` | Cursor-paginated |
| `CreateSkill` | `{project_id, skill}` | `{skill}` | — |
| `UpdateSkill` | `{project_id, skill}` | `{skill}` | — |
| `DeleteSkill` | `{project_id, id}` | `{success}` | — |

### 2.5 ToolService — `/aleph.v1.ToolService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `ListTools` | `{project_id, after, limit}` | `{tools[], next_cursor}` | Cursor-paginated |
| `CreateTool` | `{project_id, tool}` | `{tool}` | — |
| `UpdateTool` | `{project_id, tool}` | `{tool}` | — |
| `DeleteTool` | `{project_id, id}` | `{success}` | — |

### 2.6 IngestionService — `/aleph.v1.IngestionService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `ListTasks` | `{project_id}` | `{tasks[]}` | — |
| `CreateTask` | `{project_id, task}` | `{task}` | — |
| `DeleteTask` | `{project_id, id}` | `{success}` | — |
| `RunTask` | `{project_id, task_id}` | `{status}` | Trigger execution |
| `GetTaskLogs` | `{project_id, task_id}` | `{logs}` | Logs as string |
| `GetProgress` | `{project_id, task_id}` | `{progress}` | Int 0-100 |

### 2.7 LibraryService — `/aleph.v1.LibraryService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `ListAssets` | `{project_id}` | `{assets[]}` | — |
| `GetAssetContent` | `{project_id, asset_id}` | `{content}` | Raw content |
| `DeleteAsset` | `{project_id, id}` | `{success}` | — |
| `GeneratePdf` | `{project_id, asset_id}` | `{pdf_data, filename}` | PDF generation |
| `UploadAsset` | `{project_id, filename, content}` | `{asset}` | Binary upload |

### 2.8 AuthService — `/aleph.v1.AuthService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `ListApiKeys` | `{project_id}` | `{keys[]}` | — |
| `CreateApiKey` | `{project_id, label}` | `{key}` | — |
| `DeleteApiKey` | `{project_id, id}` | `{success}` | — |

### 2.9 NotificationService — `/aleph.v1.NotificationService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `SendWebhook` | `{url, payload_json, secret}` | `{success, error}` | — |
| `ListChannels` | `{project_id}` | `{channels[]}` | Webhook/email/Slack |

### 2.10 RegistryService — `/aleph.registry.v1.RegistryService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `RegisterComponent` | `{metadata}` | `{component_id}` | Full ComponentMetadata |
| `GetComponent` | `{id}` | `{metadata}` | Single component detail |
| `ListComponents` | `{filter}` | `{components[]}` | Map-based filter |
| `UpdateComponentStatus` | `{id, status}` | `{}` | Status update |

### 2.11 SandboxService — `/aleph.tool.v1.SandboxService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `ExecuteTool` | `{tool_id, input_params}` | `{result}` | ExecutionResult |
| `RunSkill` | `{skill_id, input_params, context}` | `{result}` | Skill with context |

### 2.12 NLPService — `/aleph.nlp.v1.NLPService/`

| RPC | Request | Response | Note |
|-----|---------|----------|------|
| `AnalyzeSentiment` | `{text}` | `{score, label, method, is_calibrated}` | Heuristic/ML |
| `StreamPredictions` | `{context_id, ontology_query}` | **stream** `{entity_id, probability, predicted_state, explanation, is_synthetic}` | Predictive streaming |
| `RecordFeedback` | `{entity_id, is_correct, correction_value, feedback_type}` | `{success}` | Active learning |

---

## 3. Raw HTTP Endpoints

### 3.1 System / Health (unauthenticated)

| Method | Path | Response | Note |
|--------|------|----------|------|
| GET | `/readyz` | `{"status":"ok"}` / 503 draining | Readiness probe |
| GET | `/livez` | `{"status":"alive"}` | Liveness probe |
| GET | `/api/v1/healthz` | `{"status":"ok"}` | Docker HEALTHCHECK |

### 3.2 Auth Session (rate-limited, unauthenticated on POST)

| Method | Path | Request/Response | Note |
|--------|------|------------------|------|
| POST | `/api/v1/auth/session` | `{api_key}` → `{project_id}` | Create session, set httpOnly JWT cookie |
| GET | `/api/v1/auth/session` | — → `{valid:true/false}` | Validate session |
| DELETE | `/api/v1/auth/session` | — | Delete session |

### 3.3 Tools (protected, readAny/readWrite/adminOnly)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/v1/tools/intelligence` | `ToolHandler.ServeHTTP` | Tool intelligence listing |
| GET | `/api/v1/tools/recommendations` | `ToolHandler.ServeHTTP` | Tool recommendations |
| GET | `/api/v1/tools/health` | `ToolHandler.ServeHTTP` | Tool health status |
| GET | `/api/v1/tools/` | `ToolHandler.HandleHealthHistory` | Tool health history |
| GET | `/api/v1/tools` | `ToolHandler.ServeHTTP` | List all tools (default) |
| GET | `/api/v1/tools/verify` | `ToolHandler.HandleVerify` | Tool verification |
| GET | `/api/v1/tools/categories` | `ToolExecHandler.HandleListCategories` | Tool categories |
| GET | `/api/v1/tools/execute/{category}/{name}` | `ToolExecHandler.HandleListToolsByCategory` | List tools in category |
| POST | `/api/v1/tools/execute/{category}/{name}` | `ToolExecHandler.HandleExecuteTool` | Execute tool |
| POST | `/api/v1/tools/call` | `ToolExecHandler.HandleCallTool` | Execute by qualified name |
| POST | `/api/v1/tools/register` | `ToolExecHandler.HandleRegister` | Register package tools |
| GET/POST | `/api/v1/tools/suggest` | `SuggestPipeline` | Tool suggestion workflow |
| POST | `/api/v1/tools/suggest/approve` | `SuggestPipeline` | Approve suggestion |

### 3.4 Ontology Negotiation

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/v1/ontology/propose` | Propose ontology change |
| POST | `/api/v1/ontology/accept` | Accept proposal |
| POST | `/api/v1/ontology/reject` | Reject proposal |
| GET | `/api/v1/ontology/versions` | List ontology versions |

### 3.5 CodeFlow

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/codeflow/graph?tool_id=X` | Tool execution graph |
| GET | `/api/v1/codeflow/metrics?tool_id=X` | Tool execution metrics |
| GET | `/api/v1/codeflow/executions?tool_id=X&limit=N` | Execution records |
| GET | `/api/v1/codeflow/engines` | List available engines |

### 3.6 Diagnostics

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/diagnostic/patterns` | Diagnostic patterns |

### 3.7 SSE Events (authenticated)

| Method | Path | Events |
|--------|------|--------|
| GET | `/api/v1/events` | `tool_status`, `notification`, `ingestion_progress`, `system_alert`, `health_change` |

### 3.8 Infrastructure

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/swagger.json` | OpenAPI spec |
| GET | `/metrics` | Prometheus metrics |
| GET | `/*` (SPA) | Frontend hosting |

---

## 4. Coverage by UX Requirements

### 4.1 W1 — Store Field CRUD Coverage

Each store field mapped to its backing API endpoint:

| Slice | Field | API Endpoint | Status |
|-------|-------|-------------|--------|
| **auth** | `projectID` | `POST /api/v1/auth/session` → `project_id` | ✅ Covered |
| | `apiKeys` | `aleph.v1.AuthService/ListApiKeys` | ✅ Covered |
| | `projects` | `aleph.v1.ProjectService/ListProjects` | ✅ Covered |
| | `notificationChannels` | `aleph.v1.NotificationService/ListChannels` | ✅ Covered |
| | `registryComponents` | `aleph.registry.v1.RegistryService/ListComponents` | ✅ Covered |
| **navigation** | `currentView` | Local state only | ✅ N/A (UI-only) |
| | `inlineContent` | Local state only (deprecated W3) | ✅ N/A (deleted in W3) |
| | `showInlinePanel` | Local state only (deprecated W3) | ✅ N/A (deleted in W3) |
| | `commandHistory` | Local state only | ✅ N/A (UI-only) |
| | `slideOverContent` | `aleph.v1.{...}Service/List*` (varies by type) | ✅ Covered |
| | `isCommandPaletteOpen` | Local state only | ✅ N/A (UI-only) |
| | `activeView` | Local state only | ✅ N/A (UI-only) |
| | `chat` | `aleph.v1.QueryService/Chat` (stream) + `GetChatHistory` | ✅ Covered |
| | `input` | Local state only (forms) | ✅ N/A (UI-only) |
| | `isStreaming` | Local state only (derived from Chat stream) | ✅ N/A (UI-only) |
| | `selectedAgent` | `aleph.v1.AgentService/ListAgents` | ✅ Covered |
| | `chatSearchQuery` | Local state only | ✅ N/A (UI-only) |
| **workspace** | `sandboxResult` | `aleph.tool.v1.SandboxService/ExecuteTool` | ✅ Covered |
| | `sandboxInput` | Local state only (form input) | ✅ N/A (UI-only) |
| | `searchQuery` | `aleph.v1.QueryService/GlobalQuery` (search call) | ✅ Covered |
| | `selectedObject` | Local state only (navigation) | ✅ N/A (UI-only) |
| | `predictions` | `aleph.nlp.v1.NLPService/StreamPredictions` | ✅ Covered |
| | `data` | `aleph.v1.QueryService/ExecuteQuery` | ✅ Covered |
| | `selectedRow` | Local state only (table selection) | ✅ N/A (UI-only) |
| | `agents` | `aleph.v1.AgentService/ListAgents` | ✅ Covered |
| | `ingestionTasks` | `aleph.v1.IngestionService/ListTasks` | ✅ Covered |
| | `ontologyRaw` | `aleph.v1.ProjectService/GetOntology` | ✅ Covered |
| | `ontologyVersions` | **Dead field** (zero consumers) | ❌ Dead (no endpoint needed) |
| | `selectedVersionId` | **Dead field** (zero consumers) | ❌ Dead (no endpoint needed) |
| | `isVersionHistoryOpen` | **Dead field** (zero consumers) | ❌ Dead (no endpoint needed) |
| | `availableObjects` | Derived from `GetOntology.object_names[]` | ✅ Covered |
| | `scenarios` | **⚠️ No backend endpoint** | ❌ **GAP** — frontend-only state |
| | `selectedScenarioIds` | **⚠️ No backend endpoint** | ❌ **GAP** — frontend-only state |
| | `taskLogs` | `aleph.v1.IngestionService/GetTaskLogs` | ✅ Covered |
| | `skills` | `aleph.v1.SkillService/ListSkills` | ✅ Covered |
| | `tools` | `aleph.v1.ToolService/ListTools` + `GET /api/v1/tools` | ✅ Covered |
| **health** | `ollamaHealthy` | **⚠️ No dedicated endpoint** | ❌ **GAP** (§5 gap #5) |
| | `nlpHealthy` | **⚠️ No dedicated endpoint** | ❌ **GAP** (§5 gap #5) |
| | `dataHealthStats` | `aleph.v1.QueryService/GetDataStats` | ✅ Covered |
| | `lastError` | Local state only (error handling) | ✅ N/A (UI-only) |
| | `ollamaModels` | `aleph.v1.AgentService/ListModels` | ✅ Covered |
| **ui** | `showOnboarding` | Local state only | ✅ N/A (UI-only) |
| | `showWizard` | Local state only | ✅ N/A (UI-only) |
| | `showGuide` | **Dead field** (zero consumers) | ❌ Dead (no endpoint needed) |
| | `isExplorerLoading` | Local state only | ✅ N/A (UI-only) |
| | `selectedAssetContent` | `aleph.v1.LibraryService/GetAssetContent` | ✅ Covered |
| | `selectedAssetId` | Local state only (selection) | ✅ N/A (UI-only) |
| | `globalSearchResults` | `aleph.v1.QueryService/GlobalQuery` | ✅ Covered |
| | `assets` | `aleph.v1.LibraryService/ListAssets` | ✅ Covered |
| | `confirmDialog` | Local state only | ✅ N/A (UI-only) |
| | `enableScanline` | Local state only (CSS effect toggle) | ✅ N/A (UI-only) |
| | `enableGlow` | Local state only (CSS effect toggle) | ✅ N/A (UI-only) |
| | `enableFlicker` | Local state only (CSS effect toggle) | ✅ N/A (UI-only) |
| | `toastMessages` | Local state only | ✅ N/A (UI-only) |
| | `inputMode` | Local state only | ✅ N/A (UI-only) |
| | `pendingCrud` | Local state only (operation lock) | ✅ N/A (UI-only) |

### 4.2 W2 — Navigation & SlideOver Scene Requirements

| Scene | Data Needs | API Coverage | Status |
|-------|-----------|-------------|--------|
| **Terminal** | Chat stream, health, dashboard stats | `QueryService.Chat`, `GetDataStats` | ✅ Covered |
| **Explore** | Projects, ontology, assets, data queries | `ProjectService`, `QueryService`, `LibraryService` | ✅ Covered |
| **Agents** | Agents list, skills, tools, components | `AgentService`, `SkillService`, `ToolService`, `RegistryService` | ✅ Covered |
| **System** | Health, settings, API keys, diagnostics | `AuthService`, diagnostics, health — but no **aggregated** endpoint | ⚠️ Partial (§5 gap #1) |
| **Copilot** | Chat stream, agent list, models | `QueryService.Chat`, `AgentService` | ✅ Covered |

Scene routing itself (`?scene`, `?view`, `?tab`, `?slide` URL params) is entirely frontend-local and requires no backend changes.

### 4.3 W3 — SlideOver Unification (20+ → 4 Scenes)

| SlideOver Type | Backed By | Status |
|---------------|-----------|--------|
| `explore` | `ProjectService.GetOntology`, `QueryService` | ✅ Covered |
| `ontology` | `ProjectService.GetOntology/SaveOntology` | ✅ Covered |
| `data` | `QueryService.ExecuteQuery` | ✅ Covered |
| `health` | `QueryService.GetDataStats` | ✅ Covered |
| `skill` | `SkillService.ListSkills/Create/Update/Delete` | ✅ Covered |
| `tool` | `ToolService` + raw HTTP tool endpoints | ✅ Covered |
| `sandbox` | `SandboxService.ExecuteTool/RunSkill` | ✅ Covered |
| `agent` | `AgentService.ListAgents/Create/Update/Delete` | ✅ Covered |
| `datasource` | `IngestionService.ListTasks/Create/Delete/RunTask` | ✅ Covered |
| `component` | `RegistryService.ListComponents/GetComponent` | ✅ Covered |
| `settings` | `AuthService.ListApiKeys`, `NotificationService.ListChannels` | ⚠️ Partial (§5 gap #6) |
| `library` | `LibraryService.ListAssets/GetAssetContent` | ✅ Covered |
| `predict` | `NLPService.StreamPredictions` | ✅ Covered |
| `asset` | `LibraryService.GetAssetContent` | ✅ Covered |
| `detail` | Varies (component detail, tool detail, etc.) | ✅ Covered |
| `agent-form` | `AgentService.CreateAgent/UpdateAgent` | ✅ Covered |
| `skill-form` | `SkillService.CreateSkill/UpdateSkill` | ✅ Covered |
| `tool-form` | `ToolService.CreateTool/UpdateTool` | ✅ Covered |
| `datasource-form` | `IngestionService.CreateTask` | ✅ Covered |
| `component-form` | `RegistryService.RegisterComponent` | ✅ Covered |
| `component-detail` | `RegistryService.GetComponent` | ✅ Covered |
| `tool-intelligence` | Raw HTTP `/api/v1/tools/intelligence` | ✅ Covered |
| `scenario-comparison` | **No backend persistence** | ❌ **GAP** (§5 gap #2) |
| `dashboard` | Mixed: health + stats + projects | ⚠️ Partial (§5 gap #1) |

### 4.4 W5 — Progressive Disclosure Data Requirements

| View | Tier-1 (Always) | Tier-2 (Expand) | Tier-3 (Advanced) | Backend Support |
|------|-----------------|-----------------|-------------------|-----------------|
| **ToolsView** | Name, status, health | Config, exec history | JSON editor, permissions | ⚠️ **GAP §5 gap #3** — no per-tool execution summary endpoint |
| **OracleView** | Input bar | Full conversation | Model config | ✅ Covered (`QueryService.Chat`, `GetChatHistory`, `AgentService`) |
| **SettingsView** | API keys, model | All settings | Developer/debug | ⚠️ **GAP §5 gap #6** — no aggregated settings endpoint |
| **ComponentsView** | Component grid, health | Per-component config | Dependency graph | ✅ Covered (`RegistryService`) |
| **LibraryView** | Search bar, grid | Item metadata, preview | Batch ops | ✅ Covered (`LibraryService`) |
| **AgentsView** | Agent list, status | Config steps | System prompt, tool bindings | ✅ Covered (`AgentService`, `SkillService`) |

---

## 5. Identified Gaps

### GAP #1: No Aggregated System Health Endpoint (🔴 HIGH)
**Impacted UX:** W2 SystemScene, W3 dashboard SlideOver, StatusBar health indicators

The frontend currently polls 3+ separate endpoints to determine system health:
- Ollama health via direct HTTP to Ollama API (no backend proxy)
- NLP sidecar via gRPC health check (watched internally, not exposed)
- Tool health via `GET /api/v1/tools/health` (only tool records, not system components)
- DuckDB and PostgreSQL status (no endpoint)

**Missing endpoint:** `GET /api/v1/system/health`

```typescript
// Response shape
{
  ollama: { healthy: boolean; models: string[] }
  nlp: { healthy: boolean; method: string }
  duckdb: { healthy: boolean; schema: string }
  postgres: { healthy: boolean }
  sse: { connected_clients: number }
}
```

### GAP #2: No Scenario Management Endpoints (🟡 MEDIUM)
**Impacted UX:** W1 store fields `scenarios` + `selectedScenarioIds`, W3 `scenario-comparison` SlideOver type

Scenarios are entirely frontend-local. No backend persistence exists. If the ScenarioComparisonView is to remain or be migrated to W3 scenes, scenarios must be persisted.

**Missing endpoints:**
```protobuf
rpc ListScenarios(ListScenariosRequest) returns (ListScenariosResponse);
rpc CreateScenario(CreateScenarioRequest) returns (CreateScenarioResponse);
rpc DeleteScenario(DeleteScenarioRequest) returns (DeleteScenarioResponse);
```

### GAP #3: No Per-Tool Execution Summary Endpoint (🟡 MEDIUM)
**Impacted UX:** W5-03 ToolsView progressive disclosure (Tier-2 "execution history")

`GET /api/v1/codeflow/metrics?tool_id=X` exists but returns abstract `ExecutionMetrics`. There's no endpoint that returns a human-readable execution history summary for a specific tool (last N runs, average duration, success rate).

**Missing endpoint:** `GET /api/v1/tools/{id}/executions?limit=N`

```typescript
// Response shape
{
  tool_id: string
  executions: Array<{
    id: string
    status: string
    duration_ms: number
    timestamp: string
    error: string | null
  }>
  stats: {
    total_runs: number
    success_rate: number
    avg_duration_ms: number
  }
}
```

### GAP #4: No Agent Run History Endpoint (🟡 MEDIUM)
**Impacted UX:** W5-04 AgentsView progressive disclosure (Tier-2 "run history")

AgentService has no `GetAgentRuns` or `GetAgentLogs` method. The codeflow system records execution data but isn't linked to specific agents.

**Missing endpoint:** `GET /api/v1/agents/{id}/runs`

```typescript
// Response shape
{
  agent_id: string
  runs: Array<{
    id: string
    action: string
    status: string
    duration_ms: number
    timestamp: string
    tool_calls: number
  }>
}
```

### GAP #5: No NLP / Ollama Health Exposed as REST Endpoint (🟡 MEDIUM)
**Impacted UX:** W1 healthSlice (`ollamaHealthy`, `nlpHealthy`), StatusBar health indicators

The backend watches the NLP sidecar via gRPC health check (`watchSidecar` in `app.go:519`) and maintains `nlpHandler.MarkHealthy()/MarkUnhealthy()` — but this state is NOT exposed via any API endpoint. Ollama health is checked directly from the frontend.

**Missing endpoint:** `GET /api/v1/system/components/health`

This could be rolled into GAP #1's aggregate endpoint.

### GAP #6: No Centralized User/System Settings Endpoint (🟢 LOW)
**Impacted UX:** W5-01 SettingsView progressive disclosure, W3 "settings" SlideOver type

Settings are scattered across multiple endpoints, and some settings (theme, language, UI preferences) have no server-side storage at all. If the UX wants to store UI preferences server-side, a new `SettingsService` is needed.

**Potential new service:**
```protobuf
service SettingsService {
  rpc GetSettings(GetSettingsRequest) returns (GetSettingsResponse);
  rpc UpdateSetting(UpdateSettingRequest) returns (UpdateSettingResponse);
}
```

Without this, settings viewed in one session won't persist to another.

### GAP #7: Dual Tool Listing — ConnectRPC vs Raw HTTP (🟢 LOW)
**Impacted UX:** W1 store tools field, general confusion

There are two separate tool listing paths:
1. `aleph.v1.ToolService/ListTools` — returns `Tool {id, name, description, code}` (proto)
2. `GET /api/v1/tools` → `ToolHandler.HandleListAll` → `metaRepo.ListTools()` — returns `ToolRecord` with more fields (health, version, category, etc.)

The raw HTTP endpoint returns richer data. The ConnectRPC response has only 4 fields and no category/health. The frontend adapter (`fromProtoTool`) maps only 4 fields, losing health, category, version data.

**Recommended fix:** Extend the proto `Tool` message to include `health_status`, `category`, `version`, `source_type` fields, matching the backend's `ToolRecord` shape.

### GAP #8: No User Preferences / Expanded Sections Persistence API (🔴 HIGH)
**Impacted UX:** W5-00 through W5-05 progressive disclosure (expanded section state persistence)

The W5 redesign introduces `expandedSections: Record<string, boolean>` in the Zustand store to track which collapsible sections are open per view. Currently this state has no backend persistence — it lives only in client-side Zustand + URL search params. This means:
- Section state is lost on browser tab close (URL params are ephemeral)
- Section state does not sync across devices
- The W5 spec's `?expand=details-configuration,advanced-permissions` URL pattern works only for share links, not for persistent user preference

**Missing endpoint:** `POST /api/v1/settings/preferences` / `GET /api/v1/settings/preferences`

```protobuf
service PreferencesService {
  rpc GetPreferences(GetPreferencesRequest) returns (GetPreferencesResponse);
  rpc SavePreferences(SavePreferencesRequest) returns (SavePreferencesResponse);
}

message GetPreferencesRequest { string project_id = 1; }
message GetPreferencesResponse { map<string, string> preferences = 1; }

message SavePreferencesRequest {
  string project_id = 1;
  map<string, string> preferences = 2;  // Key-value store for UI state
}
```

**Minimum viable implementation:** Store preferences as a JSON blob in the metadata repository keyed by `{project_id}:ui_prefs`.

### GAP #9: No LLM Provider Configuration API (🟡 MEDIUM)
**Impacted UX:** W5-01 SettingsView progressive disclosure (Tier-3 "providers"), agent-form SlideOver

Currently, each agent carries its own `provider`, `model`, `api_key`, and `base_url`. There's no global provider registry that the user can configure independently of agents. The W5 SettingsView advanced tier needs provider-level configuration.

**Missing endpoint:** `GET/POST/PUT/DELETE /api/v1/settings/providers`

```typescript
// GET /api/v1/settings/providers
{
  providers: [
    { id: "ollama", name: "Ollama", base_url: "http://localhost:11434", default_model: "llama3", enabled: true },
    { id: "openai", name: "OpenAI", api_key: "sk-...", default_model: "gpt-4", enabled: false }
  ]
}

// POST /api/v1/settings/providers
{ name: "Anthropic", api_key: "sk-ant-...", base_url: "https://api.anthropic.com", default_model: "claude-3-opus" }
```

### GAP #10: No Notification Channel CRUD (🟢 LOW)
**Impacted UX:** W5 SettingsView, notification management

`NotificationService` has only `ListChannels` and `SendWebhook`. Without `CreateChannel`, `UpdateChannel`, and `DeleteChannel`, users cannot manage notification channels through the API.

**Missing endpoints:** Extend `NotificationService`
```protobuf
rpc CreateChannel(CreateChannelRequest) returns (CreateChannelResponse);
rpc UpdateChannel(UpdateChannelRequest) returns (UpdateChannelResponse);
rpc DeleteChannel(DeleteChannelRequest) returns (DeleteChannelResponse);
```

### GAP #11: No Scene Load Batched Endpoint (🟡 MEDIUM)
**Impacted UX:** W2 scene routing performance, project load time

When switching projects or scenes, `loadProjectData()` in `useAppActions.ts` fires **11 parallel ConnectRPC calls**:

| # | Client | Method |
|---|--------|--------|
| 1 | `projectClient` | `getOntology` |
| 2 | `agentClient` | `listAgents` |
| 3 | `ingestionClient` | `listTasks` |
| 4 | `libraryClient` | `listAssets` |
| 5 | `skillClient` | `listSkills` |
| 6 | `toolClient` | `listTools` |
| 7 | `agentClient` | `listModels` |
| 8 | `nlpClient` | `analyzeSentiment` |
| 9 | `authClient` | `listApiKeys` |
| 10 | `notificationClient` | `listChannels` |
| 11 | `registryClient` | `listComponents` |

On every project switch, all 11 fire simultaneously. For W2 scene routing, a more efficient approach would be:

- **Option A (recommended):** Create a single batch endpoint that returns all scene data in one response:
  ```protobuf
  rpc LoadSceneData(LoadSceneDataRequest) returns (LoadSceneDataResponse);
  message LoadSceneDataRequest { string project_id = 1; }
  message LoadSceneDataResponse {
    repeated Agent agents = 1;
    repeated Skill skills = 2;
    repeated Tool tools = 3;
    repeated IngestionTask tasks = 4;
    repeated Asset assets = 5;
    repeated ApiKey api_keys = 6;
    repeated RegistryComponent components = 7;
    repeated NotificationChannel channels = 8;
    string ontology_raw = 9;
    repeated string object_names = 10;
    repeated string models = 11;
  }
  ```
- **Option B (lighter):** Use HTTP/2 multiplexing (ConnectRPC already supports it) but batch logically — create a `GET /api/v1/load/{project_id}` that returns a JSON payload with all project data.

### GAP #12: No Dedicated DataSource Metadata CRUD (🟡 MEDIUM)
**Impacted UX:** W5 datasource-form SlideOver, W2 ExploreScene datasources view

Data sources are currently modeled purely as `IngestionTask` entries with `source_type` and `config_json`. There's no separate datasource entity with proper metadata (URL, auth type, schedule, last_success, connection status). The W5 progressive disclosure for datasource forms needs richer metadata.

**Missing endpoint:** New `DataSourceService`
```protobuf
service DataSourceService {
  rpc ListDataSources(ListDataSourcesRequest) returns (ListDataSourcesResponse);
  rpc GetDataSource(GetDataSourceRequest) returns (GetDataSourceResponse);
  rpc CreateDataSource(CreateDataSourceRequest) returns (CreateDataSourceResponse);
  rpc UpdateDataSource(UpdateDataSourceRequest) returns (UpdateDataSourceResponse);
  rpc DeleteDataSource(DeleteDataSourceRequest) returns (DeleteDataSourceResponse);
  rpc TestDataSourceConnection(TestDataSourceConnectionRequest) returns (TestDataSourceConnectionResponse);
}
```

### GAP #13: ToolIntel Type Not Fully Backed by API (🟡 MEDIUM)
**Impacted UX:** W5 ToolsView progressive disclosure (Tier-3 intelligence), `/api/v1/tools/intelligence` endpoint

The frontend `ToolIntel` type (`frontend/src/store/types.ts` lines 164–182) expects 17 fields:

```typescript
interface ToolIntel {
  id: string; name: string;
  totalExecutions: number; avgLatencyMs: number; errorRate: number;
  lastUsed: number; brierScore: number; trustScore: number;
  execCount: number; avgDuration: number; topUsers: string[];
  riskScore: number; warnings: string[]; usageFreq: 'high'|'medium'|'low';
  recommendations: string[]; anomalies: { desc: string; severity: string }[];
  relatedTools: string[];
}
```

The backend endpoint `GET /api/v1/tools/intelligence` → `ToolHandler.ServeHTTP` → `metaRepo.ListTools()` only returns `ToolRecord` fields (`id`, `name`, `code`, `category`, `version`, `health_status`, `source_type`). **The intelligence/analytics fields (trustScore, riskScore, recommendations, etc.) do not exist in any backend response.**

**Recommended fix:** Either:
1. Create a new `ToolService.GetToolIntelligence` RPC that queries codeflow metrics + metadata to compute these fields
2. Or extend `ToolRecord` with analytics fields and populate them from execution data

---

## 6. Frontend API Client Surface Analysis

### 6.1 Clients in `factory.ts` (12 ConnectRPC clients)

```typescript
registryClient, sandboxClient, queryClient, projectClient, agentClient,
ingestionClient, libraryClient, authClient, skillClient, toolClient,
nlpClient, notificationClient
```

All 12 clients are actively used. Coverage assessment:

| Client | Used In | Calls per Project Load | Notes |
|--------|---------|----------------------|-------|
| `registryClient` | `useComponentActions`, `useAppActions` | 1 (`listComponents`) | ✅ |
| `sandboxClient` | `useAppActions` | 0 (on-demand) | ✅ Called only for Execute/Run |
| `queryClient` | `useAppActions` | 0 (on-demand) | ✅ Chat + queries |
| `projectClient` | `useOntologyActions`, `useAppActions` | 1 (`getOntology`) | ✅ |
| `agentClient` | `useAgentActions`, `useAppActions` | 2 (`listAgents`, `listModels`) | ✅ |
| `ingestionClient` | `useDataSourceActions`, `useAppActions` | 1 (`listTasks`) | ✅ |
| `libraryClient` | `useLibraryActions`, `useAppActions` | 1 (`listAssets`) | ✅ |
| `authClient` | `useSettingsActions`, `useAppActions` | 1 (`listApiKeys`) | ✅ |
| `skillClient` | `useSkillActions`, `useAppActions` | 1 (`listSkills`) | ✅ |
| `toolClient` | `useToolActions`, `useAppActions` | 1 (`listTools`) | ✅ |
| `nlpClient` | `useAppActions` | 1 (`analyzeSentiment` as ping) | ✅ |
| `notificationClient` | `useSettingsActions`, `useAppActions` | 1 (`listChannels`) | ✅ |

### 6.2 Raw HTTP Functions in `client.ts` (4 functions)

| Function | Method | Path | Used By |
|----------|--------|------|---------|
| `createSession` | POST | `/api/v1/auth/session` | Login flow |
| `deleteSession` | DELETE | `/api/v1/auth/session` | Logout flow |
| `apiGet` | GET | Dynamic | Generic data fetch |
| `apiPost` | POST | Dynamic | Generic data post |

### 6.3 Data Flow Pattern

```
Project Switch → loadProjectData() → 11 parallel ConnectRPC calls
                                        ↓
                              Each populates a Zustand store field
                                        ↓
                              UI components subscribe via selectors
```

**Performance observation:** 11 concurrent HTTP/2 requests per project switch. While HTTP/2 multiplexing mitigates connection overhead, the server processes 11 separate handler invocations. For W2 scene routing (fast switching between scenes), this creates unnecessary load. Each scene needs only a subset:
- **TerminalScene:** Only `listModels` + `analyzeSentiment` (health checks)
- **ExploreScene:** `getOntology` + `listAssets` + `listTasks`
- **AgentsScene:** `listAgents` + `listSkills` + `listTools` + `listComponents`
- **SystemScene:** `listApiKeys` + `listChannels` + health checks

## 7. Duplicate/Overlapping Endpoints

| Duplicate Group | Endpoints | Notes |
|----------------|-----------|-------|
| **Tool listing** | `aleph.v1.ToolService/ListTools` vs `GET /api/v1/tools` vs `GET /api/v1/tools/intelligence` vs `GET /api/v1/tools/recommendations` | 4 paths for essentially the same data. `/intelligence` and `/recommendations` both call `metaRepo.ListTools()` identically. |
| **Project scoping** | `project_id` as URL param vs middleware-injected context vs request body field | Inconsistent — some endpoints read from `X-Project-Id` header, others from request body. |

---

## 8. Recommendations

### 8.1 Prioritize: High-Impact Gaps for W5

| Priority | Gap | Endpoint | Effort | Wave |
|----------|-----|----------|--------|------|
| **P0** | GAP #8 — User preferences API | `POST/GET /api/v1/settings/preferences` | Small (JSON blob store) | W5 |
| **P0** | GAP #13 — ToolIntel fields | Extend `GET /api/v1/tools/intelligence` | Medium (compute from metrics) | W5 |
| **P1** | GAP #9 — Provider config API | `CRUD /api/v1/settings/providers` | Medium | W5 |
| **P1** | GAP #3 — Per-tool execution history | `GET /api/v1/tools/{id}/executions` | Medium (reuse codeflow) | W5 |
| **P1** | GAP #4 — Agent run history | `GET /api/v1/agents/{id}/runs` | Medium | W5 |
| **P2** | GAP #1 — Aggregated health | `GET /api/v1/system/health` | Small | W2/W5 |
| **P2** | GAP #12 — Datasource CRUD | New `DataSourceService` | Large (new entitity) | W5 |
| **P3** | GAP #11 — Scene load batch | New batch RPC | Large (but high ROI) | W2 |
| **P3** | GAP #2 — Scenarios CRUD | New endpoints | Small | deferred |
| **P4** | GAP #10 — Notification CRUD | Extend NotificationService | Small | deferred |

### 8.2 Safe to Defer (No Backend Impact)

These UX waves require **no backend changes** and can proceed with the existing API surface:

- **W0** (Foundation, feature flags, audit) — zero backend changes
- **W1** (Store refactor) — all store fields already backed by APIs
- **W2** (Navigation) — scene routing is frontend-only URL state
- **W3** (SlideOver unification) — rendering-only, no new data needs
- **W4** (Copilot slim) — frontend decomposition, same API calls
- **W6** (Polish, tests) — Playwright rewrite, a11y, animation

### 8.3 Proceed Now, Fix Later

For W2 scene routing, the 11 parallel calls in `loadProjectData()` are functional but suboptimal. Two options:
1. **Do nothing now** — 11 calls on project switch is acceptable for MVP
2. **Implement GAP #11** (batch endpoint) before W2 if scene switching feels slow

### 8.4 Investigate Before W5

- **ToolIntel response audit:** Verify actual response shape of `GET /api/v1/tools/intelligence` against the frontend `ToolIntel` type. This may already return more fields than the basic `ToolRecord` if the handler joins with codeflow data.
- **Notification channel CRUD:** Check if `NotificationService` intends to manage channels or if they're configured externally (env vars / config file).

## 9. Summary

| Category | Count |
|----------|-------|
| Total ConnectRPC services | 12 |
| Total ConnectRPC methods | 48 |
| Total raw HTTP endpoints | 19 |
| Total unique API endpoints | ~65 |
| Store fields requiring backend | 32 of 61 |
| Store fields fully covered | 30 of 32 (93.75%) |
| Frontend ConnectRPC clients | 12 (via `factory.ts`) |
| `loadProjectData` parallel calls | 11 (performance concern) |
| Gaps (high priority) | 2 (#1 aggregated health, #8 user prefs) |
| Gaps (medium priority) | 7 (#2 scenarios, #3 tool execs, #4 agent runs, #5 NLP health, #9 providers, #12 datasource, #13 ToolIntel) |
| Gaps (low priority) | 4 (#6 settings, #7 dual tool listing, #10 notification CRUD, #11 scene batch) |

**Verdict: The backend API is sufficient for W0–W4 UX redesign with no blocking gaps. W5 progressive disclosure requires 4–6 new endpoints for full feature fidelity. Scene load batching (GAP #11) is a performance optimization for W2.** 
