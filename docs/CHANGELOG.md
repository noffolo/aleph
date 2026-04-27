# Changelog

All notable changes to the Aleph-v2 project are documented in this file.

## [Unreleased]

### Waves Completed

### W0 — Sopravvivenza (Build & Environment)
- Build recovery and environment hardening
- Go build pipeline fixed, CI/CD setup
- 17/18 items completed

### W0.5 — Epistemic Integrity
- Data source validation and metadata integrity
- 5/5 items completed

### W1 — Struttura (Project Structure)
- Codebase organization and module boundaries
- 11/12 items + benchmark + plan

### W2 — Onestà Profonda (Deep Honesty)
- Decision engine, DuckDB transactions, bias framework
- 7/8 items completed (GNN deferred)

### W3 — Resilienza (Resilience)
- CI/CD, linting, unit tests, OpenTelemetry, error messages
- Timeout/retry/bulkhead, audit logging, SHA-256 checksum
- Sandbox isolation, tool metadata, health check system
- MCP discovery engine, auto-diagnostic
- 17/17 items completed

### W4 — Voce (Voice/UI)
- CSS design tokens, typography, dark palette #080810
- Glassmorphism, border-radius, CSS volatility layers
- Command/Input mode, ghost prompt, terminal effects
- SlideOver panel, React.lazy code splitting, sidebar refactor
- Tool suggestion engine, versioning rollback
- 20/20 items completed

### W5 — Accoglienza (Welcome)
- DataSourceForm 3-step wizard, split view + search + export
- Terminal effects toggle, command palette Tab integration
- GetDataStats N+1→batched optimization, app.go integrations
- 12/12 items completed

### W6 — Autocoscienza (Self-Awareness)
- Dead code removal (useViewActions.ts deleted, 28 files migrated)
- Cursor pagination, bundle budget, Playwright setup
- Cross-context tests (820 lines), shadcn/ui components
- SSE, bias checklist, tool lifecycle E2E, MCP connectivity
- 12/15 items completed (i18n, URL state, Yjs cleanup deferred)

### Residual Waves
- W-ERR: Toast notification system, panic recovery middleware, error wrapping
- W-A11Y: Skip-link + landmarks, focus trap modals, aria-labels views
- W-PERF: d3 lazy-loaded (separate chunk 439KB), ToolCache 5min TTL, React.memo
- W-DEPLOY: Docker HEALTHCHECK, CI docker-push, .env.example
