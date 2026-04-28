# Aleph-v2 API Reference

## REST Endpoints

Most REST endpoints require authentication via `X-Aleph-Api-Key` header. The health endpoint is the only unauthenticated route.

### Health & Diagnostics

#### `GET /api/v1/healthz` (no auth required)
Health check endpoint for Docker HEALTHCHECK, load balancers, and monitoring.

**Response:** `200 OK`
```json
{"status":"ok"}
```

---

#### `GET /api/v1/tools/health`
Returns health status for all registered tools.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

**Response:**
```json
{
  "tools": [
    {
      "id": "tool-id",
      "status": "healthy",
      "last_check": "2026-04-27T10:00:00Z"
    }
  ]
}
```

---

#### `POST /api/v1/tools/verify`
Verify tool connectivity and functionality.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`
- `Content-Type: application/json`

**Request:**
```json
{
  "tool_id": "tool-id"
}
```

**Response:**
```json
{
  "verified": true,
  "latency_ms": 42
}
```

---

#### `GET /api/v1/tools/{id}/health/history`
Retrieve health check history for a specific tool.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

**Response:**
```json
{
  "tool_id": "tool-id",
  "history": [
    {
      "timestamp": "2026-04-27T10:00:00Z",
      "status": "healthy",
      "latency_ms": 42
    }
  ]
}
```

---

#### `GET /api/v1/diagnostic/patterns`
Retrieve diagnostic patterns from the monitoring system.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

**Response:**
```json
{
  "patterns": [
    {
      "name": "pattern-name",
      "description": "Pattern description",
      "severity": "warning"
    }
  ]
}
```

---

### Tool Management

#### `GET /api/v1/tools`
List all registered tools.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

---

#### `GET /api/v1/tools/categories`
List available tool categories.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

---

#### `POST /api/v1/tools/execute/{category}/{name}`
Execute a tool by category and name.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`
- `Content-Type: application/json`

---

#### `POST /api/v1/tools/call`
Call a tool with parameters.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`
- `Content-Type: application/json`

---

#### `POST /api/v1/tools/register`
Register a new tool in the system.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`
- `Content-Type: application/json`

---

#### `POST /api/v1/tools/suggest`
Submit a tool suggestion for review.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`
- `Content-Type: application/json`

---

#### `POST /api/v1/tools/suggest/approve`
Approve a suggested tool.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`
- `Content-Type: application/json`

---

### CodeFlow

#### `GET /api/v1/codeflow/graph`
Retrieve code dependency graph.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

---

#### `GET /api/v1/codeflow/metrics`
Get code metrics and statistics.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

---

#### `GET /api/v1/codeflow/executions`
List code execution history.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

---

#### `GET /api/v1/codeflow/engines`
List available code execution engines.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

---

### SSE Streaming

#### `GET /api/v1/events`
Server-Sent Events endpoint for real-time updates.

**Headers:**
- `X-Aleph-Api-Key: <api-key>`

**Response:** SSE stream with events

---

## ConnectRPC Services

ConnectRPC services use HTTP/2 and protobuf serialization. Authentication via `X-Aleph-Api-Key` header.

### AgentService

**Prefix:** `/aleph.v1.AgentService/`

5 methods — backed by `handler.AgentHandler` (`internal/api/handler/agent.go`).

| Method | Description |
|--------|-------------|
| `ListAgents` | List all agents for a project |
| `CreateAgent` | Create a new agent (with provider, model, system prompt, skill IDs) |
| `GetAgent` | Get agent by ID |
| `UpdateAgent` | Update agent configuration (masked API key in response) |
| `DeleteAgent` | Remove an agent |
| `ListModels` | List available Ollama models from the configured Ollama host |

---

### SkillService

**Prefix:** `/aleph.v1.SkillService/`

3 methods — backed by `handler.SkillHandler` (`internal/api/handler/skill.go`).

| Method | Description |
|--------|-------------|
| `ListSkills` | List all skills for a project (with tool IDs) |
| `CreateSkill` | Register a new skill |
| `DeleteSkill` | Remove a skill |

---

### ToolService

**Prefix:** `/aleph.v1.ToolService/`

3 ConnectRPC methods — backed by `handler.ToolHandler` (`internal/api/handler/tool.go`).

| Method | Description |
|--------|-------------|
| `ListTools` | List all registered tools (name, description, code) |
| `CreateTool` | Register a new tool |
| `DeleteTool` | Remove a tool |

