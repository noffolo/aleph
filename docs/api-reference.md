# Aleph-v2 API Reference

> **Version:** 2.0.0 · **Last updated:** April 2026 · **Status:** Production

This document describes all public API endpoints for Aleph-v2. The API consists of three protocol layers:

- **ConnectRPC** (primary) — HTTP/2, protobuf, type-safe clients
- **REST** (legacy/tool execution) — HTTP/1, JSON, simple curl access
- **SSE** (streaming) — Server-Sent Events for real-time chat and notifications

All authenticated endpoints require the `X-Aleph-Api-Key` header. The only unauthenticated endpoints are `/api/v1/healthz`, `/readyz`, `/livez`, and `/metrics`.

---

## Authentication

```
X-Aleph-Api-Key: <your-api-key>
```

API keys are managed through the AuthService. They are stored as SHA-256 hashes, never in plaintext. At-rest encryption uses AES-256-GCM with `KEY_ENCRYPTION_KEY`.

---

## ConnectRPC Services

ConnectRPC services use protobuf serialization over HTTP/2. Each service has a path prefix like `/aleph.v1.AgentService/`.

### AgentService

**Prefix:** `/aleph.v1.AgentService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `ListAgents` | `ListAgentsRequest{project_id}` | `ListAgentsResponse{agents[]}` | List all agents for a project |
| `CreateAgent` | `CreateAgentRequest{name, provider, model, system_prompt, skill_ids[]}` | `Agent{id, name, provider, model, system_prompt, skill_ids[], created_at}` | Create a new agent |
| `GetAgent` | `GetAgentRequest{project_id, agent_id}` | `Agent` | Get agent by ID |
| `UpdateAgent` | `UpdateAgentRequest{id, name, provider, model, system_prompt, skill_ids[]}` | `Agent` | Update agent configuration |
| `DeleteAgent` | `DeleteAgentRequest{project_id, agent_id}` | `DeleteAgentResponse{}` | Remove an agent |
| `ListModels` | `ListModelsRequest{}` | `ListModelsResponse{models[]}` | List available Ollama models |

**Example — CreateAgent:**
```protobuf
// Request
{
  "project_id": "proj_abc123",
  "name": "Analyst",
  "provider": "ollama",
  "model": "llama3",
  "system_prompt": "You are a financial analyst.",
  "skill_ids": ["skill_sentiment", "skill_query"]
}

// Response
{
  "id": "agent_001",
  "name": "Analyst",
  "provider": "ollama",
  "model": "llama3",
  "system_prompt": "You are a financial analyst.",
  "skill_ids": ["skill_sentiment", "skill_query"],
  "created_at": "2026-04-27T10:00:00Z"
}
```

---

### SkillService

**Prefix:** `/aleph.v1.SkillService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `ListSkills` | `ListSkillsRequest{project_id}` | `ListSkillsResponse{skills[]}` | List all skills for a project |
| `CreateSkill` | `CreateSkillRequest{project_id, name, description, tool_ids[]}` | `Skill{id, name, description, tool_ids[], created_at}` | Register a new skill |
| `DeleteSkill` | `DeleteSkillRequest{project_id, skill_id}` | `DeleteSkillResponse{}` | Remove a skill |

**Example — ListSkills:**
```protobuf
// Request
{"project_id": "proj_abc123"}

// Response
{
  "skills": [
    {
      "id": "skill_sentiment",
      "name": "Sentiment Analysis",
      "description": "Analyze text sentiment using NLP",
      "tool_ids": ["tool_nlp_sentiment"],
      "created_at": "2026-04-27T10:00:00Z"
    }
  ]
}
```

---

### ToolService

