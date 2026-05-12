# Aleph-v2 — Context Map

## Project

**Aleph Data OS** — AI-augmented data operating system. Mono-repo at `/Users/ff3300/Desktop/aleph-v2/`.

| Layer | Language | LOC | Key Stack |
|-------|----------|-----|-----------|
| Backend | Go 1.26 | 79K prod + 27K test | ConnectRPC, DuckDB, PostgreSQL, 11 middleware layers |
| Frontend | TypeScript 5.5 | 24K | React 18, Vite 8, Zustand 4.5 (6 slices), TanStack Query, Tailwind 3.4 |
| NLP | Python 3.12 | 1.2K | gRPC, ONNX, transformers |
| API | Protobuf | 28 files | 5 services (Query/Registry/Notification/Sandbox/NLP) |

## Key Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| **ConnectRPC not REST** | All API via ConnectRPC (unary + streaming). No REST except SSE. |
| **No React Router** | View switching via Zustand `navigationSlice`. |
| **DuckDB primary + PG metadata** | DuckDB for storage+VSS. PG for metadata/audit. **Never mix them.** |
| **Dual DB migrations** | Separate `internal/migration/duckdb/` and `postgres/`. Different numbering. |
| **11 middleware layers** | Recovery→CSRF→Security→RequestID→Auth→RateLimit→Bulkhead→Timeout→Audit→CircuitBreaker→Retry |

## Quick Start

```sh
make dev              # Full stack (air + vite + NLP)
make dev-backend      # Go only (air hot reload)
make dev-frontend     # Vite only
make test-go          # go test -race -count=1 ./...
make test-frontend    # vitest + tsc --noEmit
make build            # Go binary + frontend build
```

## Code Intelligence

- **GitNexus** — 17,633 symbols, 42,173 relations, 300 flows. **MUST run `gitnexus_impact()` before editing any symbol.**
- **Graphify** — 6,341 nodes, 10,988 edges, 471 communities. Interactive graph at `graphify-out/graph.html`.
- **OpenCode skills** → `~/.config/opencode/skills/` (126 skills). Load via `skill(name="...")`.

## Wave Status (W0-W7 complete)

All waves verified: `go build ./...` ✅, `go test -race -count=1 ./...` ✅, `npx tsc --noEmit` ✅, `npx vite build` ✅.

## Session Rules

1. **Always** check `.opencode-ignore` before reading generated files (protobuf, node_modules, venv).
2. **Always** run `gitnexus_impact()` before editing functions/classes/methods.
3. **Always** run `gitnexus_detect_changes()` before committing.
4. **Always** verify with fresh diagnostics (`lsp_diagnostics`, build, test) before marking complete.
5. **Load skills before starting work**: `skill(name="aleph-debug")` for debugging, `skill(name="golang-pro")` for Go, `skill(name="react-expert")` for frontend.

## Skills Quick Reference

For debugging → `aleph-debug`
For Go backend → `golang-pro`, `database-optimizer`, `sql-pro`
For frontend → `react-expert`, `typescript-pro`, `frontend-design`
For Python NLP → `python-pro`
For security → `secure-code-guardian`, `security-reviewer`
For testing → `test-master`, `playwright-expert`
For process → `subagent-driven-development`, `work-strategy`, `verification-before-completion`

## Known Pain Points

- DuckDB RWMutex contention (~1000x slower under load) — use `database-optimizer` skill
- DB migration numbering mismatch (DuckDB 7, Postgres 8 with jump 001→003)
- W3 partial corruption from over-scoped agent — scope agents to single packages only
- `.opencode-ignore` excludes `finance/`, `osint/`, `humanecosystems/`, `synthesis/`, `genesis/`, `gnn/`, `dsl/`, `ethics/` — experimental packages
