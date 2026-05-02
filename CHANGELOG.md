# Changelog

All notable changes to the Aleph-v2 project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [2.0.0] — 2026-05-02

### Added

- **Decision Intelligence engine (PAORA)** — Plan, Act, Observe, Reflect, Admit cycle for every chat interaction, fully implemented in Go with no external orchestration.
- **Auto-repair engine** — 7 fix strategies for data anomalies: null values, outliers, duplicates, constraint violations, type errors, timestamp signatures, and correlations. Every fix is tracked and reversible.
- **Auto-suggestion engine (Genesis)** — Analyzes usage patterns and proposes new tools and skills. Each proposal passes through a sandbox validation block and a VetoRegistry with TTL expiry before activation.
- **Memory Store (VSS)** — Vector similarity search using DuckDB `list_cosine_similarity()` with per-project namespace isolation. Memories are injected into the PAORA context to improve relevance.
- **File System Watcher** — fsnotify-based directory monitoring with 500ms debounce for automatic file ingestion.
- **Data Ingestion Pipeline** — 7 dedicated fetchers: RSS/Atom, GitHub (issues and repo metadata), CSV/JSON upload, XML sitemap, Google Sheets, and IMAP email. Each fetcher includes SSRF-safe validation and sanitization.
- **NLP Sentiment sidecar** — Heuristic dictionary-based sentiment analysis (Italian and English) via Python gRPC. No transformer models are used for classification.
- **NLP Predictive endpoints** — `StreamPredictions` and `RecordFeedback` via gRPC, supporting Prophet, GBM, and market simulations. Flagged as experimental and synthetic-capable.
- **Workflow engine** — Base implementation for configurable agent workflows.
- **Circuit breaker middleware** — Subsystem protection with automatic recovery.
- **Rate limiting middleware** — Per-IP limiting with `X-Forwarded-For` → `X-Real-IP` → `RemoteAddr` fallback chain.
- **Request ID middleware** — Correlation IDs for distributed tracing.
- **CSRF middleware** — Origin and Referer validation with 5 test cases.
- **Terminal view** — Terminal-as-default layout with command/input mode and ghost prompt empty states.
- **Tool suggestion UI** — Real-time tool suggester integrated with the Genesis engine.
- **Scenario UI** — Predictive scenario visualization with confidence levels and explicit assumptions.
- **Monitoring stack** — Prometheus (:9090), Grafana (:3000), and Alertmanager (:9093) integrated via Docker Compose.
- **Playwright E2E tests** — Full end-to-end coverage for critical user flows.
- **Contract tests** — Go to Python NLP gRPC contract tests (build tag gated).
- **Design system tokens** — CSS volatility layers (`.vol-static`, `.vol-structural`, `.vol-interactive`, `.vol-signal`), glassmorphism, dark palette (`#080810`), and typography scale.
- **RBAC foundation** — Role-based access control layer for multi-user environments.

### Changed

- **API key encryption** — Transparent AES-256-GCM encryption at rest for all API keys. `KEY_ENCRYPTION_KEY` is now mandatory; `LoadConfig()` returns FATAL if missing.
- **API key hashing** — Migrated from SHA-256 to Argon2id for password-equivalent storage of API keys.
- **CSP policy** — Removed `unsafe-inline` from `style-src`; all inline styles migrated to CSS files.
- **Frontend architecture** — React.lazy code splitting into 3 chunks, SlideOver panel extraction, 4 form components refactored, and sidebar redesign.
- **NLP logging** — Converted 11 `print()` statements to proper `logging.*()` calls in the Python sidecar.
- **Performance optimization** — d3 lazy-loaded into a separate 439KB chunk; `GetDataStats` N+1 query replaced with batched fetch; `ToolCache` added with 5-minute TTL.
- **Docker Compose** — Ollama service now auto-pulls `llama3` and `nomic-embed-text` on startup.
- **CI pipeline** — Added `vitest` steps, aligned Go version, removed wasteful ESLint install step.

### Fixed

- **SQL injection vector** — Added parameterized queries and `validName()` regex validation on all identifier inputs.
- **TypeScript production errors** — Resolved all TS errors blocking production builds.
- **SSR redirect re-validation** — DNS resolution and redirect re-validation now block private IP ranges.
- **Sandbox escape** — Added command allowlist for `os/exec` isolation and package blocklist (`os/exec`, `syscall`, `unsafe`).
- **N+1 query in GetDataStats** — Replaced with batched query strategy.
- **Dead code removal** — Deleted `useViewActions.ts` and migrated 28 files to cleaner patterns.
- **Integration test failures** — Fixed 3 failing integration tests and stabilized cross-context test suite (820 lines).

### Security

- `KEY_ENCRYPTION_KEY` enforcement — AES-256-GCM encryption is mandatory for API key storage.
- Argon2id hashing — Replaced legacy SHA-256 for API key hashes.
- CORS restricted to explicit origins — No wildcard allowances.
- CSP hardened — No `unsafe-inline`, no `unsafe-eval`.
- SSRF guard — DNS resolution, redirect re-validation, and private IP blocking.
- Sandbox blocklist — Dangerous Go packages blocked from dynamic tool execution.
- Audit logging — All tool operations are logged with request IDs.
- Secrets scanning — Gitleaks runs on every CI build.

### Removed

- **Legacy SHA-256 API key storage** — Automatically detected and migrated to Argon2id.
- **Inline `<style>` tags** — All moved to external CSS files for CSP compliance.
- **Stale compiled binaries and pycache** — Removed from tracking and added to `.gitignore`.
- **Unused Zustand action files** — `useViewActions.ts` and related dead code eliminated.
- **Yjs dependency** — Cleaned up after deferred collaborative editing decision.

---

## [1.0.0] — 2026-04-19

Initial public release of Aleph-v2. Foundation commit with React/TypeScript frontend, Go backend scaffold, and Python NLP stub.

- Base project structure
- Design system tokens and lit styles
- API routing and TypeScript design system integration
- Docker Compose scaffold (6 services)

---
