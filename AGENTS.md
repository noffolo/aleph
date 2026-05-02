# AGENTS.md — Aleph-v2 Agent Map

## Current Status (30 Apr 2026)

- **W0-W7: Complete** — All waves implemented and verified
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
- `docs/` — Architecture, security, evaluation documents