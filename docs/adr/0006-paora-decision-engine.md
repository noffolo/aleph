# ADR-0006: PAORA Decision Engine (Plan → Act → Observe → Reflect → Admit)

## Status

Accepted

## Context

Aleph agents need an orchestration loop that can autonomously execute multi-step tasks by using tools, processing results, and adapting to outcomes. A simple tool-calling loop ("call tool → get result → call next tool") is insufficient because it lacks:

- **Planning**: Breaking complex tasks into coordinated sub-steps before execution
- **Context accumulation**: Building understanding across multiple tool calls
- **Error recovery**: Adapting when a tool call fails or returns unexpected results
- **Completion detection**: Knowing when a task is fully resolved vs. requiring iteration

Options considered:

| Option | Flexibility | Complexity | External Dep | Observable |
|--------|------------|------------|--------------|------------|
| Simple tool loop | Low | Low | No | Limited |
| LangGraph | High | High | Yes | Via LangSmith |
| PAORA (custom) | High | Medium | No | Full (SSE per phase) |

## Decision

Implement **PAORA** as a 5-phase iterative execution loop at `internal/decision/`:

```
Plan → Act → Observe → Reflect → Admit
```

### Phase Definitions

1. **Plan** — Given a task, generate an execution plan with ordered steps, tool selections, and success criteria. Uses the configured LLM provider to decompose the task.

2. **Act** — Execute the planned steps. Each step calls a registered tool from the tool registry. Steps may be sequential or parallel depending on the plan. Results are captured as structured observations.

3. **Observe** — Collect and structure the results from tool execution. Normalize tool outputs, capture errors, and prepare data for reflection.

4. **Reflect** — Analyze observed outcomes against the original goal. Determine if progress was made, if corrections are needed, or if the approach should change. Produces a reflection score and recommended next action (continue, retry, replan, or terminate).

5. **Admit** — Declare the task complete, failed, or requiring iteration. If the task is complete, produce the final consolidated result. If not, loop back to Plan with accumulated context from prior iterations.

### Configuration
- `MaxSteps` — Hard limit on total tool calls per task (prevents infinite loops)
- `MaxReflectDepth` — Maximum reflection recursion per observation
- `MinConfidence` — Confidence threshold for early completion
- Per-phase config objects allowing independent tuning

### Observability
Each phase emits structured SSE events for frontend consumption: `phase:plan`, `phase:act`, `phase:observe`, `phase:reflect`, `phase:admit`.

## Consequences

### Positive
- Full control over loop behavior — no external orchestration framework
- Each phase is independently testable with mock tool registry entries
- Observable via SSE events — frontend shows "thinking" per phase
- Admit phase provides clear task completion/failure semantics
- Reflection enables error recovery without manual intervention

### Negative
- Custom implementation — no community patterns, edge cases discovered over time
- Complexity grows with each new phase or phase interaction
- LLM dependency for Plan and Reflect phases means quality varies by model
- State management across iterations requires careful context serialization
- Reflection overhead may add latency to simple tasks

## Compliance

- All autonomous agent execution loops use PAORA via `internal/decision/`
- New phases added as `Step` interface implementations
- Each phase has independent unit tests with mock tool registry
- All phases emit typed SSE events for observability
- Configuration exposed via `DecisionConfig` struct; no hardcoded step limits

## Notes

- Implementation at `internal/decision/` with sub-packages per phase
- Tool registry integration via `internal/tools/registry.go`
- SSE events for PAORA phases handled by `internal/api/sse/`
- Inspired by OODA loop (Observe-Orient-Decide-Act) adapted for LLM tool use
- Related ADRs: ADR-0003 (Server-Sent Events for Real-Time Updates)