**Prefix:** `/aleph.v1.ToolService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `ListTools` | `ListToolsRequest{project_id}` | `ListToolsResponse{tools[]}` | List all registered tools |
| `CreateTool` | `CreateToolRequest{project_id, name, description, code, category}` | `Tool{id, name, description, code, category, version, created_at}` | Register a new tool |
| `DeleteTool` | `DeleteToolRequest{project_id, tool_id}` | `DeleteToolResponse{}` | Remove a tool |

**Example — CreateTool:**
```protobuf
// Request
{
  "project_id": "proj_abc123",
  "name": "csv_ingester",
  "description": "Import CSV files into DuckDB",
  "code": "package main\nimport...",
  "category": "ingestion"
}

// Response
{
  "id": "tool_csv_001",
  "name": "csv_ingester",
  "description": "Import CSV files into DuckDB",
  "category": "ingestion",
  "version": "1.0.0",
  "created_at": "2026-04-27T10:00:00Z"
}
```

---

### LibraryService

**Prefix:** `/aleph.v1.LibraryService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `ListAssets` | `ListAssetsRequest{project_id}` | `ListAssetsResponse{assets[]}` | List all assets in a project's library |
| `GetAssetContent` | `GetAssetContentRequest{project_id, asset_id}` | `GetAssetContentResponse{content, mime_type}` | Read raw content of an asset |
| `UploadAsset` | `UploadAssetRequest{project_id, path, content, mime_type}` | `Asset{id, path, size, created_at}` | Upload a new asset (path-sanitized) |
| `DeleteAsset` | `DeleteAssetRequest{project_id, asset_id}` | `DeleteAssetResponse{}` | Remove an asset |
| `GeneratePdf` | `GeneratePdfRequest{project_id, asset_id}` | `GeneratePdfResponse{pdf_url}` | Generate a PDF from an asset |

**Example — UploadAsset:**
```protobuf
// Request
{
  "project_id": "proj_abc123",
  "path": "reports/q1_2026.md",
  "content": "# Q1 2026 Report\n\nRevenue increased...",
  "mime_type": "text/markdown"
}

// Response
{
  "id": "asset_001",
  "path": "reports/q1_2026.md",
  "size": 1240,
  "created_at": "2026-04-27T10:00:00Z"
}
```

---

### ProjectService

**Prefix:** `/aleph.v1.ProjectService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `ListProjects` | `ListProjectsRequest{}` | `ListProjectsResponse{projects[]}` | List all projects |
| `CreateProject` | `CreateProjectRequest{name, description}` | `Project{id, name, description, created_at}` | Create project with directory structure |
| `DeleteProject` | `DeleteProjectRequest{project_id}` | `DeleteProjectResponse{}` | Remove a project directory tree |
| `GetOntology` | `GetOntologyRequest{project_id}` | `GetOntologyResponse{objects[], source}` | Read ontology definition |
| `SaveOntology` | `SaveOntologyRequest{project_id, content}` | `SaveOntologyResponse{backup_path}` | Atomic write with backup |
| `EmergeOntology` | `EmergeOntologyRequest{project_id}` | `EmergeOntologyResponse{objects[]}` | Auto-generate from DuckDB schema |

**Example — CreateProject:**
```protobuf
// Request
{
  "name": "market-analysis",
  "description": "Market research project"
}

// Response
{
  "id": "proj_abc123",
  "name": "market-analysis",
  "description": "Market research project",
  "created_at": "2026-04-27T10:00:00Z"
}
```

---

### NotificationService

**Prefix:** `/aleph.v1.NotificationService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `SendWebhook` | `SendWebhookRequest{project_id, url, payload, secret}` | `SendWebhookResponse{success, status_code}` | Send a webhook notification |
| `ListChannels` | `ListChannelsRequest{project_id}` | `ListChannelsResponse{channels[]}` | List notification channels |

**Example — SendWebhook:**
```protobuf
// Request
{
  "project_id": "proj_abc123",
  "url": "https://hooks.example.com/aleph",
  "payload": {"event": "ingestion_complete", "task_id": "task_001"},
  "secret": "whsec_xxx"
}

// Response
{
  "success": true,
  "status_code": 200
}
```

---

### AuthService

