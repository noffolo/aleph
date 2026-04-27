# AGENTS.md — Aleph-v2 Agent Map

## Current Status (27 Apr 2026)

- W0-W6: Complete (W0-W3 100%, W4-W5 100%, W6 80%)
- Residual: ERR ✅ A11Y ✅ PERF ✅ DEPLOY ⏳ DOCS ⏳
- Build: go build ✅ | vite build ✅

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
- `plans/` — Execution plans
- `docs/` — Architecture, security, evaluation documents
