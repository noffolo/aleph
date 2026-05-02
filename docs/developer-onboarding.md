# Developer Onboarding — Aleph-v2

> **Version:** 2.0.0 · **Last updated:** April 2026 · **Audience:** New contributors

Welcome to the Aleph-v2 development team. This guide gets you from zero to a working development environment, then explains the architecture so you can contribute effectively.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Repository Setup](#2-repository-setup)
3. [Development Environment](#3-development-environment)
4. [Architecture Overview](#4-architecture-overview)
5. [Project Structure](#5-project-structure)
6. [Key Subsystems](#6-key-subsystems)
7. [Testing](#7-testing)
8. [Contribution Guidelines](#8-contribution-guidelines)
9. [Useful Commands](#9-useful-commands)
10. [Getting Help](#10-getting-help)

---

## 1. Prerequisites

You need the following installed on your machine:

| Tool | Minimum Version | Purpose |
|------|-----------------|---------|
| Go | 1.24 | Backend compilation |
| Node.js | 20 | Frontend build |
| Docker | 24.x | Services orchestration |
| Docker Compose | 2.x | Multi-container local runs |
| Git | 2.40 | Source control |
| Make | 3.81 | Build automation |

Optional but recommended:

- **Air** (`go install github.com/air-verse/air@latest`) — Go hot reload
- **Buf** (`brew install buf`) — Protobuf code generation
- **Python 3.12** — NLP sidecar development
- **Ollama** — Local LLM inference

---

## 2. Repository Setup

### Clone and configure

```bash
git clone <repository-url>
cd aleph-v2
cp .env.example .env
```

### Environment variables

Edit `.env`. The following are **mandatory**:

```bash
# Generate with: openssl rand -hex 32
KEY_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

# Generate with: openssl rand -hex 32 (or any 32+ char random string)
JWT_SECRET=your-jwt-secret-min-32-chars-long

# PostgreSQL connection
POSTGRES_DSN=postgres://postgres:password@localhost:5432/aleph?sslmode=disable
```

`LoadConfig()` returns a FATAL log if `KEY_ENCRYPTION_KEY` is missing. Do not skip this step.

### Install dependencies

**Backend:**
```bash
go mod download
```

**Frontend:**
```bash
cd frontend && npm install
```

**NLP sidecar (optional):**
```bash
cd nlp && pip install -r requirements.txt
```

---

## 3. Development Environment

### Start everything

```bash
make dev
```

This starts:
- Go backend on `http://localhost:8080` (with Air hot reload)
- Frontend dev server on `http://localhost:5173` (Vite HMR)
- PostgreSQL on `localhost:5432`
- Ollama on `localhost:11434`

### Service URLs

| Service | URL | Notes |
|---------|-----|-------|
| Frontend | `http://localhost:5173` | Vite dev server |
| Backend API | `http://localhost:8080` | ConnectRPC + REST |
| Swagger | `http://localhost:8080/swagger.json` | OpenAPI spec |
| PostgreSQL | `localhost:5432` | Metadata storage |
| Ollama | `localhost:11434` | LLM inference |
| NLP sidecar | `localhost:8001` | Python gRPC (if running) |
| Prometheus | `http://localhost:8080/metrics` | Metrics endpoint |

### Stop everything

```bash
docker compose down
```

---

## 4. Architecture Overview

Aleph-v2 is a three-tier architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React/TS)                       │
│  TerminalView · CopilotView · SlideOver · Cmd+K Palette     │
│  Zustand Composite Store · SSE Streaming · ConnectRPC      │
└────────────────────────┬────────────────────────────────────┘
                         │ ConnectRPC (HTTP/2) + SSE + REST
┌────────────────────────┴────────────────────────────────────┐
│                    Backend Go                                │
│  QueryHandler · ChatSession · DecisionEngine (PAORA)       │
│  13 ConnectRPC Services · Sandbox · Health · Diagnostic     │
│  7 Middleware · Genesis · Tools Registry · Audit              │
└──────────┬────────────────────────────────┬────────────────┘
           │ gRPC (HTTP/2 cleartext)        │ DuckDB (read-only)
┌──────────┴──────────┐     ┌───────────────┴────────────────┐
│  Python NLP Sidecar  │     │         PostgreSQL 16          │
│  Sentiment · ONNX    │     │    API Keys · Audit · Chat     │
│  Ensemble Prophet/GBM │     └────────────────────────────────┘
│  DuckDB read-only    │
└──────────────────────┘
```

### Design principles

- **Terminal-first**: the primary UI is an interactive terminal with slash commands, Cmd+K palette, and optional scanline/glow effects
- **PAORA loop**: every action flows through Plan → Act → Observe → Reflect → Admit with graceful degradation
- **Defense-in-depth**: every input is validated, every tool is sandboxed, every API key is encrypted
- **Observability-first**: OpenTelemetry + Prometheus + structured slog from day one

---

## 5. Project Structure

```
aleph-v2/
├── main.go                    # Entry point
├── go.mod / go.sum            # Go dependencies
├── Makefile                   # Build, dev, proto targets
├── Dockerfile                 # Multi-stage Go build
├── docker-compose.yml         # 4 services
├── .golangci.yml              # 20 linters
├── buf.gen.yaml               # Protobuf generation
├── .env.example               # Environment template
│
├── internal/                  # Backend source (35+ packages)
│   ├── api/
│   │   ├── handler/           # 33 handlers
│   │   ├── proto/             # Protobuf definitions
│   │   ├── sse/               # SSE broker
│   │   └── routes/            # Route registration
│   ├── decision/              # PAORA Engine
│   ├── diagnostic/            # Error pattern classification
│   ├── errors/                # APIError with Italian codes
│   ├── genesis/               # Tool suggestion pipeline
│   ├── gnn/                   # Graph Neural Network
│   ├── health/                # HealthChecker + HistoryStore
│   ├── ingestion/             # Data ingestion engine
│   ├── llm/                   # LLM provider interface
│   ├── mcp/                   # MCP discovery + SSRF guard
│   ├── middleware/            # 8 HTTP + 6 ConnectRPC interceptors
│   ├── repository/            # 30+ CRUD methods
│   ├── sandbox/               # ExecSandbox + SecurityScanner
│   ├── telemetry/             # OTel + Prometheus
│   └── tools/                 # Registry + 5 sub-packages
│
├── frontend/                  # React/TypeScript SPA
│   ├── src/
│   │   ├── App.tsx            # Main router
│   │   ├── store/             # 6 Zustand slices
│   │   ├── components/        # 30+ components
│   │   ├── hooks/             # 9 custom hooks
│   │   ├── api/               # 12 ConnectRPC clients
│   │   ├── schemas/           # 22 Zod schemas
│   │   ├── commands/          # 16 slash commands
│   │   └── styles/            # Design tokens + CSS
│   ├── tailwind.config.js
│   ├── vite.config.ts
│   └── package.json
│
├── nlp/                       # Python sidecar
│   ├── main.py                # gRPC server
│   ├── requirements.txt       # Dependencies
│   ├── Dockerfile             # Python 3.12-slim
│   └── tests/                 # pytest suite
│
├── api/proto/                 # Protobuf source of truth
│   └── aleph/
│       ├── v1/                # Core services
│       └── nlp/v1/            # NLP services
│
└── docs/                      # Documentation
```

---

## 6. Key Subsystems

### 6.1 Decision Engine (PAORA)

Every chat session runs a decision loop:

```go
type DecisionEngine interface {
    Plan(ctx, msg, projectID, agentID, ontContent, agent) (*PlanResult, error)
    PlanWithProvider(ctx, msg, projectID, agentID, ontContent, agent, provider) (*PlanResult, error)
    Act(ctx, step, projectID) (*ActResult, error)
    Observe(ctx, step, result) (*Observation, error)
    Reflect(ctx, plan, observations) (*PlanResult, error)
    Admit(ctx, results, maxAttempts) (bool, error)
}
```

- **Plan**: LLM generates a tool execution plan
- **Act**: Execute each step
- **Observe**: Collect results and side effects
- **Reflect**: Decide whether to continue or retry
- **Admit**: Final check before returning to user

Graceful degradation: if the LLM provider is unavailable, `Plan()` falls back to keyword-based heuristic planning.

### 6.2 Sandbox

`internal/sandbox/`:

- `ExecSandbox`: tool execution with timeout and resource limits
- `SecurityScanner`: blocks 9 dangerous patterns (`os/exec`, `syscall`, `unsafe`, etc.)
- `CommandAllowlist`: 14 permitted commands, 5 blocked flags
- Network isolation via `network_mode: none`
- Read-only filesystem via `read_only: true`

### 6.3 Middleware Stack

HTTP middleware (in order):
```
CORSHandler → CSRFProtection → AuthMiddleware → SecurityHeaders → AuditMiddleware
    → RateLimitMiddleware → TimeoutMiddleware → BulkheadMiddleware
```

ConnectRPC interceptors:
```
RecoveryInterceptor → AuthInterceptor → SecurityHeaders → AuditInterceptor
    → RateLimitInterceptor → TimeoutInterceptor → BulkheadInterceptor
```

### 6.4 Repository Layer

`MetadataRepository` (`internal/repository/metadata.go`):
- 30+ CRUD methods for tools, agents, skills, projects
- All queries use positional parameters (`$1`, `$2`) with `validName()` regex validation
- LRU cache for frequent lookups
- `ToolRecord` extended with Category, Version, HealthStatus, LastCheckedAt

`AuditRepository` (`internal/repository/audit.go`):
- Records every mutation with timestamp, projectID, action, JSON details

### 6.5 Frontend Store

Zustand composite store with 6 slices:

```
useStore()
├── authSlice      — projectID, apiKey, isAuthenticated
├── navigationSlice — activeView, selectedAgent, selectedSkill
├── copilotSlice   — messages, isLoading, streamingContent
├── workspaceSlice — agents, skills, tools, datasources, library
├── healthSlice    — toolHealth, systemHealth
└── uiSlice        — toasts, slideOverPanel, theme, editingState
```

### 6.6 NLP Sidecar

Python gRPC server on port 8001:

- `AnalyzeSentiment`: keyword-based ITA/EN sentiment scoring
- `StreamPredictions`: Prophet + GBM Monte Carlo ensemble (100 paths, 252 days)
- `RecordFeedback`: Brier score calibration

Communicates with the Go backend via circuit breaker with synthetic fallback.

---

## 7. Testing

### Go tests

Run the full suite with race detection:
```bash
go test -race -count=1 ./...
```

Run a specific package:
```bash
go test -race -count=1 ./internal/decision/
```

Run a single test:
```bash
go test -race -count=1 -run TestCSRF ./internal/middleware/
```

Integration tests (require DuckDB + PostgreSQL):
```bash
go test -count=1 ./internal/integration/...
```

### Frontend tests

```bash
cd frontend && npx vitest run        # Unit + integration
cd frontend && npx tsc --noEmit      # Type check
cd frontend && npx playwright test   # E2E
```

### Python NLP tests

```bash
cd nlp && python3 -m pytest tests/ -v
```

### CI pipeline

GitHub Actions runs 5 jobs:
1. Go Build (`go build`, `go vet`, `go test`)
2. Go Lint (`.golangci.yml` with 20 linters)
3. Frontend Build (`npm run build`, `tsc --noEmit`)
4. Frontend Test (`vitest run`)
5. E2E Test (`playwright test`)

---

## 8. Contribution Guidelines

### Branch naming

```
feature/your-feature-name
fix/bug-description
docs/what-you-updated
refactor/component-name
```

### Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add tool suggestion endpoint
fix: resolve N+1 query in GetDataStats
docs: update API reference
refactor: extract SSE broker logic
test: add coverage for CSRF middleware
```

### Before submitting

Run the full verification locally:
```bash
go test -race -count=1 ./... && \
cd frontend && npx vitest run && npx tsc --noEmit && \
docker compose config
```

### Code standards

**Go:**
- `go fmt` before committing
- `go vet` for static analysis
- Godoc comments for exported symbols
- Use `%w` for error wrapping
- Never use `fmt.Sprintf` for user-facing SQL

**TypeScript:**
- Strict mode enabled
- No `as any` in production (tests: 75 allowed for Zustand mocks; prod: 16 in D3 callbacks)
- Zod schemas for runtime validation
- Prefer Zustand over prop drilling

**CSS/Tailwind:**
- Utility classes preferred
- CSS variables for theme values (`design-tokens.json`)
- Volatility layers: `.vol-static`, `.vol-structural`, `.vol-interactive`, `.vol-signal`

### Pull request checklist

- [ ] Tests added or updated
- [ ] `go test -race -count=1 ./...` passes
- [ ] `go vet ./...` passes
- [ ] Frontend build + type check passes
- [ ] No secrets or API keys in code
- [ ] Documentation updated if API changed

---

## 9. Useful Commands

```bash
# Backend
make dev              # Start backend + frontend with hot reload
make build            # Build Go binary + frontend
make run              # Build and run
make clean            # Remove build artifacts

# Frontend
cd frontend && npm run dev      # Vite dev server
cd frontend && npm run build    # Production build
cd frontend && npm run test     # Vitest

# NLP
cd nlp && python main.py        # Start gRPC server
cd nlp && pytest                # Run tests

# Protobuf
make proto            # Regenerate Go protobuf
make proto-python     # Regenerate Python protobuf

# Docker
docker compose up -d           # Start all services
docker compose logs -f backend # Tail backend logs
docker compose down            # Stop everything

# Linting
golangci-lint run     # Run all Go linters
cd frontend && npx tsc --noEmit  # TypeScript type check
```

---

## 10. Getting Help

### Documentation

- [`docs/API.md`](./API.md) — API reference (legacy)
- [`docs/api-reference.md`](./api-reference.md) — Full API reference
- [`docs/manuale-tecnico.md`](./manuale-tecnico.md) — Technical manual (Italian, comprehensive)
- [`docs/user-guide-en.md`](./user-guide-en.md) — User guide (English)
- [`docs/deployment-guide.md`](./deployment-guide.md) — Deployment instructions
- [`docs/runbook.md`](./runbook.md) — Operational procedures

### Architecture

- [`AGENTS.md`](../AGENTS.md) — Agent system and workflow
- [`docs/CHANGELOG.md`](./CHANGELOG.md) — Release history

### Community

- Open an issue for bugs or feature requests
- Join discussions for architectural decisions
- Tag `@maintainers` for security-related questions

---

Welcome aboard. Start with `make dev`, explore the terminal interface, and read `internal/decision/engine.go` to understand the heart of the system.
