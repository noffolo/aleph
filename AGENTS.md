# AGENTS.md — Aleph-v2 Agent Map

## Current Status (30 Apr 2026)

- **W0-W7: Complete** — All waves implemented and verified
- Build: `go build ./...` ✅ | `go test -race -count=1 ./...` ✅ | `go vet ./...` ✅
- Frontend: `npx tsc --noEmit` ✅ | `npx vite build` ✅ | `npx vitest run` ✅
- CI: GitHub Actions (Go + Frontend + Docker) | Security (gitleaks) | Deploy (tag-triggered)
- Docker: `docker compose config` ✅ with Ollama, PostgreSQL, NLP sidecar

## Build Agents
- **Sisyphus** (orchestrator) — Delegates work, verifies builds, ships waves
- **Sisyphus-Junior** — Focused task executor for well-scoped implementation units

## Subagent Types
| Agent | Purpose |
|-------|---------|
| `explore` | Codebase grep — find patterns, files, definitions |
| `librarian` | Reference grep — external docs, OSS examples |
| `oracle` | Read-only architecture/debug consulting |
| `visual-engineering` | Frontend UI/UX implementation |
| `ultrabrain` | Hard logic, architecture decisions |
| `deep` | Autonomous end-to-end problem solving |

## Agent Workflow
```
User Request → Sisyphus (classify + decompose) → Parallel subagents → Verify → Report
```

## State Files
- `.sisyphus/plans/` — Execution plans
- `docs/` — Architecture, security, evaluation documents
