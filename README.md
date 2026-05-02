# Aleph-v2

> Transform raw data into structured intelligence through AI agents, sandboxed execution, and DuckDB.

[![CI](https://github.com/noffolo/aleph/actions/workflows/ci.yml/badge.svg)](https://github.com/noffolo/aleph/actions)
[![Go](https://img.shields.io/badge/go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/react-18-61DAFB?logo=react&logoColor=black)](https://react.dev)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](LICENSE)

## What is Aleph?

Aleph is a Data Operating System that turns unstructured and raw data into actionable, structured intelligence. It brings together AI agents, a secure sandboxed runtime, and high-performance DuckDB analytics to let you ingest, query, and reason over your data without heavy infrastructure overhead.

Built with Go, React, and Python, Aleph connects to your existing data sources through a modular ingestion pipeline. It stores metadata in PostgreSQL, runs analytics in DuckDB, and uses an NLP sidecar powered by modern language models to extract meaning, entities, and relationships automatically. Every tool call follows the PAORA cycle: Plan, Act, Observe, Reflect, Admit. This means execution is deliberate, observable, and safe by default.

## Quick Start

The fastest way to run Aleph is with Docker Compose. This spins up the backend, frontend, PostgreSQL metadata store, DuckDB analytics, and the NLP sidecar.

```bash
git clone https://github.com/noffolo/aleph.git
cd aleph
cp .env.example .env
# edit .env and set KEY_ENCRYPTION_KEY
docker compose up --build
```

Once the stack is healthy, open your browser at `http://localhost:5173`. The API is available at `http://localhost:8080`.

To run tests across the entire project:

```bash
go test -race -count=1 ./...
cd frontend && npx vitest run
```

## Architecture Overview

```
User
  |
  v
Web UI ............ React 18 · TypeScript · Vite · Tailwind CSS
  |
  v
Backend ........... Go · Connect RPC · SSE
  |
  |-- Data ......... DuckDB (analytical + VSS) · PostgreSQL 16 (metadata)
  |-- Intelligence . Python sidecar (NLP, Prophet, GBM) via gRPC
  |-- Observability  Prometheus :9090 · Grafana :3000 · Alertmanager :9093
  |-- Security ..... Argon2id · AES-256-GCM · SSRF guard · sandboxed execution
  |
  v
Docker Compose (6 services)
```

## Key Features

- **PAORA Decision Cycle** — Plan, Act, Observe, Reflect, Admit built directly into the Go backend. No external orchestration needed.
- **Sandboxed Agent Execution** — Every tool runs in isolation with a blocklist for dangerous packages and monitored resource usage.
- **Auto-repair Engine** — Detects and fixes 7 classes of data anomalies, including nulls, outliers, duplicates, broken constraints, bad types, stale timestamps, and correlations.
- **Genesis Auto-suggestion** — Proposes new tools and skills based on your usage patterns, with a human veto before activation.
- **Memory Store (VSS)** — DuckDB-backed vector similarity search with isolated namespaces per project.
- **Data Ingestion** — Pulls from RSS, GitHub, Google Sheets, CSV/JSON, sitemaps, and IMAP with SSRF-safe validation.
- **File System Watcher** — Auto-imports dropped files with a 500ms debounce.
- **NLP Sidecar** — Heuristic sentiment analysis (IT/EN) and experimental prediction streams via gRPC.
- **Production Hardening** — Circuit breakers, rate limiting, bulkhead pattern, graceful shutdown, and CSP without unsafe-inline.

## Example: Query the Memory Store

Once the system is running, you can recall relevant context using vector similarity. The VSS layer stores project-isolated embeddings inside DuckDB and surfaces them to the decision cycle.

```bash
curl -X POST http://localhost:8080/api/v1/memory/query \
  -H "Content-Type: application/json" \
  -d '{"query":"market risk signals Q3 2026","limit":5,"project_id":"proj-01"}'
```

This returns ranked memories with confidence scores, ready to be injected into the next PAORA cycle.

## Documentation

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — System design and data flow
- [`docs/API.md`](docs/API.md) — API contracts and protobuf definitions
- [`docs/CHANGELOG.md`](docs/CHANGELOG.md) — Release history
- [`SECURITY.md`](SECURITY.md) — Vulnerability reporting and security model

## License

Distributed under the GPL-3.0 license. See [`LICENSE`](LICENSE) for details.
