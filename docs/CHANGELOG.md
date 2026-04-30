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

### W4 — Production Gate: Core Systems (Apr 2026)
- **Sources ingestion**: GitHub, sitemap, JSON API, Google Sheets ingesters wired into Engine
- **MemoryStore**: Full VSS with array_cosine_similarity, 9 tests
- **File Watcher**: fsnotify auto-ingestion with debounce
- **Genesis Suggester**: 3 heuristic analysis passes (chat→ontology, tool usage→tool, query patterns)
- **fixPerformance**: 4 anti-pattern detectors (sequential HTTP, missing ctx cancellation, string concat, repeated reads)
- **NLP Polish**: 11 print() → logging.*() conversions

### W5 — CI/CD & Infrastructure
- CI: go test -race -count=1, go vet, removed wasteful ESLint install
- docker-compose: Ollama auto-pulls llama3 + nomic-embed-text
- Security workflow: gitleaks secrets scan
- Deploy workflow: tag-triggered (v*)

### W6 — Security Hardening
- **CSP**: removed 'unsafe-inline' from style-src, inline <style> → CSS files
- **Rate limiting**: extractClientIP() with X-Forwarded-For → X-Real-IP → RemoteAddr chain
- **CSRF middleware**: Origin/Referer validation, 5 tests
- **SSE auth**: already implemented, verified
- **Release checklist**: docs/release-checklist.md
- **NLP health port**: 8001 (consistent with docker-compose)
