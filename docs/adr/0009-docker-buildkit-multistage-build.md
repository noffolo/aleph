# ADR-0009: Docker + BuildKit Multi-Stage Build

## Status

Accepted

## Context

Aleph Data OS is a polyglot application requiring compilation of three components into a deployable artifact:

1. **Go backend** — ConnectRPC service (CGO disabled, static binary)
2. **TypeScript frontend** — React SPA built with Vite, served as static files
3. **Python sidecar** — NLP service with ONNX model artifacts

For local development, the system also requires: PostgreSQL, DuckDB, Ollama (embedding models), and monitoring stack (Grafana, Prometheus, AlertManager).

Key requirements:
- **Reproducible builds**: Same source → same binary, regardless of host environment
- **Minimal final image**: Runtime should not contain compilers, package managers, or source code
- **Single artifact deploy**: Frontend embedded in Go binary via `go:embed` for one-binary deployment
- **Fast development cycles**: Docker Compose for local orchestration with hot-reload support

## Decision

### Production Build: Multi-Stage Docker with BuildKit

The `Dockerfile` uses four stages:

| Stage | Base Image | Job | Output |
|-------|-----------|-----|--------|
| 1. Go build | `golang:1.26-alpine` | `go build` with CGO_ENABLED=0 | Static Go binary |
| 2. Frontend | `node:22-alpine` | `npm ci && npm run build` | `frontend/dist/` |
| 3. Model prep | `python:3.12-slim` | Download/verify ONNX models | `models/` directory |
| 4. Runtime | `gcr.io/distroless/static-debian12` | Copy binary + embedded frontend + models | Final image |

- Frontend `dist/` is embedded into the Go binary at compile time via `//go:embed frontend/dist/*`
- Final runtime image is `distroless/static` — no shell, no package manager, no compilers
- BuildKit caching used for Go module cache and npm cache layers
- ONNX model download is conditional (skip if no `--build-arg INCLUDE_MODELS=true`)

### Development: Docker Compose

`docker-compose.yml` orchestrates the full local stack:

| Service | Image | Purpose |
|---------|-------|---------|
| `backend` | Local build | Go binary with Air hot-reload |
| `db` | `postgres:16-alpine` | PostgreSQL metadata store |
| `sidecar` | `python:3.12-slim` | NLP gRPC sidecar |
| `ollama` | `ollama/ollama` | Local embedding model serving |
| `grafana` | `grafana/grafana` | Metrics visualization |
| `prometheus` | `prom/prometheus` | Metrics collection |
| `alertmanager` | `prom/alertmanager` | Alert routing |

### Key Architecture

- **`go:embed`** bundles all frontend static files into the Go binary. The Go server serves them directly — no separate file server, no CDN required for deployment.
- **Air** (Go live-reload) used in development Compose profile — source changes trigger automatic rebuild.
- **Volume mounts** for: Go module cache, npm cache, PostgreSQL data, Ollama models, Prometheus data.

## Consequences

### Positive
- Fully reproducible builds (Docker layer caching)
- Minimal final image — distroless with no shell, no attack surface
- `go:embed` produces a single binary deployable artifact
- Compose covers the full development stack in one command (`docker compose up`)
- BuildKit caching speeds iterative builds significantly
- Conditional model download keeps build fast for standard deployments

### Negative
- Multi-GB intermediate build stages (particularly the Go builder and model stage)
- Compose orchestrates 6+ containers — heavy for resource-constrained development
- ONNX model download adds significant build time when enabled
- Distroless debugging is harder (no shell — must use `--debug` variant or docker cp)
- `go:embed` requires the frontend build to complete before Go build, increasing build coupling

## Compliance

- `Dockerfile` always uses multi-stage build with `distroless/static` as final stage
- Frontend `dist/` embedded in Go binary via `//go:embed frontend/dist/*`
- No runtime dependencies on Node.js, Python, or build tools in production image
- New services requiring additional infrastructure add a `docker-compose.yml` entry
- Existing Compose services not removed without migration plan

## Notes

- Distroless base image: https://github.com/GoogleContainerTools/distroless
- Air hot-reload: https://github.com/air-verse/air
- Related ADRs: None directly (cross-cutting infrastructure decision)