The same `ToolHandler` also exposes **raw HTTP endpoints** — see [Tool Management REST endpoints](#tool-management) above.

---

### LibraryService

**Prefix:** `/aleph.v1.LibraryService/`

5 methods — backed by `handler.LibraryHandler` (`internal/api/handler/library.go`).

| Method | Description |
|--------|-------------|
| `ListAssets` | List all assets in a project's library directory |
| `GetAssetContent` | Read raw content of an asset |
| `UploadAsset` | Upload a new asset (path-sanitized) |
| `DeleteAsset` | Remove an asset |
| `GeneratePdf` | Generate a PDF from an asset's content |

---

### ProjectService

**Prefix:** `/aleph.v1.ProjectService/`

6 methods — backed by `handler.ProjectHandler` (`internal/api/handler/project.go`).

| Method | Description |
|--------|-------------|
| `ListProjects` | List all projects (reads `data/projects/` subdirectories) |
| `CreateProject` | Create project directory structure (`raw/`, `ontologies/`, `agents/`, `skills/`) |
| `DeleteProject` | Remove a project directory tree |
| `GetOntology` | Read `core.aleph` and parse object names; falls back to DuckDB `information_schema` |
| `SaveOntology` | Atomic write with backup (`core.aleph.<timestamp>.bak`) |
| `EmergeOntology` | Auto-generate ontology definition from DuckDB schema |

---

### NotificationService

**Prefix:** `/aleph.v1.NotificationService/`

2 methods — backed by `handler.NotificationHandler` (`internal/api/handler/notification.go`).

| Method | Description |
|--------|-------------|
| `SendWebhook` | Send a webhook notification (with optional secret) |
| `ListChannels` | List notification channels for a project |

---

### AuthService

**Prefix:** `/aleph.v1.AuthService/`

3 methods — backed by `handler.AuthHandler` (`internal/api/handler/auth.go`).

| Method | Description |
|--------|-------------|
| `ListApiKeys` | List API keys (keys masked as `********` in responses) |
| `CreateApiKey` | Generate a new API key (SHA-256 hashed for storage, returned once in plaintext) |
| `DeleteApiKey` | Revoke an API key |

---

### IngestionService

**Prefix:** `/aleph.v1.IngestionService/`

6 methods — backed by `handler.IngestionHandler` (`internal/api/handler/ingestion.go`).

| Method | Description |
|--------|-------------|
| `ListTasks` | List ingestion tasks for a project |
| `CreateTask` | Create a new ingestion task (auto-generates ID if empty) |
| `GetProgress` | Get task progress percentage |
| `RunTask` | Start a task asynchronously (15-minute context timeout) |
| `GetTaskLogs` | Read task log file |
| `DeleteTask` | Remove a task |

---

### SandboxService

**Prefix:** `/aleph.v1.SandboxService/`

2 methods — backed by `handler.SandboxServiceHandler` (`internal/api/handler/sandbox_handler.go`).

| Method | Description |
|--------|-------------|
| `ExecuteTool` | Execute a tool inside the sandbox |
| `RunSkill` | Run a skill orchestration workflow |

---

### RegistryService

**Prefix:** `/aleph.v1.RegistryService/`

4 methods — backed by `handler.RegistryServiceHandler` (`internal/api/handler/registry_handler.go`).

| Method | Description |
|--------|-------------|
| `RegisterComponent` | Register a new component |
| `ListComponents` | List registered components |
| `GetComponent` | Get component details by ID |
| `UpdateComponentStatus` | Update component status |

---

### NLPService

**Prefix:** `/aleph.nlp.v1.NLPService/`

3 methods — backed by `handler.NLPHandler` (`internal/api/handler/nlp.go`). Delegates to a Python sidecar via circuit breaker.

| Method | Description |
|--------|-------------|
| `AnalyzeSentiment` | Perform NLP sentiment analysis |
| `StreamPredictions` | Stream predictions via server-sent streaming |
| `RecordFeedback` | Record feedback and trigger Brier score evaluation |

---

### QueryService

**Prefix:** `/aleph.v1.QueryService/`

8 methods — backed by `handler.QueryHandler` (`internal/api/handler/query.go`).

| Method | Description |
|--------|-------------|
| `ExecuteQuery` | Execute a DuckDB query against a project's database |
| `Chat` | Streaming chat with AI agent (SSE, supports tool calls) |
| `GetChatHistory` | Retrieve chat history for a project |
| `GetDataStats` | Get column-level statistics for a dataset |
| `ConfirmAction` | Confirm or reject a pending action |
| `GlobalQuery` | Cross-project query across all databases |
| `GetDataLineage` | Get data lineage information |
| `GetChecksum` | Compute SHA-256 checksum for a table |

---

## Authentication

All API requests require authentication via the `X-Aleph-Api-Key` header, with one exception:

- **`GET /api/v1/healthz`** — no authentication required (for Docker HEALTHCHECK and load balancers)

**REST endpoints:**
```
X-Aleph-Api-Key: <your-api-key>
```

**ConnectRPC:**
```
X-Aleph-Api-Key: <your-api-key>
```

API keys are managed through the AuthService and stored in the metadata repository (SHA-256 hashed).

---

## Error Responses

### REST Errors

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable description",
    "details": {}
  }
}
```

### ConnectRPC Errors

Standard gRPC status codes with rich error details in protobuf format.

---

## Rate Limiting

Rate limits are applied per API key. Limits are configured via environment variables.

---

## CORS

CORS is configured via the `CORS_ALLOWED_ORIGINS` environment variable. Default:
```
http://localhost:5173,http://localhost:3000
```

---

## See Also

- [`docs/CONTRIBUTING.md`](./CONTRIBUTING.md) — Development guide
- [`docs/CHANGELOG.md`](./CHANGELOG.md) — Release history
- [`AGENTS.md`](../AGENTS.md) — Agent system documentation
