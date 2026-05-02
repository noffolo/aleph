# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-05-02

### Added

- **PAORA Decision Engine** — Plan, Act, Observe, Reflect, Admit cycle built into the Go backend for every interaction.
- **Sandbox Isolation** — Hardened execution with namespaces, seccomp profiles, and cgroup limits for untrusted code.
- **RBAC Authorization** — Role-based access control with built-in admin, user, and readonly roles enforced at every layer.
- **NLP Sidecar** — Python gRPC service for entity extraction and text classification, feeding results directly into the pipeline.
- **Schema Registry** — Versioned schemas with backward-compatibility checks to keep producers and consumers aligned.
- **Repair Engine** — Automated detection and correction of data anomalies with human approval for destructive changes.
- **Monitoring** — Prometheus and Grafana dashboards for throughput, latency, error rates, and resource saturation.
- **Alerting** — Alert rules tied to metrics, with support for PagerDuty, Slack, and webhook notifications.

### Changed

- **DuckDB Concurrency Rewrite** — Replaced single-writer model with multi-tenant connection pool and optimistic locking.

### Fixed

- **SQL Injection Vulnerabilities** — Parameterized all dynamic queries and added strict input validation on identifiers.

### Security

- **JWT Hardening** — Short-lived tokens, audience validation, and secure refresh rotation to tighten authentication.
- **CSP Strict Mode** — Hardened Content Security Policy on all frontend routes to mitigate XSS injection.

[2.0.0]: https://github.com/noffolo/aleph/releases/tag/v2.0.0
