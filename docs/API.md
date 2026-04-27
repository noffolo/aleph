# Aleph-v2 API Reference

## REST Endpoints

All REST endpoints require authentication via `X-Aleph-Api-Key` header.

### Health & Diagnostics

#### `GET /api/v1/healthz`
Health check endpoint for load balancers and monitoring.

**Response:** `200 OK` on healthy status

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

### QueryService

**Prefix:** `/aleph.v1.QueryService/`

| Method | Description |
|--------|-------------|
| `ListAgents` | List all registered agents |
| `ListSkills` | List available skills |
| `ListTools` | List registered tools |

---

### AgentService

**Prefix:** `/aleph.v1.AgentService/`

| Method | Description |
|--------|-------------|
| `CreateAgent` | Create a new agent |
| `GetAgent` | Get agent by ID |
| `UpdateAgent` | Update agent configuration |
| `DeleteAgent` | Remove an agent |
| `ListAgents` | List all agents |

---

### SkillService

**Prefix:** `/aleph.v1.SkillService/`

| Method | Description |
|--------|-------------|
| `CreateSkill` | Register a new skill |
| `GetSkill` | Get skill by ID |
| `UpdateSkill` | Update skill configuration |
| `DeleteSkill` | Remove a skill |
| `ListSkills` | List all skills |

---

### ToolService

**Prefix:** `/aleph.v1.ToolService/`

| Method | Description |
|--------|-------------|
| `CreateTool` | Register a new tool |
| `GetTool` | Get tool by ID |
| `UpdateTool` | Update tool configuration |
| `DeleteTool` | Remove a tool |
| `ListTools` | List all tools |

---

### DataSourceService

**Prefix:** `/aleph.v1.DataSourceService/`

| Method | Description |
|--------|-------------|
| `CreateDataSource` | Register a data source |
| `GetDataSource` | Get data source by ID |
| `UpdateDataSource` | Update data source |
| `DeleteDataSource` | Remove a data source |
| `ListDataSources` | List all data sources |

---

### LibraryService

**Prefix:** `/aleph.v1.LibraryService/`

| Method | Description |
|--------|-------------|
| `ListLibraries` | List available libraries |
| `GetLibrary` | Get library details |

---

### ProjectService

**Prefix:** `/aleph.v1.ProjectService/`

| Method | Description |
|--------|-------------|
| `CreateProject` | Create a new project |
| `GetProject` | Get project by ID |
| `UpdateProject` | Update project |
| `DeleteProject` | Remove a project |
| `ListProjects` | List all projects |

---

### NotificationService

**Prefix:** `/aleph.v1.NotificationService/`

| Method | Description |
|--------|-------------|
| `CreateNotification` | Create a notification |
| `ListNotifications` | List notifications |
| `MarkRead` | Mark notification as read |

---

### AuthService

**Prefix:** `/aleph.v1.AuthService/`

| Method | Description |
|--------|-------------|
| `Login` | Authenticate user |
| `Logout` | End session |
| `RefreshToken` | Refresh authentication token |

---

### IngestionService

**Prefix:** `/aleph.v1.IngestionService/`

| Method | Description |
|--------|-------------|
| `IngestData` | Ingest data from source |
| `GetIngestionStatus` | Check ingestion progress |

---

### SandboxService

**Prefix:** `/aleph.v1.SandboxService/`

| Method | Description |
|--------|-------------|
| `ExecuteTool` | Execute tool in sandbox |
| `RunSkill` | Run skill orchestration |

---

### RegistryService

**Prefix:** `/aleph.v1.RegistryService/`

| Method | Description |
|--------|-------------|
| `RegisterComponent` | Register a component |
| `ListComponents` | List registered components |

---

### NLPService

**Prefix:** `/aleph.nlp.v1.NLPService/`

| Method | Description |
|--------|-------------|
| `AnalyzeText` | Perform NLP analysis |
| `ExtractEntities` | Extract named entities |

---

## Authentication

All API requests require authentication:

**REST endpoints:**
```
X-Aleph-Api-Key: <your-api-key>
```

**ConnectRPC:**
```
X-Aleph-Api-Key: <your-api-key>
```

API keys are managed through the AuthService and stored in the metadata repository.

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
