# AGENTS.md — Aleph-v2 Agent Map

## Current Status (16 May 2026)

- **W0-W7: Complete** — All waves implemented and verified
- **TDD Session Plan v2** — `docs/superpowers/plans/2026-05-16-tdd-session-plan-v2.md`
  - Reviewed by Metis + Oracle + Momus before execution
  - Key bug found: `ListComponents` filter in `duckdb_registry.go` is a silent no-op (SQL at line 129 ignores WHERE clause)
  - Key dead code found: `adapters.ts` (frontend/src/api/adapters.ts) — all 6 `fromProto*` functions imported 0 times; 28 consumers import factory.ts directly
  - Plan scope: 4 real tasks (CI/CD NLP, ListComponents bug fix, fromProto edge cases, factory.ts smoke test)
- Build: `go build ./...` ✅ | `go test -race -count=1 ./...` ✅ | `go vet ./...` ✅
- Frontend: `npx tsc --noEmit` ✅ | `npx vite build` ✅ | `npx vitest run` ✅
- CI: GitHub Actions (Go + Frontend + Docker) | Security (gitleaks) | Deploy (tag-triggered)
- Docker: `docker compose config` ✅ with Ollama, PostgreSQL, NLP sidecar

---

## Build Agents

Build agents are the permanent residents of the system. They coordinate work and execute tasks.

| Agent | Role | Responsibility |
|-------|------|---------------|
| **Sisyphus** | Orchestrator | Classifies incoming requests, decomposes work, delegates to subagents in parallel, verifies builds, and ships completed waves. |
| **Sisyphus-Junior** | Executor | Focused task runner for well-scoped implementation units. Handles single-file edits, small refactors, and atomic changes without orchestration overhead. |

---

## Subagent Types

Subagents are spawned on demand to handle specific domains. Each type has a relative cost based on token consumption and latency.

| Agent | Purpose | Cost |
|-------|---------|------|
| `explore` | Codebase grep — finds patterns, files, symbols, and definitions across the repository. | **FREE** |
| `librarian` | Reference grep — looks up external documentation, OSS examples, and API references. | **CHEAP** |
| `oracle` | Read-only architecture and debug consulting. Inspects code, traces logic, and answers questions without making changes. | **CHEAP** |
| `metis` | Strategic planning and task decomposition. Breaks complex requirements into actionable, ordered steps with dependency mapping. | **EXPENSIVE** |
| `momus` | Adversarial review and edge-case detection. Challenges assumptions, finds flaws, and proposes tests to prove vulnerabilities. | **CHEAP** |

---

## Task Categories

Use the `category` parameter to route tasks to the right cognitive profile.

| Category | When to Use |
|----------|-------------|
| `visual-engineering` | Frontend UI/UX implementation, component styling, layout fixes, Tailwind class tuning, and design-token adherence. |
| `ultrabrain` | Hard logic, architecture decisions, algorithm design, concurrency reasoning, and trade-off analysis. |
| `deep` | Autonomous end-to-end problem solving that requires exploration, multiple file edits, and verification. |
| `quick` | Fast, lightweight tasks: typo fixes, single-line changes, simple greps, or status checks. |
| `unspecified` | General-purpose tasks that do not fit a specific category. Safe default when unsure. |
| `writing` | Documentation, READMEs, commit messages, user guides, and any prose-heavy technical writing. |

---

## Agent Workflow

```
User Request
    |
    v
Sisyphus (classify + decompose)
    |
    v
Parallel Subagents (explore / librarian / oracle / metis / momus)
    |
    v
Implementation (Sisyphus-Junior or deep agent)
    |
    v
Verify (build, test, lint)
    |
    v
Report
```

---

## Adding a New Agent or Subagent

1. **Define the agent** in this file under the correct section (Build Agent or Subagent Type).
2. **Add a skill** (optional but recommended):
   - Create `docs/skills/<agent-name>/SKILL.md` with domain-specific instructions.
   - Reference it in the agent table above via a link.
3. **Register in orchestration logic**:
   - If the agent requires custom spawn logic, update the task router in `.sisyphus/config.yml` (or equivalent orchestration layer).
4. **Document costs**:
   - Mark the agent as `FREE`, `CHEAP`, or `EXPENSIVE` based on typical token usage and latency.
5. **Update tests**:
   - If the agent introduces new file patterns or build steps, add a smoke test to `internal/agents/` (or the relevant test suite).

---

## Skill Reference

Available skill directories under `docs/skills/` and `.config/opencode/skills/`:

| Skill | Scope | Trigger |
|-------|-------|---------|
| `research` | Preliminary research and outline generation | Academic or benchmark research |
| `plan` | PM planning — requirements, stack, task decomposition | Multi-step features |
| `ultrawork` | High-quality 5-phase development workflow | Complex implementations |
| `debug` | Structured bug diagnosis and fixing | Test failures, unexpected behavior |
| `brainstorm` | Design-first ideation before implementation | New features, UI changes |
| `review` | Full QA pipeline — security, performance, a11y, quality | Pre-merge checks |
| `code-reviewer` | PR diff review — bugs, smells, architecture | Code review |
| `scm` | Git operations — branching, merge, conventional commits | Git workflows |
| `frontend-design` | Production-grade frontend interfaces | Components, pages, styling |
| `golang-pro` | Go concurrency, microservices, performance | Go backend work |

For a full list of registered skills, run:

```bash
ls docs/skills/ .config/opencode/skills/ 2>/dev/null | sort -u
```

---

## State Files

- `.sisyphus/plans/` — Execution plans and wave definitions
- `docs/superpowers/plans/` — TDD session plans and feature plans
- `docs/` — Architecture, security, evaluation documents

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **aleph** (22872 symbols, 56479 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/aleph/context` | Codebase overview, check index freshness |
| `gitnexus://repo/aleph/clusters` | All functional areas |
| `gitnexus://repo/aleph/processes` | All execution flows |
| `gitnexus://repo/aleph/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
