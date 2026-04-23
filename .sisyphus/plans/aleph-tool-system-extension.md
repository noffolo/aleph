# Aleph Tool System Extension Plan

## TL;DR

> **Quick Summary**: Extend existing Aleph tool system to support: 1) pre-configured domain tool packages (finance, OSINT, social), 2) smart discovery/adaptation of tools from MCP/GitHub, and 3) OpenCode-like self-creation capabilities - all grounded in current architecture (tool registry, sandbox, agents).
> 
> **Deliverables**:
> - Extended tool metadata with categories, versions, health checks
> - Chat-UI hybrid interface (`/tool` commands + tool management views)
> - Domain tool packages (finance, OSINT, human ecosystems)
> - MCP discovery engine (OpenBB, Great Expectations, Ghidra integration)
> - Tool creation DSL and sandbox enhancement
> - Auto-diagnostic/repair subsystem
> 
> **Estimated Effort**: XL (multi-wave due to architectural depth)
> **Parallel Execution**: YES - 3 waves (registry → packages → advanced)
> **Critical Path**: Tool metadata → discovery → adaptation testing → self-creation DSLs

---

## Context

### Original Request
Extend Aleph's existing tool system to support:
1. **Pre‑configured tools**: Domain-specific packages (finance, OSINT, social)
2. **Tool discovery/config**: User suggests → Aleph verifies → downloads/adapts
3. **Tool self‑creation**: Like OpenCode - Aleph writes its own tools

**Plus**: Auto‑diagnostic and auto‑repair for tools
**Constraint**: MUST work within current Aleph architecture, not invent a new app

### Aleph‑Current Architecture (Analyzed)
**Tool System**:
- `internal/api/handler/tool.go`: Connect RPC API (`ListTools`, `CreateTool`, `DeleteTool`)
- `internal/repository/metadata.go:MetadataRepository`: `ToolRecord` (id, name, description, code)
- `internal/sandbox/exec_sandbox.go`: Isolated Go/Python execution
- Hardcoded tools in `query.go`: `analyze_sentiment`, `get_trust_score`
- Dynamic DB tools: `search_data` via DSL/DuckDB

**Skill System**:
- `internal/api/handler/skill.go`: Tool chains (`system_skills` with `tool_ids` JSON)
- Ontology DSL: `.aleph` files → `dsl.Program` in `query.go`

**Agent System**:
- `internal/api/handler/agent.go`: Project‑scoped storage, skill chain execution
- Model provider configuration (Ollama connections)

### Research Synthesis
From repo‑analysis‑tools‑skills.md:

**MCP Ecosystem Opportunities**:
- OpenBB Platform: Production MCP server (`openbb‑mcp‑server`)
-

Great Expectations: Existing MCP server (`gx‑mcp‑server`)
-

Ghidra: Community MCP implementations (222+ tools)
-

MCP Protocol: Standardized tool/resource/prompt primitives

**Tool Planning Documents**:
- CodeFlow.md: Visualization engine (privacy‑first, incremental processing)
-

HumanEcosystems.md: Relational analysis layer atop DuckDB
-

Shadowbroker.md: OSINT platform (60+ feeds) → OSINT Gateway proxy

---

## Work Objectives

### Core Objective
Transform Aleph from fixed‑tool system to adaptive‑tool platform while preserving:
1. Existing tool registry, sandbox, agent, skill architecture
2. Chat‑based user interaction (coherence)
3. Simple yet impressive UX (chat with `/` commands, UI views accessible)

### Concrete Deliverables
1. Extended Tool Registry API (metadata categories, health checks, versioning)
2. Chat‑UI hybrid (`/tool` commands triggering tool management views)
3. Domain Tool Packages (`finance`, `osint`, `human‑ecosystems`)
4. MCP Discovery Engine (`mcp://` URI scanning, verification sandbox)
5. Tool Creation DSL (OpenCode‑like templates, sandbox testing)
6. Auto‑Diagnostic/Repair Subsystem (health checks, error classification, fixing)

### Definition of Done
- Each tool package installable via `/tool install finance`
- User can say "Aleph, add prophet forecasting tool" → verified/adapted
- Developer can describe tool in DSL → auto‑generates working code
- Broken tools automatically diagnosed (with user approval) and repaired

### Must Have
- Backward compatibility with existing tools/skills/agents
- Chat‑first interaction with option for UI tool management views
- Hybrid testing strategy: TDD for core + agent‑executed QA for packages

### Must NOT Have (Guardrails)
- NO replacement of existing tool registry/sandbox/skill architecture
- NO pure‑UI‑only workflow (must support chat)
- NO replacement of current agents/skills system

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** - ALL verification is agent‑executed. No exceptions.

### Test Decision
-methodology**Hybrid approach**:
- Core framework extensions: TDD (Go tests, `go test`)
  
- Skill packages: Agent‑executed QA scenarios
-
External tool integration: Agent‑executed verification sandbox + evidence capture

### QA Policy
Every new capability includes agent‑executed QA scenarios (Playwright/Bash).
Evidence saved to `.sisyphus/evidence/`.

**Chat Interface QA**: Bash (tmux/curl) simulating chat commands
**UI Views QA**: Playwright (if UI component added)
**Tool Verification**: Python sandbox testing (isolated environment)
**MCP Discovery**: curl to MCP servers + response validation

---

## Execution Strategy

### Priority Order (Based on Dependencies)
**Wave 1** (Foundation - extends existing architecture):
1. Tool metadata extensions (categories, versions, health)
2. Chat‑UI hybrid interfaces (`/tool` commands)
3. Advanced sandbox for verification testing

**Wave 2** (Capabilities 1‑2 - demonstrates extended system):
4. Domain tool packages (finance, OSINT, human‑ecosystems)
5. MCP discovery engine + adaptation pipeline
6. User‑initiated tool suggestion workflow

**Wave 3** (Capability 3 + advanced):
7. Tool creation DSL + sandbox enhancements
8. Auto‑diagnostic/repair subsystem
9. Integration with CodeFlow/HumanEcosystems/Shadowbroker concepts

### Parallel Execution Waves

```
Wave 1 (Start Immediately - foundation extensions):
├── Task 1: Extended tool metadata schema + migrations [deep]
├── Task 2: Chat‑UI hybrid interface (/tool commands) [quick]
├── Task 3: Enhanced sandbox for verification testing [unspecified‑high]
└── Task 4: Health check system for tools [quick]

Wave 2 (After Wave 1 - domain packages + discovery):
├── Task 5: Finance package (prophet_forecast, openbb_market_data) [deep]
├── Task 6: OSINT package (shadowbroker integration) [unspecified‑high]
├── Task 7: Human‑ecosystems package [unspecified‑high]
├── Task 8: MCP discovery engine (scanning, URIs) [deep]
├── Task 9: User‑initiated tool suggestion workflow [quick]
└── Task 10: Adaptation pipeline (verification → sandbox → registry) [deep]

Wave 3 (After Wave 2 - self‑creation + auto‑repair):
├── Task 11: Tool creation DSL (OpenCode‑like templates) [artistry]
├── Task 12: Sandbox enhancements for test‑driven creation [unspecified‑high]
├── Task 13: Auto‑diagnostic subsystem (error classification) [deep]
├── Task 14: Auto‑repair strategies (patterns, regeneration) [deep]
└── Task 15: Integration with CodeFlow/HumanEcosystems/Shadowbroker [visual‑engineering]

Wave FINAL (After ALL tasks — unified verification):
├── Task F1: End‑to‑end chat‑based tool lifecycle [unspecified‑high]
├── Task F2: Entire MCP ecosystem connectivity test [unspecified‑high]
├── Task F3: Self‑repair demonstration (simulated break/fix) [unspecified‑high]
└── Task F4: Cross‑context adaptability verification [oracle]
```

### Dependency Matrix (abbreviated)