**Prefix:** `/aleph.v1.AuthService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `ListApiKeys` | `ListApiKeysRequest{project_id}` | `ListApiKeysResponse{keys[]}` | List API keys (masked) |
| `CreateApiKey` | `CreateApiKeyRequest{project_id, name}` | `CreateApiKeyResponse{key, id, plaintext}` | Generate a new key (returned once) |
| `DeleteApiKey` | `DeleteApiKeyRequest{project_id, key_id}` | `DeleteApiKeyResponse{}` | Revoke an API key |

**Example — CreateApiKey:**
```protobuf
// Request
{
  "project_id": "proj_abc123",
  "name": "ci-deployment"
}

// Response
{
  "id": "key_001",
  "key": "********",
  "plaintext": "alp_abc123xyz789..."
}
```

---

### IngestionService

**Prefix:** `/aleph.v1.IngestionService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `ListTasks` | `ListTasksRequest{project_id}` | `ListTasksResponse{tasks[]}` | List ingestion tasks |
| `CreateTask` | `CreateTaskRequest{project_id, source_type, config}` | `IngestionTask{id, source_type, status, created_at}` | Create a new task |
| `GetProgress` | `GetProgressRequest{project_id, task_id}` | `GetProgressResponse{percentage, status}` | Get task progress |
| `RunTask` | `RunTaskRequest{project_id, task_id}` | `RunTaskResponse{accepted}` | Start a task asynchronously |
| `GetTaskLogs` | `GetTaskLogsRequest{project_id, task_id}` | `GetTaskLogsResponse{lines[]}` | Read task log file |
| `DeleteTask` | `DeleteTaskRequest{project_id, task_id}` | `DeleteTaskResponse{}` | Remove a task |

**Example — RunTask:**
```protobuf
// Request
{
  "project_id": "proj_abc123",
  "task_id": "task_001"
}

// Response
{
  "accepted": true
}
```

---

### SandboxService

**Prefix:** `/aleph.v1.SandboxService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `ExecuteTool` | `ExecuteToolRequest{project_id, tool_id, parameters{}}` | `ExecuteToolResponse{output, exit_code, duration_ms}` | Execute a tool inside the sandbox |
| `RunSkill` | `RunSkillRequest{project_id, skill_id, input}` | `RunSkillResponse{results[], completed}` | Run a skill orchestration workflow |

**Example — ExecuteTool:**
```protobuf
// Request
{
  "project_id": "proj_abc123",
  "tool_id": "tool_csv_001",
  "parameters": {
    "file_path": "data/raw/sales.csv",
    "table_name": "sales"
  }
}

// Response
{
  "output": "Imported 15420 rows into table 'sales'",
  "exit_code": 0,
  "duration_ms": 1240
}
```

---

### RegistryService

**Prefix:** `/aleph.v1.RegistryService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `RegisterComponent` | `RegisterComponentRequest{name, type, url, config}` | `Component{id, name, status}` | Register a new component |
| `ListComponents` | `ListComponentsRequest{}` | `ListComponentsResponse{components[]}` | List registered components |
| `GetComponent` | `GetComponentRequest{id}` | `Component` | Get component details |
| `UpdateComponentStatus` | `UpdateComponentStatusRequest{id, status}` | `UpdateComponentStatusResponse{}` | Update component status |

**Example — RegisterComponent:**
```protobuf
// Request
{
  "name": "market-data-feed",
  "type": "mcp",
  "url": "http://localhost:3000/sse",
  "config": {"timeout_ms": 5000}
}

// Response
{
  "id": "comp_001",
  "name": "market-data-feed",
  "status": "pending"
}
```

---

### NLPService (Sidecar Bridge)

**Prefix:** `/aleph.nlp.v1.NLPService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `AnalyzeSentiment` | `AnalyzeSentimentRequest{text}` | `AnalyzeSentimentResponse{score, label}` | Sentiment analysis |
| `StreamPredictions` | `StreamPredictionsRequest{query, horizon_days}` | stream `PredictionResponse{date, value, confidence}` | Ensemble Prophet+GBM streaming |
| `RecordFeedback` | `RecordFeedbackRequest{prediction_id, actual_value}` | `RecordFeedbackResponse{brier_score}` | Record feedback for calibration |

**Example — AnalyzeSentiment:**
```protobuf
// Request
{"text": "The market outlook is very positive this quarter."}

