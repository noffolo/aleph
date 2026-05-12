# ADR-0007: DuckDB VSS for Vector Similarity Search

## Status

Accepted

## Context

Aleph Data OS requires vector similarity search for several workloads:

- **Memory retrieval**: Semantic search across agent memory vectors
- **Tool output search**: Find semantically similar tool execution results
- **Embedding-based classification**: Categorize data by embedding proximity
- **RAG queries**: Retrieve relevant context for LLM prompts based on embedding similarity

Options for vector storage and search:

| Option | Infra | Latency | Index Types | External Dep | Co-located |
|--------|-------|---------|-------------|--------------|------------|
| pgvector | PostgreSQL | Moderate | HNSW, IVFFlat | No | With metadata |
| Pinecone | Managed | Low | HNSW | Yes | No |
| Milvus | Self-hosted | Low | Multiple | Yes | No |
| DuckDB VSS | DuckDB | Moderate | HNSW | No | With tool data |
| Chroma | Embedded | Moderate | HNSW | Yes | No |

Aleph already uses DuckDB as its primary OLAP store (ADR-0001). Adding DuckDB VSS means vectors are stored alongside the data they describe — embedding vectors and the source tool outputs live in the same database, making cross-referencing simple and efficient.

## Decision

Use **DuckDB VSS** (Vector Similarity Search) extension for all vector similarity needs:

- Vectors stored as `FLOAT[]` columns in DuckDB tables alongside tool output data
- Similarity search via `array_cosine_similarity()` and `array_inner_product()` DuckDB functions
- HNSW indexes created via `CREATE INDEX USING hnsw ON table (column)`
- All VSS queries encapsulated in `internal/memory/` package for clean abstraction
- DuckDB VSS extension loaded at startup with `INSTALL vss; LOAD vss;`
- Embedding generation handled by the Python NLP sidecar or configured embedding providers (Ollama, OpenAI)

Key architectural rules:

1. Embeddings are stored in DuckDB tables, not a separate vector database
2. The `internal/memory/` package provides the query API: `FindSimilar(embedding, limit, threshold)`, `StoreVector(id, namespace, embedding)`, `DeleteVector(id)`
3. HNSW indexes are rebuilt after bulk inserts; incremental updates use HNSW's append capability
4. Dimensions are configured per-namespace to support multiple embedding models

## Consequences

### Positive
- No external vector database — zero additional infrastructure
- Data co-location: embedding vectors stored alongside source tool outputs in the same DuckDB database
- DuckDB columnar storage provides efficient vector scan for production workloads
- SQL-native search: `SELECT * FROM memory WHERE array_cosine_similarity(embedding, ?) > 0.8 ORDER BY 1 DESC LIMIT 10`
- Single migration system for both data and vectors

### Negative
- DuckDB single-writer mode limits concurrent embedding write throughput
- Fewer indexing options than pgvector (no IVFFlat in DuckDB VSS)
- Limited to embedding dimensions compatible with DuckDB's FLOAT[] type
- DuckDB VSS is a community extension — less mature than pgvector
- HNSW index builds require memory proportional to dataset size

## Compliance

- All vector similarity queries go through `internal/memory/` package
- Embedding columns are always `FLOAT[]` type
- HNSW indexing via `CREATE INDEX USING hnsw` on all vector columns
- DuckDB VSS extension loaded at startup (`INSTALL vss; LOAD vss;`)
- No external vector database dependencies imported
- Namespace isolation: each vector namespace uses its own DuckDB table or schema

## Notes

- DuckDB VSS extension: https://github.com/duckdb/community-extensions/tree/main/extensions/vss
- Embedding generation: `internal/nlp/` (Python sidecar) or `internal/embed/` (Go-native via Ollama)
- Related ADRs: ADR-0001 (DuckDB + PostgreSQL Dual Storage)
