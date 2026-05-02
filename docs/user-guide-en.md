# User Guide — Aleph-v2

> **Version:** 2.0.0 · **Last updated:** April 2026 · **Language:** English

Welcome to Aleph-v2, the multi-agent predictive intelligence platform. This guide helps you use the application from basic concepts to advanced workflows.

---

## Table of Contents

1. [Getting Started](#1-getting-started)
2. [Key Concepts](#2-key-concepts)
3. [Terminal Interface](#3-terminal-interface)
4. [Main Workflows](#4-main-workflows)
5. [Project Management](#5-project-management)
6. [Agents and Skills](#6-agents-and-skills)
7. [Tools and Sandbox](#7-tools-and-sandbox)
8. [Chat and Data Analysis](#8-chat-and-data-analysis)
9. [Data Ingestion](#9-data-ingestion)
10. [Notifications and Webhooks](#10-notifications-and-webhooks)
11. [Security and Privacy](#11-security-and-privacy)
12. [Troubleshooting](#12-troubleshooting)

---

## 1. Getting Started

### Access

Open your browser at the Aleph server address (for example `http://localhost:5173` locally). You are presented with an interactive terminal, the main interface of the application.

### First Login

At startup you need to enter an API key. If you do not have one, ask your system administrator to create one via `AuthService > CreateApiKey`. The key is shown only once: copy it and store it in a password manager.

### Quick Navigation

| Shortcut | Action |
|----------|--------|
| `Cmd+K` (Mac) / `Ctrl+K` (Win) | Open command palette |
| `↑` / `↓` | Scroll command history |
| `Tab` | Command autocompletion |
| `Esc` | Close panels and modals |

---

## 2. Key Concepts

### Project

A project is a container for data, agents, tools, and skills. Each project has a separate directory (`raw/`, `ontologies/`, `agents/`, `skills/`). You can switch between projects without leaving the interface.

### Agent

An agent is an AI instance configured with an LLM provider (for example Ollama), a model (for example llama3), and a system prompt that defines its behavior. Each agent can have one or more skills.

### Skill

A skill groups one or more tools for specific purposes. For example, the "Financial Analysis" skill might include tools for sentiment analysis, market data fetching, and report generation.

### Tool

A tool is an executable piece of code that performs a precise task: import a CSV, analyze text sentiment, run a DuckDB query. Tools run inside an isolated sandbox.

### Ontology

The ontology describes the data structure of the project (tables, columns, relationships). You can read it, edit it, or generate it automatically from the DuckDB database.

### PAORA Cycle

Every interaction with the agent follows the decision cycle Plan > Act > Observe > Reflect > Admit. This ensures the agent plans, executes, verifies, and admits results autonomously.

---

## 3. Terminal Interface

### Layout

The interface is divided into three areas:

- **Top header**: active project name, selected agent name, connection status
- **Central chat area**: user messages and agent replies, with inline rendering of tables, charts, and tools
- **SlideOver side panel**: opens on the right for complex forms (agent creation, file upload, settings)

### Slash Commands

Type `/` to see the list of 16 built-in commands:

| Command | What it does |
|---------|--------------|
| `/help` | Show all available commands |
| `/clear` | Clear the chat session |
| `/model` | Change LLM model |
| `/agent` | List or switch active agent |
| `/tool` | Tool management (install, list, health, diagnose) |
| `/skills` | Show current agent skills |
| `/status` | Connection and services status |
| `/export` | Export conversation to Markdown |
| `/diagnose` | Run quick diagnostics |
| `/theme` | Toggle light/dark theme |
| `/debug` | Toggle debug mode |

### Visual Effects

You can activate scanline, flicker, and glow effects from the settings menu (`/theme`). These are purely aesthetic and do not affect functionality.

---

## 4. Main Workflows

### 4.1 Data Analysis via Chat

1. Select a project from the header
2. Type a question in the terminal: `SHOW ME sales BY month`
3. The agent plans the action (Plan), executes the DuckDB query (Act), shows the results (Observe)
4. If the result is good, the agent confirms it (Admit). Otherwise, it retries (Reflect)
5. You can continue the conversation with follow-up questions

### 4.2 Data Import

1. Open the side panel with `Cmd+K` > "New Data Source"
2. Choose the source: CSV, JSON, API URL, Google Sheets, RSS, GitHub
3. Configure parameters (for example, file URL or SQL query)
4. Start the ingestion task
5. Check progress with `/status` or in the Ingestion panel

### 4.3 Creating a Custom Agent

1. Type `/agent` > "Create new agent"
2. Fill in the form in the side panel:
   - Name (for example, "Market Analyst")
   - LLM provider (Ollama, OpenAI)
   - Model (llama3, gpt-4)
   - System prompt (behavior instructions)
   - Skills to assign
3. Save and activate the agent

### 4.4 Registering a New Tool

1. Type `/tool` > "Register tool"
2. Enter name, description, and Go code for the tool
3. The system runs a security scan (SecurityScanner) before registration
4. The tool becomes available for all projects

---

## 5. Project Management

### Creating a Project

```
Cmd+K > New Project
```

Enter name and description. The system automatically creates the directory structure:
```
data/projects/<project-name>/
├── raw/           # Source files
├── ontologies/    # Ontology definitions
├── agents/        # Agent configurations
└── skills/        # Skill configurations
```

### Switching Project

Click the project name in the header and select another from the list, or use:
```
/project <project-name>
```

### Ontology

To see the current data structure:
```
/ontology show
```

To generate it automatically from the database:
```
/ontology emerge
```

To edit it manually:
```
/ontology edit
```

---

## 6. Agents and Skills

### Switching Agent

```
/agent <agent-name>
```

Or use the palette `Cmd+K` > "Switch Agent".

### Built-in Skills

| Skill | Included Tools | Typical Use |
|-------|----------------|-------------|
| Data Query | `execute_query`, `get_data_stats` | Database exploration |
| Sentiment Analysis | `analyze_sentiment` | Opinion mining on text |
| Prediction | `stream_predictions` | Time series forecasting |
| Ingestion | `csv_ingester`, `json_ingester` | Data import |
| CodeFlow | `code_metrics`, `dependency_graph` | Code analysis |

### Assigning Skills

In the agent edit form (SlideOver > AgentForm), check the skills you want to activate. The agent will use them automatically when it detects a compatible task.

---

## 7. Tools and Sandbox

### Running a Tool

You can call a tool directly from chat:
```
Run csv_ingester on data/raw/sales.csv
```

Or via REST endpoint:
```bash
curl -X POST http://localhost:8080/api/v1/tools/call \
  -H "X-Aleph-Api-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{"tool_id": "tool_csv_001", "parameters": {"file_path": "sales.csv"}}'
```

### Sandbox Security

Tools run in an isolated environment with these restrictions:

- Configurable execution timeout
- Allowlist of 14 permitted commands
- Blocked flags (`-rf`, `--force`, `--no-dry-run`, `-exec`, `--allow-root`)
- Regex blocking shell metacharacters
- No network access (`network_mode: none`)
- Read-only filesystem (`read_only: true`)

### Tool Health Check

Check the status of all tools:
```
/tool health
```

History for a specific tool:
```
/tool health <tool-id>
```

---

## 8. Chat and Data Analysis

### Streaming Chat

The chat uses SSE (Server-Sent Events) for real-time streaming. You see the agent response appear word by word, without waiting.

### Inline Tool Call

When the agent decides to use a tool, you see an inline box in the chat with:
- Tool name
- Passed parameters
- Returned output
- Execution time

### Action Confirmation

For destructive actions (delete, mass update), the agent asks for confirmation:
```
⚠️ Action required: delete table 'sales_2025'
Confirm? (yes/no)
```

### Exporting Conversations

```
/export
```

The conversation is downloaded as a Markdown file with timestamp.

---

## 9. Data Ingestion

### Supported Sources

| Source | Configuration | Example |
|--------|---------------|---------|
| CSV | Local file path | `./data/raw/sales.csv` |
| JSON | URL or path | `https://api.example.com/data.json` |
| API | Endpoint + headers | `GET /api/v1/users` |
| Google Sheets | Spreadsheet ID + range | `1BxiMV.../Sheet1!A1:D10` |
| RSS | Feed URL | `https://news.ycombinator.com/rss` |
| GitHub | Repo + path | `owner/repo/data/` |
| Email | IMAP config | `imap.gmail.com:993` |

### Monitoring Ingestion

During an active ingestion task:
```
/status
```

Shows:
- Completion percentage
- Processed / total rows
- Any errors
- Real-time logs

---

## 10. Notifications and Webhooks

### Configuring a Webhook

1. Open the Notifications panel (SlideOver)
2. Add a webhook channel
3. Enter URL and secret (optional)
4. Choose events to notify (ingestion complete, tool failure, health alert)

### Manual Send

You can send a test webhook:
```
/notify send https://hooks.example.com/aleph {"event": "test"}
```

---

## 11. Security and Privacy

### API Keys

- API keys are SHA-256 hashed before storage
- Encrypted at rest with AES-256-GCM
- Shown in plaintext only at creation time
- Revocable at any time

### Sensitive Data

- No logs contain API keys or passwords
- Tool parameters are validated with regex
- SQL injection is impossible thanks to parameterized queries
- Tool code is scanned before execution

### Audit

Every write operation (create, update, delete) is recorded with:
- Timestamp
- Project ID
- Action performed
- JSON details of the operation

You can view the audit log via REST API or the administration panel.

---

## 12. Troubleshooting

### Agent Not Responding

1. Check `/status` — is the LLM service (Ollama) running?
2. Verify the agent has a valid provider and model
3. If Ollama is down, the agent falls back to degraded mode (heuristic planning)

### "Tool Not Found" Error

1. Verify the tool is registered: `/tool list`
2. Check that the agent skill includes that tool
3. If it is an MCP tool, check connectivity: `/mcp status`

### Slow Query

1. Check table statistics: `GET /api/v1/query/data-stats`
2. Check for missing indexes on filtered columns
3. For very large tables, use `LIMIT` in queries

### Authentication Issues

1. Verify the `X-Aleph-Api-Key` header is present
2. Check that the key has not expired or been revoked
3. For CORS issues, verify the origin is in `CORS_ALLOWED_ORIGINS`

### Docker Healthcheck Failed

| Service | Diagnostic command |
|---------|----------------------|
| Backend | `docker compose exec aleph-backend wget -qO- http://localhost:8080/readyz` |
| NLP | `docker compose exec aleph-nlp-sidecar python -c "import grpc; print('ok')"` |
| DB | `docker compose exec aleph-db pg_isready -U postgres` |

---

## Quick Glossary

| Term | Meaning |
|------|---------|
| **Agent** | AI instance with LLM model, prompt, and skills |
| **Skill** | Grouping of tools for a purpose |
| **Tool** | Executable code in sandbox |
| **Project** | Isolated container of data and configurations |
| **Ontology** | Project data schema |
| **PAORA** | Cycle Plan > Act > Observe > Reflect > Admit |
| **SSE** | Server-Sent Events, HTTP streaming |
| **MCP** | Model Context Protocol, for external tools |

---

## Other Guides

- [`docs/user-guide-it.md`](./user-guide-it.md) — Guida utente in italiano
- [`docs/api-reference.md`](./api-reference.md) — Complete API reference
- [`docs/deployment-guide.md`](./deployment-guide.md) — Deployment guide
