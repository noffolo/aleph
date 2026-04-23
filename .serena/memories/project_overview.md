# Aleph-v2: Decision Intelligence System

## Purpose
Ecosystem that transforms raw data flows into predictive scenarios. It's a "Data OS" with ontology-based data modeling, AI agents (Ollama), and prediction capabilities.

## Tech Stack
- **Backend**: Go 1.25, Connect RPC, DuckDB (analytics), PostgreSQL (system metadata)
- **NLP Sidecar**: Python (gRPC), ONNX/PyTorch, Prophet/XGBoost for ensemble predictions
- **Frontend**: React 18, TypeScript, Vite 5, Tailwind CSS, Zustand, D3, Leaflet
- **Orchestration**: Docker Compose

## Key Directories
- `main.go` - Standalone binary (embeds frontend)
- `cmd/aleph-server/` - Docker-deployable server (subset of services)
- `internal/api/handler/` - Connect RPC handlers
- `internal/api/proto/` - Generated protobuf/connect code
- `internal/dsl/` - Aleph DSL parser (participle) and SQL compiler
- `internal/storage/` - DuckDB + PostgreSQL wrappers
- `internal/ingestion/` - Data ingestion engine
- `internal/registry/` - DuckDB-based component registry
- `internal/auth/` - Auth service (mock)
- `internal/sandbox/` - Tool/skill execution sandbox (stub)
- `internal/predict/` - Brier monitor + factor manager (references NLP proto types)
- `nlp/` - Python gRPC sidecar for NLP/predictions

## Two Entry Points
1. `main.go` (root) - Full-featured app with all handlers, embedded frontend, PostgreSQL required
2. `cmd/aleph-server/main.go` - Docker-only server with Registry+Sandbox+Project only, DuckDB only

## DSL
Custom "Aleph" language parsed with `alecthomas/participle/v2`. Defines objects, properties, factors, relations, datasets. Compiler generates DuckDB SQL (read_parquet based).
