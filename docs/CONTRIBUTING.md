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

Create your local environment file:

```bash
cp .env.example .env
```

Edit `.env` with your configuration values.

### 3. Install Dependencies

**Backend:**
```bash
go mod download
```

**Frontend:**
```bash
npm install
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

Run tests:
```bash
go test ./...
```

### Frontend

```bash
npm run build
```

Run tests:
```bash
npx playwright test
```

## Development Workflow

### Running Locally

- **Frontend:** `http://localhost:5173`
- **Backend API:** `http://localhost:8080`
- **API Documentation:** `http://localhost:8080/swagger.json`

### Hot Reload

The `make dev` command enables hot reload for both backend and frontend during development.

## Coding Standards

### Go

- Format code with `go fmt` before committing
- Run `go vet` for static analysis
- Follow Go best practices and standard library conventions
- Use meaningful variable and function names
- Document public APIs with godoc comments

### TypeScript

- Strict mode enabled in `tsconfig.json`
- No implicit `any` types
- Use TypeScript interfaces for API contracts
- Follow React best practices (functional components, hooks)
- Run `npm run lint` before committing

### CSS/Tailwind

- Use Tailwind utility classes for styling
- Follow the design token system (see `docs/`)
- Maintain dark palette `#080810` as base
- Use CSS variables for theme values

## Pull Request Process

### 1. Branch from Main

```bash
git checkout main
git pull
git checkout -b feature/your-feature-name
```

### 2. Make Changes

- Keep changes focused and atomic
- Write tests for new functionality
- Update documentation as needed

### 3. Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
feat: add tool suggestion endpoint
fix: resolve N+1 query in GetDataStats
docs: update API reference
refactor: extract SSE broker logic
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

### 4. Squash Commits

Before merging, squash related commits into logical units:

```bash
git rebase -i main
```

### 5. Create Pull Request

- Push your branch: `git push origin feature/your-feature-name`
- Open a PR on GitHub
- Fill out the PR template
- Request review from maintainers

### 6. Review Process

- Address review feedback
- Ensure CI checks pass
- Maintain code quality standards

## Architecture Overview

For architecture details, see:

- [`AGENTS.md`](../AGENTS.md) — Agent system and workflow
- [`docs/API.md`](./API.md) — API reference
- [`docs/CHANGELOG.md`](./CHANGELOG.md) — Release history

## Questions?

Open an issue for clarification on any aspect of development.
