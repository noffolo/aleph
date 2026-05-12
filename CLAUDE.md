<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **aleph** (17633 symbols, 42173 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

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

<!-- graphify:start -->
# Graphify — Cognitive Map

This project has a persistent knowledge graph built by Graphify. Load the cognitive map at session start before exploring unfamiliar code.

## Session Start Protocol

**ALWAYS do this first in any new session:**

1. **Read the graph report**: `graphify-out/GRAPH_REPORT.md` — god nodes, community map, surprising connections, suggested questions (69KB, ~500 lines)
2. **For structured queries**: `graphify-out/graph.json` — 6,341 nodes, 10,988 edges, 45 labeled communities
3. **For visual navigation**: `graphify-out/graph.html` — interactive graph (aggregated community view, 471 nodes)

## How to Use

| Goal | Action |
|------|--------|
| "Where is X in the architecture?" | Read GRAPH_REPORT.md → find the community → navigate to source files |
| "What connects Y to Z?" | Read GRAPH_REPORT.md "Surprising Connections" section |
| "What's the most central component?" | Check GRAPH_REPORT.md "God Nodes" section |
| Trace a cross-community path | Open graph.html and navigate visually |

## Key Communities (Top 12 by size)

| Community | Nodes | Description |
|-----------|-------|-------------|
| C0 | 116 | API Client Layer (Connect RPC clients) |
| C1 | 78 | Frontend Views & Components |
| C2 | 75 | gRPC Service Handlers |
| C3 | 74 | Form Schemas & Validation |
| C4 | 74 | App Shell & Layout |
| C5 | 73 | UI Widgets & Error Boundaries |
| C6 | 64 | Error Handling & API Errors |
| C7 | 63 | Store & Protocol Encoding |
| C8 | 62 | Protobuf Adapters & Forms |
| C9 | 62 | Graph & Versioning |
| C10 | 62 | Finance Tools Testing |
| C11 | 57 | Registry Protobuf |

**God Nodes** (highest betweenness): `NewError()` (degree=145, connects 20+ communities), `NewRequest()` (degree=87), `useStore` (degree=77), `t()` (degree=65), `MetadataRepository` (degree=45).

## Updates

To rebuild the graph after significant code changes:
```bash
npx graphify /Users/ff3300/Desktop/aleph-v2 --update --no-viz
```
<!-- graphify:end -->