# Version History

A concise timeline of Aleph releases with dates and key highlights.

---

## v2.0.0 — 2026-05-02

**Status:** Production

The first production-ready release. Completes the core Decision Intelligence platform with decision engine, auto-repair, memory, monitoring, security hardening, and a redesigned UI.

**Highlights:**
- PAORA decision intelligence engine (Plan, Act, Observe, Reflect, Admit)
- Auto-repair engine with 7 fix strategies
- Genesis auto-suggestion with sandbox and veto registry
- Memory Store (VSS) with DuckDB vector similarity
- 7 data ingestion fetchers (RSS, GitHub, CSV/JSON, sitemap, sheets, email)
- File system watcher with debounce
- NLP sentiment sidecar (heuristic IT/EN)
- NLP predictive endpoints (Prophet, GBM, simulations)
- Circuit breaker, rate limiting, request ID, and CSRF middleware
- Terminal-as-default UI with command palette and glassmorphism
- Prometheus + Grafana + Alertmonitoring stack
- Playwright E2E tests
- Hardened security: AES-256-GCM, Argon2id, CSP, SSRF guard, sandbox blocklist

**Commits:** 50+ since v1.0.0

---

## v1.0.0 — 2026-04-19

**Status:** Initial Release

Foundation release. Scaffolded the full-stack architecture with React/TypeScript frontend, Go backend, Python NLP sidecar, and Docker Compose orchestration.

**Highlights:**
- Base project structure
- React 18 + TypeScript 5 + Vite + Tailwind CSS frontend
- Go backend with Connect RPC and REST handlers
- Design system tokens and lit styles
- API routing and TypeScript integration
- Docker Compose with 6 services
- DuckDB + PostgreSQL storage layer

**Commits:** 4 (initial commit through stabilization)

---

## Pre-release History

Aleph-v2 development began in April 2026 as a ground-up rewrite of earlier experimental work. The project uses a wave-based development model (W0 through W7) documented in the repository plans.

| Wave | Theme | Period | Focus |
|---|---|---|---|
| W0 | Survival | Apr 2026 | Build recovery, environment hardening, CI/CD |
| W0.5 | Epistemic Integrity | Apr 2026 | Data validation, metadata integrity |
| W1 | Structure | Apr 2026 | Codebase organization, module boundaries |
| W2 | Deep Honesty | Apr 2026 | Decision engine, bias framework, DuckDB transactions |
| W3 | Resilience | Apr 2026 | CI/CD, linting, tests, sandbox, health checks, MCP |
| W4 | Voice/UI | Apr 2026 | Design system, terminal UI, tool suggester |
| W5 | Welcome | Apr 2026 | Forms, split views, search, export, terminal effects |
| W6 | Self-Awareness | Apr 2026 | Dead code removal, pagination, Playwright, E2E |
| W7 | Production | Apr–May 2026 | Security hardening, monitoring, release prep |

All waves are complete as of v2.0.0.

---

*Last updated: 2026-05-02*
