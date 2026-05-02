# Contributing to Aleph-v2

This guide covers local setup, development workflow, and contribution standards.

## Prerequisites

- **Go** 1.24 or later
- **Node.js** 22 or later
- **Docker** and Docker Compose
- **Git**

## Local Setup

### 1. Clone the Repository

```bash
git clone https://github.com/noffolo/aleph.git
cd aleph
```

### 2. Configure the Environment

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

### 4. Start the Development Environment

```bash
make dev
```

This starts both backend and frontend with hot reload enabled. Alternatively, run the full stack with Docker:

```bash
docker compose up --build -d
```

**Access points:**

| Service | URL |
|---------|-----|
| Frontend | `http://localhost:5173` |
| Backend API | `http://localhost:8080` |
| API Docs | `http://localhost:8080/swagger.json` |
| PostgreSQL | `localhost:5432` |
| Ollama | `localhost:11434` |

**Hot reload:**

- Backend: AIR (`.air.toml`) — `make dev` triggers rebuild on file changes.
- Frontend: Vite HMR — instant hot module replacement.

## Branch Strategy

All work happens on branches prefixed with `feat/`.

```bash
git checkout main && git pull
git checkout -b feat/your-feature-name
```

- `main` is the default branch and is always deployable.
- Feature branches are short-lived and merged via Pull Request.
- Delete branches after merge to keep the repository clean.

## Conventional Commits

Use [Conventional Commits](https://www.conventionalcommits.org/) for all commit messages.

| Type | Use When |
|------|----------|
| `feat` | Adding a new feature or user-visible capability |
| `fix` | Fixing a bug |
| `chore` | Maintenance tasks, dependency updates, CI tweaks |
| `docs` | Documentation only changes |
| `test` | Adding or correcting tests |
| `refactor` | Code changes that neither fix bugs nor add features |

**Examples:**

```bash
git commit -m "feat: add memory query endpoint with VSS"
git commit -m "fix: resolve race in SSE broker on client disconnect"
git commit -m "refactor: extract sandbox runner into separate package"
git commit -m "docs: update API reference for ingest endpoint"
git commit -m "test: add edge-case coverage for DuckDB timeout"
git commit -m "chore: bump Go to 1.24.2 in CI"
```

## Running Tests Before a Pull Request

Always run the full test suite before opening a PR. The following commands must pass.

### Backend

```bash
# Build
go build ./...

# All tests with race detector
go test -race -count=1 ./...

# Static analysis
go vet ./...
```

### Frontend

```bash
cd frontend

# TypeScript type check
npx tsc --noEmit

# Unit and integration tests
npx vitest run

# Optional: E2E tests (requires built frontend)
npx playwright test
```

## Code Review Process

1. Open a Pull Request against `main`.
2. Ensure all CI checks pass (Go build + test, Frontend build + test, Docker validation).
3. Request review from at least one maintainer.
4. Address review feedback promptly. Mark conversations as resolved once fixed.
5. **Approval criteria:** correctness, test coverage, performance impact, security implications, and documentation updates.
6. The maintainer merges once approved. Do not merge your own PR without a second pair of eyes.

## Style Guide

### Go

- Run `gofmt` on every save or before committing.
- Run `go vet` for static analysis.
- Document exported symbols with godoc comments.
- Wrap errors with `%w`, never drop them silently.
- Parameterize all SQL queries. Never use `fmt.Sprintf` for user input.

### TypeScript

- Strict mode is enforced in `tsconfig.json`.
- No `as any` in production code. (Tests allow 75 instances for Zustand mocks; prod allows 16 in D3 callbacks only.)
- Use Zod schemas for runtime validation of API responses and user input.
- Prefer Zustand over prop drilling for global state.
- Run ESLint and Prettier before committing:

```bash
cd frontend
npm run lint
npx prettier --check .
```

### CSS / Tailwind

- Prefer Tailwind utility classes.
- Use CSS variables for theme values (see Tailwind config or `frontend/src/index.css`).
- Dark palette `#080810` is the background base.
- Volatility layers: `.vol-static`, `.vol-structural`, `.vol-interactive`, `.vol-signal`.

## Reporting Bugs

Open a GitHub issue and include the following:

- **What you were trying to do** — steps to reproduce.
- **What you expected** — the correct behavior.
- **What happened instead** — actual behavior with logs.
- **Environment** — Go version, Node version, OS, and commit hash.
- **Relevant logs or screenshots** — sanitize any secrets before pasting.

If you have a fix, reference the issue number in your PR description (e.g., `Fixes #123`).

## Additional Resources

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — System design and data flow
- [`docs/API.md`](docs/API.md) — API contracts and protobuf definitions
- [`docs/CHANGELOG.md`](docs/CHANGELOG.md) — Release history
- [`SECURITY.md`](SECURITY.md) — Vulnerability reporting and security model
- [`AGENTS.md`](AGENTS.md) — Agent system and workflow