# Contributing to Aleph-v2

This guide covers local setup, development workflow, and contribution standards.

## Prerequisites

- **Go** 1.24 or later
- **Node.js** 20 or later
- **Docker** and Docker Compose
- **Git**

## Local Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd aleph-v2
```

### 2. Environment Configuration

```bash
cp .env.example .env
# REQUIRED: set KEY_ENCRYPTION_KEY
echo "KEY_ENCRYPTION_KEY=$(openssl rand -hex 32)" >> .env
```

Edit `.env` with your configuration values. `KEY_ENCRYPTION_KEY` is mandatory — `LoadConfig()` returns FATAL if missing.

### 3. Install Dependencies

**Backend:**
```bash
go mod download
```

**Frontend:**
```bash
cd frontend && npm install
```

### 4. Start Development Environment

```bash
make dev
```

This starts both backend and frontend with hot reload enabled.

## Building

### Backend

```bash
go build ./...
```

Run tests with race detection:
```bash
go test -race -count=1 ./...
```

Run vet:
```bash
go vet ./...
```

### Frontend

```bash
cd frontend && npx vite build
```

Run tests:
```bash
cd frontend && npx vitest run && npx tsc --noEmit
```

### Docker

```bash
docker compose config   # validate YAML
docker compose build    # build all services
```

## Development Workflow

### Running Locally

- **Frontend:** `http://localhost:5173`
- **Backend API:** `http://localhost:8080`
- **API Documentation:** `http://localhost:8080/swagger.json`
- **PostgreSQL:** `localhost:5432`
- **Ollama:** `localhost:11434`

### Hot Reload

- Backend: Air (`.air.toml`) — `make dev` triggers rebuild on file changes
- Frontend: Vite HMR — instant hot module replacement

## Testing

### Go Tests

```bash
# All tests with race detector
go test -race -count=1 ./...

# Specific package
go test -race -count=1 ./internal/middleware/...

# Single test
go test -race -count=1 -run TestCSRF ./internal/middleware/

# Integration tests (require DuckDB + PostgreSQL)
go test -count=1 ./internal/integration/...
```

### Frontend Tests

```bash
# Vitest (unit + integration)
cd frontend && npx vitest run

# TypeScript type check (no emit)
cd frontend && npx tsc --noEmit

# E2E (Playwright) — requires built frontend
cd frontend && npx playwright test
```

### Python NLP Tests

```bash
cd nlp && python3 -m pytest tests/ -v
```

## Coding Standards

### Go

- Format code with `go fmt` before committing
- Run `go vet` for static analysis
- Document exported symbols with godoc comments
- Use `%w` for error wrapping
- Parameterize all SQL queries — never use `fmt.Sprintf` for user input

### TypeScript

- Strict mode in `tsconfig.json`
- No `as any` in production code (tests: 75 as any for Zustand mocks; prod: 16 allowed in D3 callbacks)
- Use Zod schemas for runtime validation
- Prefer Zustand over prop drilling for global state

### CSS/Tailwind

- Prefer Tailwind utility classes
- CSS variables for theme values (see `design-tokens.json`)
- Dark palette `#080810` as background base
- Volatility layers: `.vol-static`, `.vol-structural`, `.vol-interactive`, `.vol-signal`

## Pull Request Process

### 1. Branch from Main

```bash
git checkout main && git pull
git checkout -b feature/your-feature-name
```

### 2. Make Changes

- Keep changes focused and atomic
- Write tests for new functionality
- Run full test suite before pushing: `go test -race -count=1 ./... && cd frontend && npx vitest run && npx tsc --noEmit`

### 3. Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add tool suggestion endpoint
fix: resolve N+1 query in GetDataStats
docs: update API reference
refactor: extract SSE broker logic
```

### 4. Create Pull Request

- Push your branch: `git push origin feature/your-feature-name`
- Ensure CI checks pass (Go + Frontend + Docker)
- Request review from maintainers

## Architecture Overview

For architecture details:
- [`AGENTS.md`](../AGENTS.md) — Agent system and workflow
- [`docs/API.md`](./API.md) — API reference
- [`docs/CHANGELOG.md`](./CHANGELOG.md) — Release history
- [`docs/manuale-tecnico.md`](./manuale-tecnico.md) — Full technical manual (Italian)
