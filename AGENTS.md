# AGENTS.md — Aleph-v2 Agent Map

## Current Status (17 May 2026)

- **All TDD cycles v1-v9: Complete** — Full TDD session across 9 plan iterations
- **Bugfix cycles 1-3: Complete** — app.go nil guards (eng/pg/db.Close), duckdb.go rollback error logs, NotificationService sync.Once double-close fix, engine.go fetchIMAP SSRF bypass fix, migration v9 broken index fix, Pre-commit hook `go vet` paths (Go 1.26), CLONE_NEWNET flag mismatch in namespace_isolated.go
- **Linux-phase tests:** namespace_isolation_error_test.go (//go:build linux) — correct function signatures for ExecuteIsolated (ctx, tmpDir string, cmd *exec.Cmd) and prepareSandboxedCmd (ctx, cmd *exec.Cmd, execID string)
- **Test coverage:** Go ~82.7% mean (all 46 packages with _test.go), Frontend 81%/75%/80%/82%
- **Dead code removed:** `adapters.ts` — all 6 `fromProto*` functions imported 0 times
- **Key bugs found & fixed:**
  - `ListComponents` filter in `duckdb_registry.go` is a silent no-op (SQL ignores WHERE clause)
  - `err != context.Canceled` — not using `errors.Is` (wrong comparison)
  - `watchSidecar()` nil `nlpHandler` panic (no nil guard on MarkUnhealthy/MarkHealthy)
  - `NotificationService.Stop()` double-close panic (close of closed channel)
  - `Engine.Close()` nil guard missing (a.eng.Close() panics if engine not initialized)
  - `fetchIMAP()` raw `tls.Dial(tcp, ...)` without SSRF validation
  - `_ = tx.Rollback()` swallowed errors in 3 DuckDB transaction rollback sites
  - Pre-commit hook `go vet` paths missing `./` prefix (Go 1.26 compatibility)
  - Postgres migration v9: broken index `idx_agents_project_status` referencing nonexistent `system_agents.status` column
  - `namespace_isolated.go` CLONE_NEWNET flag missing — test expected it, source didn't set it (broken linux namespace isolation)
- **E2E:** Playwright tests consolidated from orphaned `frontend/e2e/` → `frontend/tests/e2e/` (12 files)
- **Build:** `go build ./...` ✅ | `go test -race -count=1 ./...` ✅ | `go vet ./...` ✅ | `npx tsc --noEmit` ✅ (0 errors, was 19 before A2 fix) | `npx vitest run` ✅ (1358 tests, 81% stmts)
- **CI:** GitHub Actions (Go + Frontend + Docker + NLP) | Security (gitleaks) | Deploy (tag-triggered) | Pre-commit hooks: go-vet, tsc, vitest
- **Docker:** `docker compose config` ✅ with Ollama, PostgreSQL, NLP sidecar
- **GitNexus:** 23,032+ nodes, 56,758+ edges, 797+ clusters, 300 flows
- **Remaining (macOS-untestable):** seccomp/namespace Linux-only (~20% sandbox), VerifyTool integration path (~9% requires Docker), NewAlephApp/Serve integration tests (require Postgres), GOOS=linux go vet block by go-duckdb CGO dependency

### Honest Assessment (17 May 2026)

**Solid:** Build clean (go/vet/tsc/vitest all green). Go coverage 82.7% mean (43/46 packages tested). Front-end 81% stmts, 0 tsc errors (fixed 19 inherited). Real bugs found & fixed: data corruption (ListComponents filter), SSRF bypass (IMAP), crash (nlpHandler nil panic, NotificationService double-close), broken migration v9 index, broken Linux namespace isolation (CLONE_NEWNET). Dead code removed (adapters.ts). CI with 7 jobs + pre-commit hooks. Code intelligence: GitNexus 23K nodes, Graphify 13K nodes.

**Acceptable but improvable:** internal/app ~25% coverage (wiring/DI — integration-level, ~75% genuinely hard to unit-test). internal/sandbox ~59% (~20% Linux-only, ~9% Docker-only). ~80 front-end files without tests (mostly thin wrappers). No E2E in CI. 5 experimental packages (finance/osint/humanecosystems/gnn/dsl) are .opencode-ignore'd and not integrated.

**Missing:** No production deployment manifests (K8s/Helm/Terraform). No coverage gates in CI. Benchmarks exist but don't block CI. Contract tests need Postgres service container in CI. GOOS=linux go vet blocked by go-duckdb CGO. Graphify HTML viz too large (13K nodes > 5K limit). Seccomp tests process-destructive on Linux (t.Skip guards needed).

**Bottom line:** Project is solid for an actively-developed monolith. Most dangerous bugs have been found and fixed. Remaining coverage gaps are real platform limitations (macOS, no Postgres in CI) or deployment infrastructure. Next high-impact steps: Linux seccomp/namespace execution, E2E in CI, deploy infra.

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

This project is indexed by GitNexus as **aleph** (23032 symbols, 56758 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

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