// Response
{
  "score": 0.72,
  "label": "positive"
}
```

---

### QueryService

**Prefix:** `/aleph.v1.QueryService/`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `ExecuteQuery` | `ExecuteQueryRequest{project_id, sql}` | `ExecuteQueryResponse{columns[], rows[]}` | Execute a DuckDB query |
| `Chat` | `ChatRequest{project_id, message, agent_id}` | stream SSE | Streaming chat with AI agent |
| `GetChatHistory` | `GetChatHistoryRequest{project_id, limit}` | `GetChatHistoryResponse{messages[]}` | Retrieve chat history |
| `GetDataStats` | `GetDataStatsRequest{project_id, table_name}` | `GetDataStatsResponse{columns[], stats{}}` | Column-level statistics |
| `ConfirmAction` | `ConfirmActionRequest{project_id, action_id, confirmed}` | `ConfirmActionResponse{result}` | Confirm or reject a pending action |
| `GlobalQuery` | `GlobalQueryRequest{sql}` | `GlobalQueryResponse{results[]}` | Cross-project query |
| `GetDataLineage` | `GetDataLineageRequest{project_id, table_name}` | `GetDataLineageResponse{sources[], transformations[]}` | Data lineage |
| `GetChecksum` | `GetChecksumRequest{project_id, table_name}` | `GetChecksumResponse{checksum}` | SHA-256 checksum for a table |

**Example — ExecuteQuery:**
```protobuf
// Request
{
  "project_id": "proj_abc123",
  "sql": "SELECT * FROM sales LIMIT 10"
}

// Response
{
  "columns": ["id", "product", "revenue", "date"],
  "rows": [
    [1, "Widget A", 1250.00, "2026-01-15"],
    [2, "Widget B", 890.50, "2026-01-16"]
  ]
}
```

---

## REST Endpoints

### Health & Diagnostics

#### `GET /api/v1/healthz` (no auth)
Health check for Docker HEALTHCHECK and load balancers.

**Response:** `200 OK`
```json
{"status": "ok"}
```

#### `GET /api/v1/tools/health`
Returns health status for all registered tools.

**Response:**
```json
{
  "tools": [
    {
      "id": "tool_csv_001",
      "status": "healthy",
      "last_check": "2026-04-27T10:00:00Z"
    }
  ]
}
```

#### `POST /api/v1/tools/verify`
Verify tool connectivity and functionality.

**Request:**
```json
{"tool_id": "tool_csv_001"}
```

**Response:**
```json
{
  "verified": true,
  "latency_ms": 42
}
```

#### `GET /api/v1/diagnostic/patterns`
Retrieve diagnostic patterns from the monitoring system.

**Response:**
```json
{
  "patterns": [
    {
      "name": "connection_timeout",
      "description": "Database connection timeout detected",
      "severity": "warning"
    }
  ]
}
```

---

### Tool Management

#### `GET /api/v1/tools`
List all registered tools.

**Response:**
```json
{
  "tools": [
    {
      "id": "tool_csv_001",
      "name": "csv_ingester",
      "description": "Import CSV files into DuckDB",
      "category": "ingestion",
      "version": "1.0.0"
    }
  ]
}
```

#### `GET /api/v1/tools/categories`
List available tool categories.

**Response:**
```json
{
  "categories": ["ingestion", "analysis", "finance", "osint", "codeflow"]
}
```

#### `POST /api/v1/tools/execute/{category}/{name}`
Execute a tool by category and name.

**Request:**
```json
{
  "parameters": {
    "file_path": "data/raw/sales.csv"
  }
}
```

**Response:**
```json
{
  "output": "Imported 15420 rows",
  "exit_code": 0,
  "duration_ms": 1240
}
```

#### `POST /api/v1/tools/call`
Call a tool with parameters.

**Request:**
```json
{
  "tool_id": "tool_csv_001",
  "parameters": {
    "file_path": "data/raw/sales.csv",
    "table_name": "sales"
  }
}
```

#### `POST /api/v1/tools/register`
Register a new tool.

**Request:**
```json
{
  "name": "json_ingester",
  "description": "Import JSON files",
  "code": "package main...",
  "category": "ingestion"
}
```

#### `POST /api/v1/tools/suggest`
Submit a tool suggestion for review.

**Request:**
```json
{
  "description": "A tool that fetches weather data",
  "use_case": "Enrich market data with weather patterns"
}
```

**Response:**
```json
{
  "suggestion_id": "sugg_001",
  "status": "pending_review"
}
```

#### `POST /api/v1/tools/suggest/approve`
Approve a suggested tool.

**Request:**
```json
{"suggestion_id": "sugg_001"}
```

---

### CodeFlow

#### `GET /api/v1/codeflow/graph`
Retrieve code dependency graph.

**Response:**
```json
{
  "nodes": [
    {"id": "pkg_decision", "type": "package", "label": "decision"}
  ],
  "edges": [
    {"from": "pkg_query", "to": "pkg_decision", "type": "import"}
  ]
}
```

#### `GET /api/v1/codeflow/metrics`
Get code metrics and statistics.

**Response:**
```json
{
  "packages": 35,
  "handlers": 33,
  "tests": 150,
  "coverage_percent": 78
}
```

#### `GET /api/v1/codeflow/executions`
List code execution history.

#### `GET /api/v1/codeflow/engines`
List available code execution engines.

---

### SSE Streaming

#### `GET /api/v1/events`
Server-Sent Events endpoint for real-time updates.

**Query params:** `?api_key=<key>`

**Response:** SSE stream
```
event: message
data: {"type": "chat_delta", "content": "The analysis shows..."}

