# Aleph-v2 Repository Analysis & Tool/Skill Development Plan

## TL;DR

> **Quick Summary**: Deep analysis of 15+ repositories across 6 categories reveals that MCP (Model Context Protocol) is the optimal path for Aleph-v2 tool/skill enhancement. OpenBB Platform, Great Expectations, and Ghidra already have production MCP servers—offering ready-to-adopt patterns. Immediate priority: implement Python MCP server wrapping NLP sidecar + proxy to external MCP servers.
> 
> **Deliverables**: 
> - Python MCP server implementation (`nlp/mcp_server.py`)
> - 5‑7 core tools (prophet_forecast, analyze_sentiment enhanced, openbb_market_data proxy, etc.)
> - DSL‑based tool registration syntax
> - MCP‑aware skill composition engine
> 
> **Estimated Effort**: Large (6‑week phased rollout)
> **Parallel Execution**: YES — MCP server + tool wrapping + DSL integration
> **Critical Path**: MCP server → tool registration → skill composition → advanced pipelines

---

## Context

### Original Request
Deep analysis of 15+ repositories across 6 categories (finance/trading, data analysis, sentiment/NLP, reverse engineering, causality, orchestration) to identify tool/skill inspirations for Aleph‑v2's agents. **Constraint**: DO NOT modify Aleph itself unless critical; focus on developing/adapting NEW tools + skills for Aleph's agents.

### Aleph‑v2 Background
- Architecture: Go backend (Connect RPC, DuckDB), Python NLP sidecar (PyTorch, Prophet, XGBoost, LangChain), React/TypeScript frontend
- Current agent tools: `search_data`, `analyze_sentiment`, `get_trust_score` (hardcoded) + dynamic DB tools
- Skills: project‑scoped tool chains
- NLP: gRPC to Python sidecar (NLPAdapter, LLM Provider interface)
- Sandbox system: executes Python/Go code
- DSL system: `.aleph` ontology → DuckDB SQL
- **No MCP implementation** — custom function‑calling

### Research Methodology
6 parallel librarian/explore agents investigated 20+ repositories across categories:

1. **Finance/Trading** (5 repos): OpenBB Platform (MCP‑ready), Backtrader, Zipline‑reloaded, FinRL, Prophet
2. **Data Analysis** (4 repos): PyCaret, Great Expectations (GX has MCP server), YData‑Profiling, scikit‑learn
3. **Sentiment/NLP** (2 repos): Snorkel (weak supervision), BERTopic (topic modeling)
4. **Reverse Engineering** (4 repos): Ghidra (NSA SRE framework, community MCP servers with 222+ tools), Radare2, Cutter
5. **Causality/Orchestration** (5 repos): DoWhy (causal inference), CausalML (uplift modeling), MCP Protocol Spec (critical), LangGraph (stateful agent orchestration), CrewAI (multi‑agent orchestration)
6. **Aleph‑v2 internals**: explored current tool/skill system

---

## Work Objectives

### Core Objective
Enable Aleph‑v2 agents to discover, compose, and execute tools from a rich ecosystem via MCP (Model Context Protocol), while maintaining Aleph's unique ontology‑driven intelligence and sandbox security.

### Concrete Deliverables
- Python MCP server (`nlp/mcp_server.py`) exposing NLP/ML tools
- MCP‑compatible tool wrappers for 5‑7 core capabilities
- DSL syntax for tool registration (`tool prophet_forecast { ... }`)
- MCP‑aware skill composition engine
- Integration with external MCP servers (OpenBB, GX, Ghidra)

### Definition of Done
Text‑based acceptance criteria will be added in the TODO section (MCP server runs, tools respond, skill chains execute).

### Must Have
- Maintain Aleph ontology/DSL compatibility
- Keep sandbox security isolation
- No breaking changes to existing Connect RPC API
- Enable tool discovery and chaining

### Must NOT Have (Guardrails)
- No removal of existing Aleph functionality
- No direct modification of core Aleph code unless absolutely necessary
- No introduction of un‑sandboxed code execution
- Avoid AI‑slop: excessive abstraction, generic names, documentation bloat

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — All verification is agent‑executed. No acceptance criteria requiring “user manually tests/confirms.”

### Test Decision
-I see test infrastructure (Go tests + Python pytest). **Should we include automated tests for this MCP integration work?**

### QA Policy
Every task MUST include agent‑executed QA scenarios:
— Frontend/UI: Not applicable (backend‑only work)
— TUI/CLI: Use interactive_bash (tmux) — start MCP server, send requests via CLI client
— API/Backend: Use Bash (curl/nc) — send MCP protocol messages, assert responses
— Library/Module: Use Bash (Python scripts) — import MCP server, call functions

Evidence saved to `.sisyphus/evidence/task‑{N}‑{scenario‑slug}.{ext}`.

---

## Execution Strategy

### Parallel Execution Waves
**Wave 1 (Start Immediately — foundation)**: MCP server + core tools
**Wave 2 (After Wave 1 — integration)**: DSL registration + external MCP proxies
**Wave 3 (After Wave 2 — composition)**: Skill engine + advanced pipelines

Detailed dependency matrix and agent dispatch will follow in the TODO section.

---

## TODOs

---

## Final Verification Wave 
(Mandatory — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit “okay” before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
- [ ] F2. **Code Quality Review** — `unspecified‑high`
- [ ] F3. **Real Manual QA** — `unspecified‑high` (+ `playwright` skill if UI)
- [ ] F4. **Scope Fidelity Check** — `deep`

---

## Success Criteria

### Verification Commands
```bash
python nlp/mcp_server.py &
python -c "import asyncio; import mcp.client; ..."  # Test MCP client connection
# Expected: MCP server responds with tool list
```

### Final Checklist
.
[ ] Python MCP server runs and exposes tools
.
[ ] DSL can register new tools
.
[ ] Skill engine can compose MCP‑tool chains
.
[ ] External MCP servers (OpenBB, GX) are reachable via proxy
.
[ ] No existing Aleph functionality broken
