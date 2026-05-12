# Architecture Decision Records (ADR)

This directory contains Architecture Decision Records (ADRs) for the Aleph Data OS project.

ADRs document architectural decisions and their context, consequences, and compliance criteria. They serve as a historical record of why the system is designed the way it is.

## Format

Each ADR follows the [MADR (Markdown ADR)](https://adr.github.io/madr/) format:

- `# ADR-NNNN: Title`
- `## Status` — Accepted, Proposed, Deprecated, Superseded
- `## Context` — Problem description and driving forces
- `## Decision` — The decision made
- `## Consequences` — Positive and negative tradeoffs
- `## Compliance` — How to verify the decision is followed
- `## Notes` — References, links, related ADRs

## Index

| # | Title | Status | Area |
|---|-------|--------|------|
| [ADR-0001](0001-duckdb-postgresql-dual-storage.md) | DuckDB + PostgreSQL Dual Storage | Accepted | Backend / Storage |
| [ADR-0002](0002-connectrpc-over-http2.md) | ConnectRPC over HTTP/2 | Accepted | Backend / API |
| [ADR-0003](0003-server-sent-events-for-real-time-updates.md) | Server-Sent Events for Real-Time Updates | Accepted | Backend / API |
| [ADR-0004](0004-zustand-driven-view-routing.md) | Zustand-Driven View Routing (No React Router) | Accepted | Frontend / Architecture |
| [ADR-0005](0005-zod-frontend-validation-fromproto-mappers.md) | Zod for Frontend Validation + fromProto Mappers | Accepted | Frontend / Validation |
| [ADR-0006](0006-paora-decision-engine.md) | PAORA Decision Engine (Plan → Act → Observe → Reflect → Admit) | Accepted | Backend / Orchestration |
| [ADR-0007](0007-duckdb-vss-vector-similarity-search.md) | DuckDB VSS for Vector Similarity Search | Accepted | Backend / Storage |
| [ADR-0008](0008-argon2id-aes256gcm-security.md) | Argon2id + AES-256-GCM for Security | Accepted | Backend / Security |
| [ADR-0009](0009-docker-buildkit-multistage-build.md) | Docker + BuildKit Multi-Stage Build | Accepted | Infrastructure / Build |
| [ADR-0010](0010-rbac-jwt-bearer-authentication.md) | RBAC + JWT Bearer Authentication | Accepted | Backend / Security |

## Status Key

- **Accepted** — Decision made and implemented (or currently being implemented)
- **Proposed** — Under review, not yet decided
- **Deprecated** — Still valid but no longer recommended for new work
- **Superseded** — Replaced by a newer ADR (listed in Notes)

## Decision-Making Process

1. Identify the need for a decision (new feature, refactoring, tech debt)
2. Draft the ADR describing context, options, and recommendation
3. Review with stakeholders
4. Revise and mark as Accepted
5. Implement and update Compliance section as needed
6. Link related ADRs to maintain the decision graph

## License

These ADRs are part of the Aleph Data OS project documentation.
