# CI/CD Pipeline Configuration for Aleph-v2

## Overview
This document describes the CI/CD pipeline for Aleph-v2. The pipeline ensures code quality, automated testing, and deployment readiness across Go backend, React/TypeScript frontend, and Docker.

## GitHub Actions Workflows

### 1. CI Pipeline (`.github/workflows/ci.yml`)
**Trigger:** On push to `main` branch, pull requests, and manual dispatch

**Jobs:**
- **Go Backend:** Linting (`golangci-lint`), tests with `go test -race -count=1 ./...` (no caching), `go vet ./...`, binary existence check
- **Frontend:** TypeScript type checking (`tsc --noEmit`), `vitest run`, Vite build with `--logLevel error`, Playwright E2E tests
- **Docker:** Build verification (`docker build .`), `docker compose config` check
- **Docker Push:** On push to main only — builds and pushes to Docker Hub with `DOCKER_USERNAME`/`DOCKER_PASSWORD` secrets. Tags: `latest` and `{branch}-{short-sha}`.
- **CI Summary:** Status reporting for all preceding jobs

**Key Checks:**
- Go module verification and caching (`go mod download && go mod verify`)
- Race detection in Go tests (`-race` flag)
- TypeScript type checking (`--noEmit`)
- Vitest (113 tests across 17 files)
- Multi-stage Docker build validation

### 2. Security Scan (`.github/workflows/security.yml`)
**Trigger:** On push to `main`, pull requests, and weekly cron (Monday 6am UTC)

**Jobs:**
- **Secrets scan:** Uses `gitleaks/gitleaks-action@v2` to detect hardcoded secrets, API keys, and credentials in the repository

### 3. Deploy (`.github/workflows/deploy.yml`)
**Trigger:** On tag push matching `v*` (e.g. `v1.2.3`)

**Jobs:**
- **Build and Push:** Builds Docker image with semver tags (`{version}`, `{major}.{minor}`), pushes to Docker Hub using `DOCKER_USERNAME`/`DOCKER_PASSWORD` secrets
- Placeholder for staging/production deployment triggers

## Branch Protection Recommendations

### Required Status Checks (GitHub Settings → Branches → main)
1. **Go Backend** (`go-backend`)
2. **Frontend** (`frontend`)
3. **Docker Build Verification** (`docker`)

### Additional Branch Protection Rules
1. Require 1 approval minimum for PRs
2. Dismiss stale approvals on new commits
3. Require status checks to pass before merging
4. Require conversation resolution before merging

## Configuration Files

### `.golangci.yml`
Enabled linters: `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`
Disabled (too strict): `wsl`, `funlen`, `gocognit`, `nestif`
Project-specific exceptions: `algedonic`, `aleph`

### Docker Configuration
- **Root Dockerfile:** Builds Go backend binary
- **Frontend Dockerfile:** Nginx serving Vite-built assets
- **Python NLP Dockerfile:** Separate NLP sidecar container
- **docker-compose.yml:** Full stack with PostgreSQL, Ollama (pre-pulls llama3+nomic-embed-text), NLP sidecar, and frontend

## Local Development Validation
```bash
# Go
go vet ./...
go test -race -count=1 ./...

# Frontend
cd frontend && npx tsc --noEmit && npx vitest run && npx vite build

# Docker
docker compose config
```

## Security
- **Secrets scanning**: `gitleaks` in CI via security.yml
- **Go vet**: Enforced in CI pipeline
- **TypeScript**: Strict mode, `--noEmit` type checking
- No ESLint in CI (tsc --noEmit provides sufficient type coverage)

---

*Last updated: 2026-04-30*