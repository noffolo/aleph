# ADR-0001: DuckDB + PostgreSQL Dual Storage

## Status

Accepted

## Context

Aleph Data OS ingests, stores, and queries both operational metadata and analytical data. These two categories have fundamentally different requirements:

- **Analytical / OLAP workloads**: Tool execution outputs, embedding vectors, session data, vector similarity search, tool registry queries. High-throughput reads, columnar access patterns, large datasets.
- **Operational / OLTP workloads**: User accounts, project configurations, audit logs, long-term configuration. ACID compliance required, frequent single-row CRUD, strong consistency.

A single-database approach forces compromises. PostgreSQL with pgvector can handle vector search but degrades under analytical query volume. DuckDB is a columnar OLAP database that excels at analytical queries but lacks multi-writer ACID support and is not suited for operational metadata.

Additionally, DuckDB's VSS (Vector Similarity Search) extension via `array_cosine_similarity()` provides zero-infrastructure vector search without adding an external vector database. This co-locates embeddings with the data they describe.

## Decision

Use a dual-database architecture:

1. **DuckDB** — Primary OLAP store. Handles tool execution data, embedding vectors, session data, tool registry, and all analytical/query workloads. Vector similarity search via DuckDB VSS extension with HNSW indexing. Single-writer mode with `sync.RWMutex` at the application layer.

2. **PostgreSQL** — Metadata store. Handles user accounts, projects, audit logs, long-term configuration, and any data requiring strict ACID compliance with proper indexing for CRUD patterns.

Both databases are initialized via separate migration systems:
- `internal/migration/` — DuckDB schema migrations
- `internal/migration/postgres/` — PostgreSQL schema migrations

The Go application opens and manages both connections at startup. The storage layer abstracts access: `internal/storage/duckdb/` for DuckDB operations, `internal/storage/postgres/` for PostgreSQL operations.

Cross-database joins are not possible at the database level. Any query combining data from both stores must be performed in application code by querying each database separately and merging results in Go.

## Consequences

### Positive
- Query isolation: analytical workloads do not lock or contend with operational queries
- Each database optimized for its workload pattern (columnar vs row-oriented)
- DuckDB VSS co-locates embeddings with source data, avoiding external vector DB
- PostgreSQL provides reliable ACID guarantees for user-facing data
- No external vector database dependency

### Negative
- Dual migration management — schema changes must be coordinated across two systems
- Cross-database joins impossible at DB level; application-layer merging required
- DuckDB single-writer mode necessitates `sync.RWMutex` in application code, limiting concurrent write throughput
- Two database connections consume more resources than one
- Operational complexity of maintaining both databases

## Compliance

- All analytical/query data → DuckDB (`internal/storage/duckdb/`)
- All user/project/config data → PostgreSQL (`internal/storage/postgres/`)
- Migration files go in the appropriate directory (`internal/migration/` or `internal/migration/postgres/`)
- No cross-database JOINs in SQL; use application-layer merging
- DuckDB write operations must use the shared mutex to respect single-writer mode

## Notes

- DuckDB VSS extension loaded at startup via `INSTALL vss; LOAD vss;`
- Related ADRs: ADR-0007 (DuckDB VSS for Vector Similarity Search)
