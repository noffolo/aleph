# Aleph-v2 — Context Map

## Project

**Aleph Data OS** — AI-augmented data operating system. Mono-repo at `/Users/ff3300/Desktop/aleph-v2/`.

| Layer | Language | LOC | Key Stack |
|-------|----------|-----|-----------|
| Backend | Go 1.26 | 79K prod + 38K test | ConnectRPC, DuckDB, PostgreSQL, 11 middleware layers |
| Frontend | TypeScript 5.5 | 24K | React 18, Vite 8, Zustand 4.5 (6 slices), TanStack Query, Tailwind 3.4 |
| NLP | Python 3.12 | 2.5K | gRPC, ONNX, transformers |
| API | Protobuf | 28 files | 5 services (Query/Registry/Notification/Sandbox/NLP) |

## Key Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| **ConnectRPC not REST** | All API via ConnectRPC (unary + streaming). No REST except SSE. |
| **No React Router** | View switching via Zustand `navigationSlice`. |
| **DuckDB primary + PG metadata** | DuckDB for storage+VSS. PG for metadata/audit. **Never mix them.** |
| **Dual DB migrations** | Separate `internal/migration/duckdb/` and `postgres/`. Different numbering. |
| **11 middleware layers** | Recovery→CSRF→Security→RequestID→Auth→RateLimit→Bulkhead→Timeout→Audit→CircuitBreaker→Retry |

## TDD Session (May 2026 — 15+ hours across v2-v9)

Active plans at `docs/superpowers/plans/2026-05-16-tdd-session-plan-v*.md`.

**Key outcomes:**
- `ListComponents` filter BUG: SQL ignored WHERE clause — fixed with dynamic WHERE builder
- `adapters.ts` DEAD CODE: all 6 `fromProto*` functions imported 0 times — removed
- CI: Python NLP `nlp-test` job added alongside Go/frontend
- 19 pre-existing frontend tsc errors in 6 test files — fixed (CopilotChat, DataSourceForm, SkillForm, ToolsView, useSSE, navigationSlice)
- Pre-commit hooks added (go-vet, tsc, vitest)
- `watchSidecar()` extracted with nil guards — exposed nlpHandler nil panic bug
- `NotificationService.Stop()` double-close — fixed with sync.Once
- `fetchIMAP()` SSRF bypass — raw tls.Dial without validation — fixed
- App nil guards: eng/pg/db.Close in `Close()` — added
- DuckDB rollback errors: `_ = tx.Rollback()` → `slog.Warn` (3 sites)
- Postgres migration v9: broken index on nonexistent column — removed
- CLONE_NEWNET: source was missing the flag, test expected it — fixed
- app_integration_test.go: integration test with real DuckDB + Postgres — created

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

- **GitNexus** — 22,991+ symbols, 56,693+ relations, 300 flows. **MUST run `gitnexus_impact()` before editing any symbol.**
- **Graphify** — 13,210 nodes, 21,640 edges, 814 communities. Rebuild: `graphify update /path/to/aleph-v2 --no-viz`. (graph.html skipped — too large for HTML viz)
- **OpenCode skills** → `~/.config/opencode/skills/` (126+ skills). Load via `skill(name="...")`.

## All GitNexus + Graphify references are now in CLAUDE.md

CLAUDE.md has the complete GitNexus Always Do/Never Do rules and the Graphify cognitive map reference. AGENTS.md has the full agent map. This file is the lightweight context map.

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