```

- **1**: Extends existing tool.go, repository - 5‑10, all
-

**2**: /tool command parsing → UI view triggers - 9, all
-

**3**: Enhanced sandbox execution → 8‑12, 14
-

**5‑7**: Depends on 1 (metadata), 3 (sandbox) → 9‑10, 15
-

**8**: MCP scanning → 9‑10 (adaptation), 11‑12 (creation)
-

**9**: User suggestion workflow → 8‑10 (discovery/adaptation)
-

**11‑12**: Tool creation DSL → 13‑14 (auto‑repair), 15
-

**13‑14**: Auto‑diagnostic/repair → depends on 1‑12, final verification
```

---

## TODOs

> Implementation + Test = ONE Task. Never separate.
> EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.
> **A task WITHOUT QA Scenarios is INCOMPLETE. No exceptions.**

- [ ] 1. Extended tool metadata schema + migrations

  **What to do**:
  - Add fields to `ToolRecord`: `category`, `version`, `health_status`, `last_checked_at`, `source_type` (local/mcp/github)
  - Create migration SQL for `system_tools` table
  - Update `repository.MetadataRepository` methods
  - Add health check API endpoint

  **Must NOT do**:
  - Change existing tool names/descriptions/code fields
  - Break backward compatibility with current tools

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Database schema changes require careful analysis of existing queries and migrations
  - **Skills**: [`golang-pro`, `sql-pro`]
    - `golang-pro`: For Go repository pattern updates and API endpoint implementation
    - `sql-pro`: For SQL migration design to ensure data integrity and performance

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Task 2, 3, 4)
  - **Blocks**: Tasks 5‑10 (domain packages need metadata)
  - **Blocked By**: None (can start immediately)

  **References** (CRITICAL - Be Exhaustive):

  **Pattern References** (existing code to follow):
  - `internal/api/handler/tool.go:20-51` - Existing tool CRUD API pattern
  - `internal/repository/metadata.go:234-259` - Repository structure for `ToolRecord`

  **API/Type References** (contracts to implement against):
  - `internal/repository/metadata.go:ToolRecord` - Current struct definition
  - `internal/repository/metadata.go:44-46` - Table schema (system_tools CREATE TABLE statement)

  **Test References** (testing patterns to follow):
  - `internal/api/handler/tool_test.go` does not exist - create new tests
  - Current testing patterns: `internal/api/handler/agent_test.go` for API testing pattern

  **External References** (libraries and frameworks):
  - Go migration tools: `github.com/pressly/goose/v3` - Migration pattern (check migration directory)

   **WHY Each Reference Matters** (explain the relevance):
  - `internal/api/handler/tool.go:20 51`: Pattern for adding new fields to API request/response
  - `internal/repository/metadata.go:234-259`: How to extend repository methods safely
  - `internal/repository/metadata.go:ToolRecord`: Must maintain backward compatibility

  **Acceptance Criteria**:

  **If TDD (tests enabled)**:
  - [ ] Test file updated: `internal/api/handler/tool_test.go`
  - [ ] `go test ./internal/api/handler/...` → PASS (no new failures)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Extended metadata API accepts new fields
    Tool: Bash (curl)
    Preconditions: Aleph running with current tools
    Steps:
      1. Create tool with category field: `curl -X POST http://localhost:8080/api/v1/tools -d '{"name":"test", "description":"test", "category":"finance"}'`
      2. Retrieve tool: `curl http://localhost:8080/api/v1/tools/test`
      3. Parse JSON response for `category` field
    Expected Result: Response contains `"category":"finance"` field
    Failure Indicators: Missing category field, 400/500 error
    Evidence: .sisyphus/evidence/task-1-metadata-api.json

  Scenario: Health check endpoint returns status
    Tool: Bash (curl)
    Preconditions: Tool created with extended metadata
    Steps:
      1. Call health endpoint: `curl http://localhost:8080/api/v1/tools/test/health`
      2. Parse JSON for `health_status` field
    Expected Result: Response contains `"health_status":"unknown"` or similar
    Failure Indicators: Missing endpoint, incorrect status type
    Evidence: .sisyphus/evidence/task-1-health-endpoint.json
  ```

  **Evidence to Capture**:
  - [ ] API responses showing new fields
  - [ ] Migration SQL file created
  - [ ] Health endpoint working

  **Commit**: YES (groups with Task 2)
  - Message: `feat(tools): extend metadata with categories, health checks`
  - Files: `internal/api/handler/tool.go`, `internal/repository/metadata.go`, `migrations/YYYYMMDD_add_tool_metadata_fields.sql`
  - Pre-commit: `go test ./internal/api/handler/...`

---

- [ ] 2. Chat‑UI hybrid interface (/tool commands)

  **What to do**:
  - Add `/tool` command parser to chat interface
  - Implement chat command: `/tool install {package}`
  - Create UI view trigger: chat command optionally opens tool management view
  - Add basic UI component for tool management (if UI capability exists)

  **Must NOT do**:
  - Create pure UI‑only workflow (must support chat commands)
  - Replace existing chat interaction patterns

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI component creation and chat‑UI integration requires frontend design
  - **Skills**: [`react-expert`, `typescript-pro`, `cli-developer`]
    - `react-expert`: For React UI component if Aleph has UI layer
    - `typescript-pro`: TypeScript interface for command parsing
    - `cli-developer`: Chat command interface design similar to CLI

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Task 1, 3, 4)
  - **Blocks**: Task 9 (user‑initiated suggestion workflow)
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - Frontend chat interface files if exists (check `frontend/` or `ui/`)
  - Existing command parsing patterns in Aleph chat

  **API/Type References**:
  - Chat message type definitions
  - Command routing patterns

  **Test References**:
  - If UI tests exist: frontend test patterns

  **External References**:
  - Command‑line parsing: `commander.js` or similar patterns

  **WHY Each Reference Matters**:
  - Need to understand Aleph's current chat architecture to extend it correctly

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Command parsing tests added
  - [ ] UI component tests if applicable

  **QA Scenarios**:

  ```
  Scenario: /tool install command triggers response
    Tool: Bash (tmux or curl simulating chat)
    Preconditions: Aleph running with extended metadata
    Steps:
      1. Send chat message: "/tool install finance"
      2. Capture response text
      3. Verify response acknowledges command
    Expected Result: Response contains "Installing finance package..." or similar
    Failure Indicators: Command not recognized, error response
    Evidence: .sisyphus/evidence/task-2-chat-command.txt

  Scenario: /tool command optionally opens UI view
    Tool: Playwright (if UI layer exists)
    Preconditions: UI accessible, chat interface available
    Steps:
      1. Navigate to chat interface
      2. Enter "/tool list"
      3. Wait for UI tool management view to appear (if designed)
      4. Screenshot view
    Expected Result: Tool management UI appears or chat responds with list
    Failure Indicators: UI missing, no response
    Evidence: .sisyphus/evidence/task-2-ui-view.png (if UI appears)
  ```

  **Evidence to Capture**:
  - [ ] Chat command successful response
  - [ ] UI view screenshot (if applicable)

   **Commit**: YES (groups with Task section)
   - Message: `feat(chat): add /tool commands for tool management`
   - Files: Chat command parsing components, optional UI components
   - Pre-commit: `npm test` or equivalent if UI components

---

- [ ] 3. Enhanced sandbox for verification testing

  **What to do**:
  - Extend `exec_sandbox.go` to support verification test mode
  - Add test isolation: separate environment for each verification run
  - Implement test result capture (stdout/stderr, exit codes, timing)
  - Add safety checks: timeout limits, resource limits, network isolation
  - Create verification API endpoint for agent‑executed QA

  **Must NOT do**:
  - Break existing sandbox execution for regular tools
  - Remove current safety measures
  - Allow network access during verification tests (unless explicitly configured)

  **Recommended Agent Profile**:
  - **Category**: `unspecified‑high`
    - Reason: Sandbox enhancements require careful security and isolation design
  - **Skills**: [`golang‑pro`, `python‑pro`, `secure‑code‑guardian`]
    - `golang‑pro`: Go sandbox implementation changes
    - `python‑pro`: Python sidecar sandbox integration if applicable
    - `secure‑code‑guardian`: Security hardening for test isolation

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Task 1, 2, 4)
  - **Blocks**: Tasks 8‑10 (discovery/adaptation need verification), 12 (tool creation testing)
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/sandbox/exec_sandbox.go:1‑100` - Current sandbox execution flow
  - Existing test isolation patterns in Aleph codebase

  **API/Type References**:
  - Sandbox request/response structures
  - Verification test result formats

  **Test References**:
  - Sandbox unit tests if exist
  - Security test patterns

  **External References**:
  - Docker/container isolation patterns for reference
  - Timeout/limitation best practices

  **WHY Each Reference Matters**:
  - `exec_sandbox.go`: Must understand current execution flow to extend safely
  - Security patterns: Verification tests may execute untrusted code

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Sandbox verification tests added
  - [ ] Security isolation tests pass

  **QA Scenarios**:

  ```
  Scenario: Verification sandbox executes test code safely
    Tool: Bash (curl)
    Preconditions: Enhanced sandbox running
    Steps:
      1. POST verification request: `curl -X POST http://localhost:8080/api/v1/sandbox/verify -d '{"code":"print(\\\"hello\\\")", "language":"python"}'`
      2. Parse response for success status
      3. Check stdout contains "hello"
    Expected Result: Response contains `{"success":true, "stdout":"hello\\n"}`
    Failure Indicators: Timeout, security violation, missing endpoint
    Evidence: .sisyphus/evidence/task‑3‑verification‑success.json

  Scenario: Verification sandbox isolates malicious code
    Tool: Bash (curl)
    Preconditions: Enhanced sandbox with security isolation
    Steps:
      1. Attempt dangerous code: `curl -X POST http://localhost:8080/api/v1/sandbox/verify -d '{"code":"import os; os.system(\\\"rm -rf /\\\"})", "language":"python"}'`
      2. Parse response for error/security violation
    Expected Result: Code rejected or safely contained (no actual deletion)
    Failure Indicators: Code executes successfully, system damage
    Evidence: .sisyphus/evidence/task‑3‑security‑containment.json
  ```

  **Evidence to Capture**:
  - [ ] Verification API responses
  - [ ] Security containment test results

  **Commit**: YES
   - Message: `feat(sandbox): enhance verification testing with isolation`
   - Files: `internal/sandbox/exec_sandbox.go`, `internal/api/handler/verification.go`
   - Pre‑commit: `go test ./internal/sandbox/...`

---

-
[ ] 4. Health check system for tools

  **What to do**:
  - Implement regular health check scheduler for tools
  - Create health check types: code syntax validation, dependency availability, execution test
  - Store health status in extended metadata (Task 1)
  - Add health check history table for trend analysis
  - Create health dashboard API endpoint

  **Must NOT do**:
  - Perform health checks that modify tool code
  - Disable tools based on health status without user approval
  - Create excessive load with too‑frequent checks

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Health checking is straightforward scheduling + status tracking
  - **Skills**: [`golang‑pro`, `devops‑engineer`]
    - `golang‑pro`: For scheduler implementation and API
    - `devops‑engineer`: For health monitoring patterns and alert strategies

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Task 1, 2, 3)
  - **Blocks**: Tasks 13‑14 (auto‑diagnostic/repair)
  - **Blocked By**: Task 1 (extends ToolRecord metadata)

  **References**:

  **Pattern References**:
  - Existing scheduler patterns in Aleph codebase
  - Tool execution patterns for test runs

  **API/Type References**:
  - Tool metadata (Task 1 additions)
  - Health status enum definitions

  **Test References**:
  - Scheduler testing patterns

  **External References**:
  - Cron scheduling libraries (`github.com/robfig/cron/v3`)
  - Health check best practices

  **WHY Each Reference Matters**:
  - Need to integrate with Task 1 metadata schema
  - Should follow Aleph's existing scheduling approach

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Health check scheduler tests
  - [ ] API endpoint tests

  **QA Scenarios**:

  ```
  Scenario: Health check scheduler runs periodic checks
    Tool: Bash (curl)
    Preconditions: Tool registered, health system enabled
    Steps:
      1. Manually trigger health check: `curl -X POST http://localhost:8080/api/v1/tools/test/health/check`
      2. Wait 10 seconds
      3. Retrieve health status: `curl http://localhost:8080/api/v1/tools/test/health`
    Expected Result: Status shows `last_checked_at` updated, `health_status` populated
    Failure Indicators: No update, error status
    Evidence: .sisyphus/evidence/task‑4‑health‑check.json

  Scenario: Health dashboard shows tool status
    Tool: Bash (curl)
    Preconditions: Multiple tools with health statuses
    Steps:
      1. Request dashboard: `curl http://localhost:8080/api/v1/tools/health/dashboard`
      2. Parse JSON array of tool statuses
    Expected Result: Array contains all tools with health status fields
    Failure Indicators: Missing tools, incorrect status format
    Evidence: .sisyphus/evidence/task‑4‑dashboard.json
  ```

  **Evidence to Capture**:
  - [ ] Health check execution evidence
  - [ ] Dashboard response showing multiple tools

   **Commit**: YES
   - Message: `feat(tools): add health check system with scheduler`
   - Files: `internal/tool/health/checker.go`, `internal/tool/health/scheduler.go`, `internal/api/handler/health.go`
   - Pre‑commit: `go test ./internal/tool/health/...`

---

**WAVE 2 TASKS (Domain Packages + Discovery)**

- [ ] 5. Finance package (prophet_forecast, openbb_market_data)

  **What to do**:
  - Create `finance` tool package with core tools:
    1. `prophet_forecast`: Time‑series forecasting using Facebook Prophet
    2. `openbb_market_data`: Proxy to OpenBB MCP server for financial data
    3. `sentiment_analysis`: Extend current sentiment analysis with financial news focus
  - Package structure: `tools/finance/` directory with Go/Python implementations
  - Package metadata: `package.json` (or equivalent) describing version, dependencies
  - Install script: `/tool install finance` triggers package deployment

  **Must NOT do**:
  - Modify existing non‑finance tools
  - Require external API keys without user configuration
  - Store sensitive financial data without encryption

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Financial tool integration requires careful data handling and MCP protocol understanding
  - **Skills**: [`python‑pro`, `golang‑pro`, `api‑designer`]
    - `python‑pro`: Prophet implementation and financial libraries
    - `golang‑pro`: OpenBB MCP proxy implementation in Go
    - `api‑designer`: Design clean financial data API interfaces

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6‑10)
  - **Blocks**: Task 9 (user‑initiated suggestion workflow for finance tools)
  - **Blocked By**: Tasks 1‑4 (metadata, sandbox verification, health)

  **References**:

  **Pattern References**:
  - Existing tool implementation in `query.go` (`analyze_sentiment`)
  - Sandbox execution patterns

  **API/Type References**:
  - MCP protocol specification for OpenBB integration
  - Prophet library API

  **Test References**:
  - Financial data testing patterns (mock data, no real API calls)

  **External References**:
  - Facebook Prophet documentation
  - OpenBB MCP server documentation

  **WHY Each Reference Matters**:
  - `analyze_sentiment` in `query.go`: Pattern for adding new tool to existing system
  - MCP protocol: Required for OpenBB integration

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Finance tool tests pass with mock data
  - [ ] Prophet forecasting integration tests

  **QA Scenarios**:

  ```
  Scenario: Finance package installs via /tool install finance
    Tool: Bash (tmux/curl simulating chat)
    Preconditions: Chat `/tool` command available (Task 2)
    Steps:
      1. Send chat: `/tool install finance`
      2. Wait for installation completion
      3. Verify finance tools registered: `curl http://localhost:8080/api/v1/tools?category=finance`
    Expected Result: API returns prophet_forecast and openbb_market_data tools
    Failure Indicators: Missing tools, installation error
    Evidence: .sisyphus/evidence/task‑5‑finance‑install.json

  Scenario: Prophet forecast tool executes successfully
    Tool: Bash (curl)
    Preconditions: Finance package installed, test time‑series data available
    Steps:
      1. Call prophet_forecast: `curl -X POST http://localhost:8080/api/v1/tools/prophet_forecast/execute -d '{"object_name":"sales_data", "periods":30}'`
      2. Parse response for forecast dataframe
    Expected Result: Response contains forecast JSON with dates and predictions
    Failure Indicators: Tool execution error, missing data
    Evidence: .sisyphus/evidence/task‑5‑prophet‑execution.json
  ```

  **Evidence to Capture**:
  - [ ] Package installation success
  - [ ] Tool execution results

  **Commit**: YES (groups with Tasks 6‑7)
   - Message: `feat(packages): add finance tool package with prophet_forecast`
   - Files: `tools/finance/prophet_forecast.go`, `tools/finance/openbb_proxy.go`, `tools/finance/package.json`
   - Pre‑commit: `go test ./tools/finance/...`

---

- [ ] 6. OSINT package (shadowbroker integration)

  **What to do**:
  - Create `osint` tool package integrating Shadowbroker OSINT platform
  - Implement OSINT Gateway proxy service (Go) translating Aleph gRPC → Shadowbroker HTTP API
  - Core OSINT tools:
    1. `osint_region_dossier`: Regional intelligence summary
    2. `osint_threat_level`: Threat assessment
    3. `osint_vessel_tracking`: AIS maritime tracking
    4. `osint_flight_tracking`: ADS‑B aircraft tracking
    5. `osint_correlation_alerts`: Cross‑source correlation
  - Add caching (DuckDB + in‑memory LRU), rate limiting, circuit breaker patterns
  - Legal compliance checks (GDPR, terms of service)

  **Must NOT do**:
  - Store sensitive OSINT data without encryption
  - Violate data source terms of service
  - Create unlimited API calls (must implement throttling)

  **Recommended Agent Profile**:
  - **Category**: `unspecified‑high`
    - Reason: OSINT integration requires network security, legal compliance, complex caching
  - **Skills**: [`golang‑pro`, `secure‑code‑guardian`, `api‑designer`]
    - `golang‑pro`: Gateway proxy implementation
    - `secure‑code‑guardian`: Network isolation and data security
    - `api‑designer`: Clean OSINT data API design

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 7‑10)
  - **Blocks**: Task 15 (integration with Shadowbroker concepts)
  - **Blocked By**: Tasks161‑4 (foundation)

  **References**:

  **Pattern References**:
  - Existing gRPC patterns in Aleph
  - HTTP client patterns

  **API/Type References**:
  - Shadowbroker API documentation from tool plan
  - OSINT data structures

  **Test References**:
  - Network service mocking patterns

  **External References**:
  - Circuit breaker libraries (`github.com/sony/gobreaker`)
  - Rate limiting libraries

  **WHY Each Reference Matters**:
  - Need to understand Aleph's gRPC patterns for proxy design
  - Shadowbroker API details required for integration

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] OSINT Gateway tests with mocked Shadowbroker
  - [ ] Caching and rate limiting tests

  **QA Scenarios**:

  ```
  Scenario: OSINT package installs successfully
    Tool: Bash (tmux/curl)
    Preconditions: Chat `/tool` command available
    Steps:
      1. Send chat: `/tool install osint`
      2. Verify installation
      3. Check OSINT tools registered: `curl http://localhost:8080/api/v1/tools?category=osint`
    Expected Result: API returns 5+ OSINT tools
    Failure Indicators: Missing tools, installation error
    Evidence: .sisyphus/evidence/task‑6‑osint‑install.json

  Scenario: OSINT gateway proxies request to Shadowbroker
    Tool: Bash (curl)
    Preconditions: OSINT package installed, Shadowbroker mock available
    Steps:
      1. Call osint_region_dossier: `curl -X POST http://localhost:8080/api/v1/tools/osint_region_dossier/execute -d '{"region":"north‑america"}'`
      2. Verify response contains region intelligence
    Expected Result: Response contains structured OSINT data
    Failure Indicators: Proxy failure, timeout, security violation
    Evidence: .sisyphus/evidence/task‑6‑osint‑proxy.json
  ```

  **Evidence to Capture**:
  - [ ] Package installation success
  - [ ] Gateway proxy functionality

  **Commit**: YES (groups with Tasks 5, 7)
   - Message: `feat(packages): add OSINT tool package with Shadowbroker integration`
   - Files: `tools/osint/gateway.go`, `tools/osint/cache.go`, `tools/osint/package.json`
   - Pre‑commit: `go test ./tools/osint/...`

---

- [ ] 7. Human‑ecosystems package

  **What to do**:
  - Create `human‑ecosystems` tool package implementing relational analysis layer
  - Tools based on HumanEcosystems.md tool plan:
    1. `he_research_profiles`: YAML research profile management
    2. `he_relational_engine`: Relationship graph analysis
    3. `he_geographic_context`: Geographic data enrichment
    4. `he_pattern_classifier`: Pattern detection and classification
    5. `he_plugin_viz`: Visualization plugin registry
  - Architecture: Layer atop existing DuckDB/PostgreSQL (not ingestion layer)
  - Implementation phases: Tier1 (profiles+enriched APIs+geo), Tier2 (relational engine), Tier3 (governance)

  **Must NOT do**:
  - Create new ingestion system (use existing DuckDB)
  - Store personal identifiable information without anonymization
  - Implement real‑time social media scraping (not part of plan)

  **Recommended Agent Profile**:
  - **Category**: `unspecified‑high`
    - Reason: Human ecosystems analysis requires complex relational modeling and privacy‑preserving design
  - **Skills**: [`golang‑pro`, `python‑pro`, `architecture‑designer`]
    - `golang‑pro`: Core engine implementation
    - `python‑pro`: Pattern classification and ML components
    - `architecture‑designer`: Relational layer architectural design

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5‑6, 8‑10)
  - **Blocks**: Task 15 (integration with HumanEcosystems concepts)
  - **Blocked By**: Tasks 1‑4

  **References**:

  **Pattern References**:
  - Existing relational query patterns in Aleph
  - DuckDB integration patterns

  **API/Type References**:
  - HumanEcosystems.md specification details
  - Research profile YAML schema

  **Test References**:
  - Graph algorithm testing patterns
  - Privacy compliance testing

  **External References**:
  - Graph database patterns for reference
  - Geographic data processing libraries

  **WHY Each Reference Matters**:
  - HumanEcosystems.md provides detailed requirements for this package
  - Must integrate with Aleph's existing DuckDB data layer

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Human‑ecosystems tool tests
  - [ ] Privacy compliance verification

  **QA Scenarios**:

  ```
  Scenario: Human‑ecosystems package installs
    Tool: Bash (tmux/curl)
    Preconditions: Chat `/tool` command available
    Steps:
      1. Send chat: `/tool install human‑ecosystems`
      2. Verify installation
      3. Check tools registered: `curl http://localhost:8080/api/v1/tools?category=human‑ecosystems`
    Expected Result: API returns 5+ HE tools
    Failure Indicators: Missing tools, installation error
    Evidence: .sisyphus/evidence/task‑7‑he‑install.json

  Scenario: Relational engine analyzes connections
    Tool: Bash (curl)
    Preconditions: HE package installed, test relational data available
    Steps:
      1. Call he_relational_engine: `curl -X POST http://localhost:8080/api/v1/tools/he_relational_engine/execute -d '{"entity_ids":["e1","e2","e3"]}'`
      2. Parse response for relationship graph
    Expected Result: Response contains relationship matrix and strength metrics
    Failure Indicators: Engine failure, data errors
    Evidence: .sisyphus/evidence/task‑7‑relational‑engine.json
  ```

  **Evidence to Capture**:
  - [ ] Package installation success
  - [ ] Relational analysis functionality

  **Commit**: YES (groups with Tasks 5‑6)
   - Message: `feat(packages): add human‑ecosystems tool package`
   - Files: `tools/he/engine.go`, `tools/he/profiles.go`, `tools/he/package.json`
   - Pre‑commit: `go test ./tools/he/...`

---

**WAVE 2 TASKS (Domain Packages + Discovery)**

- [ ] 8. MCP discovery engine (scanning, URIs)

  **What to do**:
  - Implement MCP server discovery subsystem:
    1. URI scanner: `mcp://` protocol support, local/remote server detection
    2. Server registration: Store discovered MCP servers in registry
    3. Tool schema extraction: Fetch tool schemas from MCP servers
    4. Health checking: Verify server availability, version compatibility
  - Support discovery sources:
    1. Static configuration (pre‑configured OpenBB, GX, Ghidra servers)
    2. Network scanning (local subnet, known ports)
    3. User‑provided URIs
    4. Community registry integration (future)
  - Security: Verify server certificates, sandbox isolation for unknown servers

  **Must NOT do**:
  - Automatically execute tools from untrusted servers
  - Bypass user approval for tool adoption
  - Create network scanning that violates network policies

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: MCP protocol implementation requires network security and protocol expertise
  - **Skills**: [`golang‑pro`, `mcp‑developer`, `secure‑code‑guardian`]
    - `golang‑pro`: Network client implementation
    - `mcp‑developer`: MCP protocol expertise
    - `secure‑code‑guardian`: Network security hardening

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5‑7, 9‑10)
  - **Blocks**: Tasks 9‑10 (user suggestion + adaptation), Tasks 11‑12 (tool creation)
  - **Blocked By**: Tasks 3 (sandbox verification), Tasks 1 (metadata)

  **References**:

  **Pattern References**:
  - Existing HTTP client patterns in Aleph
  - Network service discovery patterns

  **API/Type References**:
  - MCP protocol specification
  - Tool schema definition structures

  **Test References**:
  - Network client testing with mocks
  - Protocol parsing tests

  **External References**:
  - MCP SDK libraries (`github.com/modelcontextprotocol/sdk`)
  - DNS‑SD/mDNS discovery patterns

  **WHY Each Reference Matters**:
  - MCP protocol specification required for correct implementation
  - Network security patterns essential for safe discovery

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] MCP client tests with mocked servers
  - [ ] Discovery algorithm tests

  **QA Scenarios**:

  ```
  Scenario: MCP discovery finds configured OpenBB server
    Tool: Bash (curl)
    Preconditions: OpenBB MCP server mock available, configured in Aleph
    Steps:
      1. Trigger discovery scan: `curl -X POST http://localhost:8080/api/v1/tools/discovery/scan`
      2. Check discovery results: `curl http://localhost:8080/api/v1/tools/discovery/servers`
    Expected Result: Results include OpenBB server with tool count > 0
    Failure Indicators: No servers found, connection errors
    Evidence: .sisyphus/evidence/task‑8‑mcp‑discovery.json

  Scenario: Tool schema extraction from MCP server
    Tool: Bash (curl)
    Preconditions: MCP server discovered
    Steps:
      1. Extract tool schemas: `curl -X POST http://localhost:8080/api/v1/tools/discovery/servers/openbb/schemas`
      2. Verify schema structure
    Expected Result: JSON array of tool schemas with names, descriptions, parameters
    Failure Indicators: Empty array, malformed schemas
    Evidence: .sisyphus/evidence/task‑8‑schema‑extraction.json
  ```

  **Evidence to Capture**:
  - [ ] Server discovery results
  - [ ] Tool schema extraction examples

  **Commit**: YES
   - Message: `feat(discovery): add MCP server discovery engine`
   - Files: `internal/tool/discovery/scanner.go`, `internal/tool/discovery/mcp_client.go`
   - Pre‑commit: `go test ./internal/tool/discovery/...`

---

- [ ] 9. User‑initiated tool suggestion workflow

  **What to do**:
  - Implement complete user‑initiated tool suggestion flow:
    1. Chat command: "Aleph, add {tool_name}" or UI suggestion form
    2. Query parsing: Extract tool intent, search parameters
    3. Search phase: MCP discovery (Task 8) + GitHub code search
    4. Verification phase: Sandbox testing (Task 3) candidate tools
    5. Adaptation phase: Code adaptation for Aleph context
    6. Registration phase: Add to tool registry with metadata
  - User feedback loop: Progress updates, success/failure notifications
  - Fallback strategies: Multiple search sources, manual escalation

  **Must NOT do**:
  - Install tools without user confirmation
  - Skip verification testing
  - Modify user's data during adaptation

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Workflow orchestration builds on existing components
  - **Skills**: [`golang‑pro`, `api‑designer`]
    - `golang‑pro`: Workflow state management
    - `api‑designer`: Clean user feedback API design

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5‑8, 10)
  - **Blocks**: None (enables user‑initiated capability)
  - **Blocked By**: Tasks 1‑4 (foundation), Task 8 (discovery)

  **References**:

  **Pattern References**:
  - Existing workflow patterns in Aleph
  - Chat command parsing (Task 2)

  **API/Type References**:
  - Search request/response structures
  - User feedback message formats

  **Test References**:
  - Workflow state machine tests
  - User interaction tests

  **External References**:
  - GitHub API for code search
  - Workflow engine patterns

  **WHY Each Reference Matters**:
  - Builds directly on Task 2 chat command parsing
  - Uses Task 8 discovery and Task 3 verification

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Suggestion workflow tests
  - [ ] User feedback API tests

  **QA Scenarios**:

  ```
  Scenario: User suggests tool via chat command
    Tool: Bash (tmux simulating chat)
    Preconditions: Chat interface, discovery engine, sandbox verification
    Steps:
      1. Send chat: "Aleph, add prophet forecasting tool"
      2. Capture immediate acknowledgement
      3. Wait for workflow completion (poll status)
      4. Verify tool registered: `curl http://localhost:8080/api/v1/tools/prophet_forecast`
    Expected Result: Tool successfully discovered, verified, adapted, registered
    Failure Indicators: No acknowledgement, workflow failure, missing tool
    Evidence: .sisyphus/evidence/task‑9‑chat‑suggestion.txt

  Scenario: Suggestion workflow provides progress updates
    Tool: Bash (curl)
    Preconditions: Workflow in progress
    Steps:
      1. Check workflow status: `curl http://localhost:8080/api/v1/tools/suggestion/{id}/status`
      2. Parse status response for phase details
    Expected Result: Status shows current phase (searching, verifying, adapting)
    Failure Indicators: Missing status endpoint, unclear phase information
    Evidence: .sisyphus/evidence/task‑9‑workflow‑status.json
  ```

  **Evidence to Capture**:
  - [ ] Chat interaction success
  - [ ] Workflow progress tracking

  **Commit**: YES
   - Message: `feat(workflow): add user‑initiated tool suggestion workflow`
   - Files: `internal/tool/suggestion/workflow.go`, `internal/api/handler/suggestion.go`
   - Pre‑commit: `go test ./internal/tool/suggestion/...`

---

- [ ] 10. Adaptation pipeline (verification → sandbox → registry)

  **What to do**:
  - Implement tool adaptation pipeline for discovered tools:
    1. **Verification**: Sandbox test execution (Task 3) with test inputs
    2. **Analysis**: Code structure analysis (imports, dependencies, API patterns)
    3. **Adaptation**: Code transformation for Aleph context:
       - Wrap with Aleph tool signature
       - Add error handling, logging
       - Resolve dependency conflicts
       - Adjust for sandbox environment
    4. **Testing**: Post‑adaptation verification in sandbox
    5. **Registration**: Add to tool registry with adapted code
  - Create adaptation templates for common patterns:
    - Python tool → Go wrapper
    - MCP tool → Aleph proxy
    - Library function → standalone tool
  - Adaptation quality metrics: Success rate, performance impact

  **Must NOT do**:
  - Modify original source code (maintain provenance)
  - Create adaptations that break original functionality
  - Skip post‑adaptation testing

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Code analysis and transformation requires sophisticated AST manipulation
  - **Skills**: [`golang‑pro`, `python‑pro`, `code‑translator`]
    - `golang‑pro`: Go adaptation logic
    - `python‑pro`: Python code analysis
    - `code‑translator`: Code transformation expertise

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5‑9)
  - **Blocks**: Tasks 11‑12 (tool creation uses adaptation patterns)
  - **Blocked By**: Tasks 1‑4, 8 (metadata, sandbox, discovery)

  **References**:

  **Pattern References**:
  - Existing code execution patterns
  - Error handling patterns in Aleph tools

  **API/Type References**:
  - Tool code structure definitions
  - Adaptation template formats

  **Test References**:
  - Code transformation tests
  - Adaptation verification tests

  **External References**:
  - AST parsing libraries for Go/Python
  - Code generation patterns

  **WHY Each Reference Matters**:
  - Need understanding of Aleph tool signatures for correct wrapping
  - Code transformation requires AST manipulation expertise

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Adaptation pipeline tests
  - [ ] Code transformation tests

  **QA Scenarios**:

  ```
  Scenario: Adaptation pipeline processes discovered tool
    Tool: Bash (curl)
    Preconditions: MCP tool discovered (Task 8), sandbox verification working
    Steps:
      1. Trigger adaptation: `curl -X POST http://localhost:8080/api/v1/tools/adaptation/process -d '{"tool_name":"external_tool", "source_code":"..."}'`
      2. Monitor adaptation phases via API
      3. Verify adapted tool registered
    Expected Result: Adapted tool successfully registered and executable
    Failure Indicators: Adaptation failure, malformed output, execution errors
    Evidence: .sisyphus/evidence/task‑10‑adaptation‑pipeline.json

  Scenario: Adapted tool maintains original functionality
    Tool: Bash (curl)
    Preconditions: Adapted tool registered
    Steps:
      1. Execute adapted tool with test inputs
      2. Compare outputs with expected behavior (if test oracle available)
    Expected Result: Adapted tool produces equivalent results to original
    Failure Indicators: Different results, new errors, performance regression
    Evidence: .sisyphus/evidence/task‑10‑functionality‑preservation.json
  ```

  **Evidence to Capture**:
  - [ ] Adaptation pipeline execution
  - [ ] Adapted tool functionality verification

  **Commit**: YES
   - Message: `feat(adaptation): add tool adaptation pipeline`
   - Files: `internal/tool/adaptation/pipeline.go`, `internal/tool/adaptation/templates.go`
   - Pre‑commit: `go test ./internal/tool/adaptation/...`

---

**WAVE 3 TASKS (Self‑Creation + Advanced)**

- [ ] 11. Tool creation DSL (OpenCode‑like templates)

  **What to do**:
  - Extend Aleph DSL for tool creation:
    1. **Tool definition syntax**: `.aleph` extension for tool specifications
    2. **Template system**: Pre‑built templates for common tool patterns
    3. **Code generation**: DSL → Go/Python implementation code
    4. **Validation**: Syntax checking, dependency resolution
    5. **Documentation generation**: Auto‑generate tool documentation
  - DSL features:
    - Tool metadata (name, description, category)
    - Parameter definitions (name, type, default, validation)
    - Return type specifications
    - Implementation blocks (Go/Python code snippets)
    - Test cases specification
    - Dependency declarations
  - Integration with sandbox: Generated code automatically tested

  **Must NOT do**:
  - Create DSL that replaces existing `.aleph` ontology syntax
  - Generate unsafe code without sandbox testing
  - Skip validation of generated code

  **Recommended Agent Profile**:
  - **Category**: `artistry`
    - Reason: DSL design requires creative language design and template engineering
  - **Skills**: [`golang‑pro`, `python‑pro`, `architecture‑designer`]
    - `golang‑pro`: DSL parser implementation
    - `python‑pro`: Template system for Python tools
    - `architecture‑designer`: Clean DSL language design

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 12‑15)
  - **Blocks**: Tasks 12 (sandbox enhancements for creation), 13‑14 (auto‑repair uses DSL)
  - **Blocked By**: Tasks 1‑10 (foundation + adaptation patterns)

  **References**:

  **Pattern References**:
  - Existing `.aleph` DSL parser in Aleph
  - Code generation patterns

  **API/Type References**:
  - Tool DSL grammar definition
  - Template file formats

  **Test References**:
  - DSL parsing tests
  - Code generation verification tests

  **External References**:
  - Parser generator libraries (`goyacc`, `ANTLR`)
  - Template engine patterns

  **WHY Each Reference Matters**:
  - Must extend existing Aleph DSL parser system
  - Code generation requires understanding of Aleph tool patterns

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] DSL parser tests
  - [ ] Code generation tests

  **QA Scenarios**:

  ```
  Scenario: Tool DSL parses and generates working tool
    Tool: Bash (curl + sandbox)
    Preconditions: DSL parser available, sandbox verification
    Steps:
      1. Submit tool DSL definition via API
      2. Verify parsing success
      3. Trigger code generation
      4. Execute generated tool in sandbox
    Expected Result: Generated tool executes successfully with test inputs
    Failure Indicators: Parse error, generation failure, execution error
    Evidence: .sisyphus/evidence/task‑11‑dsl‑generation.json

  Scenario: Tool DSL includes test cases that pass
    Tool: Bash (curl)
    Preconditions: Tool DSL with test cases specified
    Steps:
      1. Generate tool from DSL
      2. Run specified test cases in sandbox
      3. Verify all tests pass
    Expected Result: All DSL‑specified test cases pass for generated tool
    Failure Indicators: Test failures, missing test execution
    Evidence: .sisyphus/evidence/task‑11‑dsl‑tests.json
  ```

  **Evidence to Capture**:
  - [ ] DSL parsing success
  - [ ] Generated tool execution success
  - [ ] Test case verification

  **Commit**: YES (groups with Task 12)
   - Message: `feat(dsl): add tool creation DSL with code generation`
   - Files: `dsl/tool_parser.go`, `dsl/tool_templates/`, `internal/tool/creation/generator.go`
   - Pre‑commit: `go test ./dsl/... ./internal/tool/creation/...`

---

”— [ ] 12. Sandbox enhancements for test‑driven creation

  **What to do**:
  - Enhance sandbox for tool creation workflow:
    1. **Interactive development**: Step‑by‑step execution debugging
    2. **Test‑driven scaffolding**: Generate test suite from DSL, run tests during creation
    3. **Dependency mocking**: Mock external dependencies for isolated testing
    4. **Performance profiling**: Execution time, memory usage tracking
    5. **Security scanning**: Static analysis for security vulnerabilities
    6. **Code quality metrics**: Complexity, style checking
  - Integration with Task 11 DSL: Sandbox automatically tests generated code
  - Developer feedback: Real‑time execution results, error highlighting, suggestions

  **Must NOT do**:
  - Weaken existing sandbox security isolation
  - Allow network access during creation testing (unless mock)
  - Skip security scanning for generated code

  **Recommended Agent Profile**:
  - **Category**: `unspecified‑high`
    - Reason: Sandbox enhancements for creation require sophisticated execution environment
  - **Skills**: [`golang‑pro`, `python‑pro`, `secure‑code‑guardian`]
    - `golang‑pro`: Sandbox execution enhancements
    - `python‑pro`: Python sandbox improvements
    - `secure‑code‑guardian`: Maintain security during enhanced capabilities

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11, 13‑15)
  - **Blocks**: Task 13‑14 (auto‑diagnostic/repair use enhanced sandbox)
  - **Blocked By**: Task 3 (basic sandbox verification), Task 11 (DSL)

  **References**:

  **Pattern References**:
  - Existing sandbox execution (Task 3)
  - Debugging patterns in Aleph

  **API/Type References**:
  - Enhanced sandbox request/response for creation
  - Test result aggregation formats

  **Test References**:
  - Sandbox enhancement tests
  - Security scanning verification

  **External References**:
  - Code analysis libraries (`gosec`, `bandit`)
  - Profiling tools integration

  **WHY Each Reference Matters**:
  - Builds directly on Task 3 sandbox verification
  - Requires understanding of security isolation to maintain safety

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Sandbox enhancement tests
  - [ ] Security scanning integration tests

  **QA Scenarios**:

  ```
  Scenario: Enhanced sandbox supports interactive development
    Tool: Bash (curl)
    Preconditions: Sandbox enhancements deployed
    Steps:
      1. Submit partial tool code for stepwise execution
      2. Receive execution results with variable state
      3. Submit next code increment
      4. Verify cumulative execution
    Expected Result: Interactive execution maintains state between steps
    Failure Indicators: State loss, execution errors, security violations
    Evidence: .sisyphus/evidence/task‑12‑interactive‑sandbox.json

  Scenario: Sandbox performs security scanning on generated code
    Tool: Bash (curl)
    Preconditions: Security scanning enabled
    Steps:
      1. Submit tool code with potential vulnerability
      2. Receive security scan results
      3. Verify vulnerability detection
    Expected Result: Security scan identifies vulnerability with explanation
    Failure Indicators: Missed vulnerability, false positives
    Evidence: .sisyphus/evidence/task‑12‑security‑scan.json
  ```

  **Evidence to Capture**:
  - [ ] Interactive development functionality
  - [ ] Security scanning results

  **Commit**: YES (groups with Task 11)
   - Message: `feat(sandbox): enhance for test‑driven tool creation`
   - Files: `internal/sandbox/creation_mode.go`, `internal/sandbox/security_scanner.go`
   - Pre‑commit: `go test ./internal/sandbox/...`

---

- [ ] 13. Auto‑diagnostic subsystem (error classification)

  **What to do**:
  - Implement automatic tool diagnostics:
    1. **Error monitoring**: Capture tool execution errors, logs, metrics
    2. **Pattern classification**: Classify errors into categories:
       - Syntax errors (compile‑time)
       - Runtime exceptions (null pointer, index out of bounds)
       - Dependency errors (missing imports, version conflicts)
       - Performance issues (timeouts, memory leaks)
       - Security violations
       - Logical errors (incorrect outputs)
    3. **Root cause analysis**: Trace errors to specific code lines, dependencies
    4. **Severity assessment**: Impact on functionality, frequency
    5. **History tracking**: Error trends over time, recurrence patterns
  - Integration with health checks (Task 4): Diagnostic data feeds health status
  - Alert system: Notify users of critical issues

  **Must NOT do**:
  - Automatically disable tools based on diagnostics
  - Share diagnostic data externally without user consent
  - Create excessive monitoring overhead

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Error analysis requires advanced pattern recognition and classification
  - **Skills**: [`golang‑pro`, `python‑pro`, `debugging‑wizard`]
    - `golang‑pro`: Error capture and analysis
    - `python‑pro`: Python error pattern recognition
    - `debugging‑wizard`: Systematic debugging expertise

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11‑12, 14‑15)
  - **Blocks**: Task 14 (auto‑repair uses diagnostics)
  - **Blocked By**: Tasks 1‑4 (metadata, health), Tasks 3, 12 (sandbox)

  **References**:

  **Pattern References**:
  - Error handling patterns in Aleph
  - Log collection patterns

  **API/Type References**:
  - Error classification taxonomy
  - Diagnostic report formats

  **Test References**:
  - Error injection and detection tests
  - Classification accuracy tests

  **External References**:
  - Error pattern libraries/common classifications
  - Root cause analysis methodologies

  **WHY Each Reference Matters**:
  - Must integrate with existing error handling in Aleph
  - Classification taxonomy design requires expertise

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Diagnostic subsystem tests
  - [ ] Error classification accuracy tests

  **QA Scenarios**:

  ```
  Scenario: Auto‑diagnostic detects and classifies tool error
    Tool: Bash (curl)
    Preconditions: Diagnostic subsystem active, test tool with known error
    Steps:
      1. Execute faulty tool
      2. Query diagnostics: `curl http://localhost:8080/api/v1/tools/faulty/diagnostics`
      3. Verify error classification
    Expected Result: Diagnostic report contains correct error category and root cause
    Failure Indicators: Missing diagnostics, incorrect classification
    Evidence: .sisyphus/evidence/task‑13‑diagnostic‑detection.json

  Scenario: Error severity assessment reflects impact
    Tool: Bash (curl)
    Preconditions: Multiple tool errors of varying severity
    Steps:
      1. Query diagnostic dashboard: `curl http://localhost:8080/api/v1/tools/diagnostics/dashboard`
      2. Verify severity rankings
    Expected Result: Critical errors ranked higher than warnings
    Failure Indicators: Incorrect severity assignment, missing errors
    Evidence: .sisyphus/evidence/task‑13‑severity‑assessment.json
  ```

  **Evidence to Capture**:
  - [ ] Error detection and classification
  - [ ] Severity assessment accuracy

  **Commit**: YES (groups with Task 14)
   - Message: `feat(diagnostics): add auto‑diagnostic subsystem for tools`
   - Files: `internal/tool/diagnostics/classifier.go`, `internal/tool/diagnostics/monitor.go`
   - Pre‑commit: `go test ./internal/tool/diagnostics/...`

---

- [ ] 14. Auto‑repair strategies (patterns, regeneration)

  **What to do**:
  - Implement automated repair strategies for diagnosed issues:
    1. **Repair catalog**: Pre‑defined fixes for common error patterns:
       - Missing imports → add import statements
       - Syntax errors → auto‑correct with language server
       - Deprecated APIs → update to current versions
       - Configuration errors → fix configuration
       - Performance issues → optimize code patterns
    2. **Regeneration**: For severe issues, regenerate tool from DSL/Template
    3. **User approval workflow**: Present repair plan, get user confirmation
    4. **Repair execution**: Apply fixes in sandbox, verify, deploy
    5. **Repair history**: Track repair attempts, success rates
  - Integration with diagnostics (Task 13): Repair triggered by specific error classifications
  - Integration with DSL (Task 11): Regeneration uses tool DSL definitions
  - Safety checks: Backup original tool, rollback on repair failure

  **Must NOT do**:
  - Apply repairs without user approval
  - Modify user data during repair
  - Skip verification of repaired tool

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Auto‑repair requires sophisticated code transformation and verification
  - **Skills**: [`golang‑pro`, `python‑pro`, `code‑translator`]
    - `golang‑pro`: Repair strategy implementation
    - `python‑pro`: Python‑specific repairs
    - `code‑translator`: Code transformation for repairs

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11‑13, 15)
  - **Blocks**: None (completes auto‑repair capability)
  - **Blocked By**: Tasks 1‑13 (everything prior)

  **References**:

  **Pattern References**:
  - Code transformation patterns (Task 10 adaptation)
  - DSL regeneration patterns (Task 11)

  **API/Type References**:
  - Repair plan format
  - User approval workflow API

  **Test References**:
  - Repair strategy tests
  - Verification tests for repaired tools

  **External References**:
  - Auto‑fix patterns from IDEs/language servers
  - Code regeneration methodologies

  **WHY Each Reference Matters**:
  - Builds on adaptation pipeline (Task 10) for code transformation
  - Uses DSL system (Task 11) for regeneration

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Repair strategy tests
  - [ ] Repair verification tests

  **QA Scenarios**:

  ```
  Scenario: Auto‑repair proposes fix for diagnosed issue
    Tool: Bash (curl)
    Preconditions: Tool with known error, diagnostics active
    Steps:
      1. Trigger repair analysis: `curl -X POST http://localhost:8080/api/v1/tools/faulty/repair/analyze`
      2. Review repair plan proposal
      3. Approve repair (simulated)
      4. Execute repair
      5. Verify tool functionality restored
    Expected Result: Repair plan proposed, executed, tool functionality restored
    Failure Indicators: No repair plan, repair failure, functionality not restored
    Evidence: .sisyphus/evidence/task‑14‑auto‑repair.json

  Scenario: Repair rollback on failure
    Tool: Bash (curl)
    Preconditions: Tool backup system active
    Steps:
      1. Attempt flawed repair that will fail
      2. Verify rollback occurs
      3. Check original tool still exists
    Expected Result: Failed repair triggers rollback, original tool preserved
    Failure Indicators: No rollback, tool corrupted, data loss
    Evidence: .sisyphus/evidence/task‑14‑repair‑rollback.json
  ```

  **Evidence to Capture**:
  - [ ] Repair proposal and execution
  - [ ] Rollback functionality

  **Commit**: YES (groups with Task 13)
   - Message: `feat(repair): add auto‑repair strategies for tool maintenance`
   - Files: `internal/tool/repair/strategies.go`, `internal/tool/repair/executor.go`
   - Pre‑commit: `go test ./internal/tool/repair/...`

---

- [ ] 15. Integration with CodeFlow/HumanEcosystems/Shadowbroker concepts

  **What to do**:
  - Integrate extended tool system with three planning documents:
    1. **CodeFlow.md integration**: Visualization engine for tool execution flows
       - Tool execution graphs, dependency visualization
       - Performance metrics visualization
       - Privacy‑first visualization generation
    2. **HumanEcosystems.md integration**: Relational context for tool usage
       - Tool usage patterns analysis
       - Cross‑tool relationship mapping
       - Geographic context for tool applicability
    3. **Shadowbroker.md integration**: OSINT intelligence for tool discovery
       - Threat intelligence for security tools
       - Market analysis for financial tools
       - Geospatial context for location‑based tools
  - Create cross‑document synthesis layer:
    - Unified tool intelligence dashboard
    - Context‑aware tool recommendations
    - Risk assessment for tool adoption

  **Must NOT do**:
  - Implement full planning documents (scope is integration only)
  - Create duplicate functionality from planning documents
  - Violate privacy/security principles of source documents

  **Recommended Agent Profile**:
  - **Category**: `visual‑engineering`
    - Reason: Integration requires UI/visualization design and cross‑system synthesis
  - **Skills**: [`react‑expert`, `architecture‑designer`, `api‑designer`]
    - `react‑expert`: Dashboard UI implementation
    - `architecture‑designer`: Cross‑system integration design
    - `api‑designer`: Clean integration API design

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11‑14)
  - **Blocks**: None (final integration task)
  - **Blocked By**: Tasks 5‑7 (domain packages), Tasks 1‑14 (all prior)

  **References**:

  **Pattern References**:
  - Existing visualization patterns in Aleph (if any)
  - Dashboard UI patterns

  **API/Type References**:
  - Tool intelligence data structures
  - Integration API endpoints

  **Test References**:
  - Integration verification tests
  - Dashboard functionality tests

  **External References**:
  - D3.js or similar visualization libraries
  - Dashboard design patterns

  **WHY Each Reference Matters**:
  - Planning documents provide specific requirements for integration
  - Need to maintain privacy/security principles from source documents

  **Acceptance Criteria**:

  **If TDD**:
  - [ ] Integration API tests
  - [ ] Dashboard component tests

  **QA Scenarios**:

  ```
  Scenario: Tool intelligence dashboard displays cross‑document insights
    Tool: Playwright (browser automation)
    Preconditions: Integration implemented, tools with metadata
    Steps:
      1. Navigate to tool intelligence dashboard
      2. Verify visualization components (execution graphs, relationship maps)
      3. Verify context‑aware recommendations
    Expected Result: Dashboard displays integrated insights from all three planning documents
    Failure Indicators: Missing visualizations, incorrect data, privacy violations
    Evidence: .sisyphus/evidence/task‑15‑dashboard‑screenshots.png

  Scenario: OSINT intelligence informs tool security assessment
    Tool: Bash (curl)
    Preconditions: Shadowbroker integration active
    Steps:
      1. Request security assessment for tool: `curl http://localhost:8080/api/v1/tools/test/security‑assessment`
      2. Verify OSINT‑based threat intelligence included
    Expected Result: Assessment includes OSINT‑derived threat context
    Failure Indicators: Missing OSINT data, generic assessment only
    Evidence: .sisyphus/evidence/task‑15‑osint‑assessment.json
  ```

  **Evidence to Capture**:
  - [ ] Dashboard visualization screenshots
  - [ ] Integrated intelligence API responses

  **Commit**: YES
   - Message: `feat(integration): unify tool system with CodeFlow/HE/Shadowbroker`
   - Files: `internal/tool/intelligence/dashboard.go`, `ui/components/ToolIntelligenceDashboard.tsx`
   - Pre‑commit: `go test ./internal/tool/intelligence/...` and frontend tests

---

—

## Final Verification Wave

- [ ] F1. **End‑to‑end chat‑based tool lifecycle** — `unspecified‑high`
  Simulate user saying "Aleph, add prophet forecasting tool" → verify MCP discovery → adaptation → sandbox test → registry addition → health check. Capture chat interaction evidence.
  Output: `Discovery [PASS/FAIL] | Adaptation [PASS/FAIL] | Registration [PASS/FAIL] | VERDICT`

-layout**Entire MCP ecosystem connectivity test** — `unspecified‑high`
  Test connections to: OpenBB MCP server, Great Expectations MCP server, Ghidra MCP community servers. Validate tool schemas, resource URIs, prompt templates. Network isolation check.
  Output: `OpenBB [PASS/FAIL] | GX [PASS/FAIL] | Ghidra [PASS/FAIL] | Network [CLEAN/ISSUES] | VERDICT`

- [ ] F3. **Self‑repair demonstration** — `unspecified‑high`
  Intentionally break a tool (malformed code, missing imports), trigger auto‑diagnostic, simulate repair strategies (regeneration, template application). Verify restored functionality.
  Output: `Break Detection [PASS/FAIL] | Repair Strategy [VALID/INVALID] | Restoration [PASS/FAIL] | VERDICT`

- [ ] F4. **Cross‑context adaptability verification** — `oracle`
  Deploy Aleph in 3 contexts: finance analysis (prophet + market data), OSINT intelligence (shadowbroker), social research (human‑ecosystems). Verify tool packages work coherently, no conflicts.
  Output: `Contexts [3/3 operational] | Conflicts [CLEAN/N issues] | Coherence [PASS/FAIL] | VERDICT`

---

## Commit Strategy

- **Wave 1**: `feat(tools): extend metadata with categories, health checks`
- **Wave 2**: `feat(packages): add finance, OSINT, human‑ecosystems tool packages`
. **Wave 3**: `feat(creation): add tool creation DSL and auto‑repair subsystem`

---

## Success Criteria

### Verification Commands
```bash
curl -X POST http://localhost:8080/api/v1/tool/suggest -d '{"name":"prophet_forecast"}'  # Expected: 202 Accepted, discovery agent launched
aleph chat "Aleph, add prophet forecasting tool"  # Expected: "I'll search for prophet forecasting tools..."
go test ./internal/tool/discovery/...  # Expected: PASS
```

### Final Checklist
1. [ ] Existing tools (`analyze_sentiment`, `get_trust_score`, `search_data`) still work
2. [ ] Chat‑based `/tool` commands operational
3. [ ] Domain tool packages installable via `/tool install {package}`
4. [ ] MCP discovery locates OpenBB/GX/Ghidra tools
5. [ ] User can suggest tool → verified → adapted → registered
6. [ ] Developer can create tool via DSL → sandbox tested → deployed
7. [ ] Auto‑diagnostic detects broken tools
8. [ ] Auto‑repair can fix common issues (with user approval)
9. [ ] Cross‑context adaptability demonstrated