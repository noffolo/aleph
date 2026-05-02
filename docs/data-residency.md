# Data Residency — Aleph-v2

## 1. Storage Architecture

Aleph-v2 uses three primary storage layers, each with distinct residency characteristics.

### 1.1 PostgreSQL (Metadata)

| Aspect | Detail |
|---|---|
| **Engine** | PostgreSQL 16 |
| **Data stored** | Project records, agent configs, API key hashes, audit logs, chat history, task/skill metadata |
| **Connection** | Configurable via `DATABASE_URL` environment variable |
| **Default location** | Local Docker container (`aleph-db`) |
| **Persistence** | Docker volume `pgdata` |
| **Backup** | Deployer responsibility; `pg_dump` recommended |

### 1.2 DuckDB (Analytical Data)

| Aspect | Detail |
|---|---|
| **Engine** | DuckDB (embedded) |
| **Data stored** | User-created data tables, vector embeddings, ontology snapshots |
| **Connection** | File-based, controlled by `DUCKDB_PATH` env var |
| **Default location** | `aleph.db` in working directory |
| **Persistence** | Single file (can be moved/copied) |
| **Backup** | File copy; DuckDB supports `.backup` command via SQL |

### 1.3 Filesystem Storage

| Aspect | Detail |
|---|---|
| **Data stored** | Raw uploads (CSV, JSON), ontology files, backup snapshots |
| **Default location** | `projects/` directory, configurable via `PROJECTS_ROOT` env var |
| **Backup** | Filesystem snapshot or rsync |

## 2. Backup Policies

### 2.1 Recommended Backup Strategy

```bash
# PostgreSQL backup
pg_dump -h localhost -U aleph -d aleph > aleph_pg_$(date +%Y%m%d).sql

# DuckDB backup
cp aleph.db aleph.db.$(date +%Y%m%d)

# Filesystem backup
tar czf projects_$(date +%Y%m%d).tar.gz projects/
```

### 2.2 Recovery Point Objective (RPO)

| Layer | Recommended RPO |
|---|---|
| PostgreSQL | 24 hours |
| DuckDB | 24 hours (or per-ETL cycle) |
| Filesystem | 24 hours |

### 2.3 Recovery Time Objective (RTO)

| Scenario | Estimated RTO |
|---|---|
| PostgreSQL restore | 5-15 minutes (depends on volume) |
| DuckDB restore | 1-5 minutes |
| Full system recovery | 15-30 minutes |

## 3. Data Locality

### 3.1 Default Deployment

In the default Docker Compose deployment, all data resides on the local host:

```
Host filesystem
├── Docker volume `pgdata` → PostgreSQL data
├── <working-dir>/aleph.db → DuckDB data
└── <working-dir>/projects/ → File uploads & ontologies
```

### 3.2 Regional Deployment

PostgreSQL and DuckDB can be pointed at remote instances:

```env
# Remote PostgreSQL
DATABASE_URL=postgres://user:pass@db.eu-west-1.example.com:5432/aleph

# Remote DuckDB (NFS or S3-mounted)
DUCKDB_PATH=/mnt/nfs/aleph.db
```

Considerations for cross-region deployment:

- **Latency**: DuckDB is an embedded database designed for local access. Remote DuckDB on network filesystem may degrade performance for write-heavy operations.
- **Compliance**: Ensure data storage location complies with regional regulations (GDPR, CCPA, LGPD, etc.).
- **Backup**: Remote PostgreSQL should use managed backup services (RDS automated backups, Cloud SQL, etc.).

## 4. Data Encryption at Rest

| Layer | Default | Recommendation |
|---|---|---|
| PostgreSQL | None (deployer responsibility) | Enable `pg_data` encryption; use managed service with encryption-at-rest |
| DuckDB | None (deployer responsibility) | Filesystem-level encryption (LUKS, eCryptfs) |
| Filesystem | None (deployer responsibility) | Encrypted volume or filesystem |

The `KEY_ENCRYPTION_KEY` environment variable provides application-level encryption for sensitive fields (API keys in PostgreSQL) using AES-256-GCM. This is independent of storage-level encryption.

## 5. Data in Transit

| Path | Protocol | Encryption |
|---|---|---|
| Browser → Backend | HTTP/2 (h2c) or HTTPS | TLS when behind reverse proxy |
| Backend → PostgreSQL | TCP | TLS when configured (`sslmode=require`) |
| Backend → DuckDB | Local file | N/A (same host) |
| Backend → Ollama | HTTP | Localhost (127.0.0.1:11434) |
| Backend → NLP sidecar | gRPC | Docker internal network |
| Backend → External LLM | HTTPS | TLS (mandatory for OpenAI/Anthropic) |

## 6. Data Export / Portability

All data can be exported for migration or compliance purposes:

```bash
# Export PostgreSQL metadata
pg_dump --data-only -h localhost -U aleph -d aleph -f aleph_metadata.sql

# Export DuckDB data
# Method 1: Copy the file
cp aleph.db aleph_export.db

# Method 2: SQL export
echo "EXPORT DATABASE './aleph_export';" | duckdb aleph.db

# Export filesystem data
tar czf projects_export.tar.gz projects/
```

## 7. Disaster Recovery

### 7.1 Full Recovery Procedure

1. Restore PostgreSQL from backup:
   ```bash
   createdb aleph
   psql -h localhost -U aleph -d aleph < aleph_pg_backup.sql
   ```

2. Restore DuckDB from backup:
   ```bash
   cp aleph.db.backup aleph.db
   ```

3. Restore filesystem data:
   ```bash
   tar xzf projects_backup.tar.gz
   ```

4. Start services:
   ```bash
   docker compose up -d
   ```

### 7.2 Validation

After recovery, verify:

- `go build ./...` passes
- Health endpoint at `/healthz` returns 200
- Audit logs contain the recovery event (if manually logged)
- API keys are functional (test with `curl`)

## 8. Compliance Checklist

- [ ] `KEY_ENCRYPTION_KEY` is set for API key encryption at rest
- [ ] PostgreSQL connection uses TLS (`sslmode=require`)
- [ ] Backup schedule is configured (recommended: daily)
- [ ] Backup retention is configured (minimum: 30 days)
- [ ] Data residency requirements are documented for the deployment region
- [ ] Delete cascade is verified (test project deletion includes cleanup of all layers)
- [ ] Audit logging is enabled and retention is configured