event: message
data: {"type": "tool_call", "tool": "csv_ingester", "status": "running"}

event: done
data: {}
```

---

## Error Responses

### REST Errors
```json
{
  "error": {
    "code": "ERR_NON_TROVATO",
    "message": "Risorsa richiesta non esistente",
    "details": {}
  }
}
```

### ConnectRPC Errors
Standard gRPC status codes with protobuf error details.

| Code | HTTP Status | Meaning |
|------|-------------|---------|
| `ERR_AUTENTICAZIONE` | 401 | Invalid or missing credentials |
| `ERR_AUTORIZZAZIONE` | 403 | Insufficient permissions |
| `ERR_NON_TROVATO` | 404 | Resource not found |
| `ERR_VALIDAZIONE` | 400 | Invalid input data |
| `ERR_RATE_LIMIT` | 429 | Rate limit exceeded |
| `ERR_INTERNO` | 500 | Internal server error |
| `ERR_SERVIZIO_NON_DISPONIBILE` | 503 | External dependency unreachable |
| `ERR_TIMEOUT` | 504 | Operation timed out |
| `ERR_LIMITA_DIMENSIONE` | 413 | Payload too large |

---

## Rate Limiting

Limits are per API key, configurable via environment variables:

| Endpoint type | Default limit |
|---------------|---------------|
| Chat (`/aleph.v1.QueryService/Chat`) | 60 requests/minute |
| Health (`/api/v1/tools/health`) | 120 requests/minute |
| All other endpoints | 100 requests/minute |

---

## CORS

Default allowed origins:
```
http://localhost:5173,http://localhost:3000
```

Override via `CORS_ALLOWED_ORIGINS` environment variable.

---

## See Also

- [`docs/CONTRIBUTING.md`](./CONTRIBUTING.md) — Development guide
- [`docs/user-guide-en.md`](./user-guide-en.md) — User guide (English)
- [`docs/deployment-guide.md`](./deployment-guide.md) — Deployment instructions
