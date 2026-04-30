# Aleph-v2 Architecture

## Overview
Aleph-v2 is a data operating system combining Go backend (DuckDB + PostgreSQL), Python NLP sidecar, and React frontend.

## Backend (Go)
```
internal/
  app/           — Main wiring (AlephApp struct, Serve, Close)
  api/
    handler/     — HTTP + Connect RPC handlers
    proto/       — Protobuf definitions + generated code
    sse/         — Server-Sent Events broker
  config/        — Env-based configuration
  crypto/        — AES-256-GCM encryption
  decision/      — PAORA decision engine (Plan→Act→Observe→Reflect→Admit)
  diagnostic/    — Error pattern detection
  dsl/           — Domain-specific language compiler
  health/        — Periodic health checker
  ingestion/     — Data ingestion pipeline with sources (RSS, GitHub, CSV, JSON, sitemap, sheets, email)
  llm/           — Provider interface for LLMs (Ollama, OpenAI)
  mcp/           — MCP discovery engine
  memory/        — VSS memory store (DuckDB array_cosine_similarity)
  middleware/    — 8 middleware: Auth, Audit, CORS, CSRF, Rate-limit (X-Forwarded-For), Timeout, Bulkhead, Security (CSP)
  migrate/       — DuckDB + PostgreSQL migrations
  nlp_adapter/   — Python sidecar adapter
  predict/       — Brier score monitoring
  registry/      — DuckDB registry
  repair/        — Auto-repair engine (7 fix strategies, 1173 lines)
  repository/    — Metadata + audit persistence
  sandbox/       — Tool execution + verification
  service/
    watcher/     — File system watcher (fsnotify) for auto-ingestion
    notification/— Notification service
  storage/       — DuckDB + PostgreSQL connections
  telemetry/     — OpenTelemetry instrumentation
  tools/
    adaptation/  — Tool adaptation pipeline
    codeflow/    — Cross-document code flow
    finance/     — Finance tool stubs
    humanecosystems/ — Human ecosystems tools
    osint/       — OSINT tool stubs
    synthesis/   — Cross-context synthesis
```

## NLP Sidecar (Python)
```
nlp/
  ensemble.py    — Model ensemble (calibrated)
  main.py        — gRPC server with logging
  requirements.txt
```

## Frontend (React + TypeScript + Vite)
```
frontend/
  src/
    api/         — Connect RPC clients + hooks (12 services)
    components/  — UI components (Terminal, Copilot, CommandPalette, SlideOver, ...)
    store/       — Zustand slices (navigation, ui, health, auth, settings, workspace, copilot)
    views/       — Page-level components
    schemas/     — Zod validation schemas (22 schemas)
    lib/         — Utilities
    commands/    — Slash commands (16 built-in)
    styles/      — Design tokens, base CSS
```

## Data Flow
```
Client → HTTP/h2c → CORS → CSRF → Telemetry → Middleware stack → Handler → Repository/Service
                                         ↓
                              SSE Broker ← Health/MCP/Diagnostic monitors
```

## Key Decisions
1. **DuckDB** for analytics, **PostgreSQL** for system records — deliberate separation
2. **Connect RPC** (gRPC over HTTP/2) for typed API — no REST overhead
3. **SSE** for unidirectional server→client push — simpler than gRPC-Web for EventSource
4. **No React Router** — View switching via Zustand navigationSlice
5. **Zod schemas** for runtime validation on frontend
6. **PAORA loop** — Plan → Act → Observe → Reflect → Admit for every chat interaction
7. **VSS via DuckDB** — Vector similarity search built on native DuckDB array_cosine_similarity
