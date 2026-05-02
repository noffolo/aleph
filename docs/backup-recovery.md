# DuckDB Backup & Recovery

> **Last updated:** May 2026
> **Scope:** DuckDB analytic store (`internal/storage/duckdb_backup.go`)
> **PostgreSQL backup:** See `scripts/backup-pg.sh` (separate system)

---

## 1. Backup Strategy

Aleph-v2 uses **two complementary backup mechanisms** for DuckDB:

| Mechanism | Method | Best for | Frequency |
|-----------|--------|----------|-----------|
| **File copy** | `CHECKPOINT` + `cp` | Quick, full-database snapshots | Default: every 24h |
| **EXPORT DATABASE** | DuckDB `EXPORT DATABASE` | Schema + data as Parquet (portable) | On-demand / pre-migration |

### 1.1 File-Copy Backup (`Backup()`)

- Runs `CHECKPOINT` to flush the write-ahead log to the main database file
- Copies the `.duckdb` file under the **write lock** (blocks all reads and writes during the copy)
- Produces a self-contained `.duckdb` file that can be opened directly by DuckDB
- **Limitation:** Includes internal fragmentation; not portable across DuckDB versions

### 1.2 EXPORT DATABASE (`ExportDatabase()`)

- Executes DuckDB's `EXPORT DATABASE` command under the write lock
- Produces a directory containing:
  - `schema.sql` — DDL to recreate all tables, views, and indexes
  - `load.sql` — COPY statements to load each table from Parquet files
  - `*.parquet` — One columnar Parquet file per table
- **Portable** across DuckDB versions and architectures
- All files are `fsync`'d for durability

### 1.3 Auto-Backup (`AutoBackup()`)

Runs in a background goroutine started by `AlephApp.Serve()`:

1. Creates an initial backup on startup
2. Creates a new backup at each `BACKUP_INTERVAL`
3. Retains up to `BACKUP_KEEP` recent backups (deletes oldest first)
4. Writes a `.meta` sidecar file alongside each backup with source path and timestamp
5. Stops cleanly when the application context is cancelled

---

## 2. RPO / RTO Targets

| Metric | Target | Notes |
|--------|--------|-------|
| **RPO (Recovery Point Objective)** | 24h | Configurable via `BACKUP_INTERVAL` |
| **RTO (Recovery Time Objective)** | < 5 min | File-copy restore; depends on database size |
| **RPO during active write** | Transaction-consistent | CHECKPOINT + write lock guarantees no partial transactions |

### RPO Tuning

| `BACKUP_INTERVAL` | RPO | Cost |
|-------------------|-----|------|
| 1h | 1h | Higher I/O, more backups retained |
| 6h | 6h | Balanced |
| 24h (default) | 24h | Low I/O overhead |
| 72h | 72h | Minimal I/O |

For workloads with frequent imports, set `BACKUP_INTERVAL=1h` or `BACKUP_INTERVAL=30m`.

---

## 3. Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `BACKUP_INTERVAL` | `24h` | Interval between automatic backups (Go duration) |
| `BACKUP_DIR` | `{DATA_ROOT}/../backups/duckdb` | Directory for backup files |
| `BACKUP_KEEP` | `7` | Number of recent backups to retain |

### Example configurations

**Production (conservative):**
```env
BACKUP_INTERVAL=6h
BACKUP_DIR=/data/backups/duckdb
BACKUP_KEEP=14
```

**High-frequency ingestion:**
```env
BACKUP_INTERVAL=30m
BACKUP_DIR=/data/backups/duckdb
BACKUP_KEEP=48
```

---

## 4. Recovery Runbook

### 4.1 File-Copy Restore (Standard)

**Use when:** The live `.duckdb` file is corrupted or lost, and you have a `.duckdb` backup file.

```
Step 1: STOP APPLICATION TRAFFIC
──────────────────────────────────
  docker compose stop backend

Step 2: VERIFY THE BACKUP
──────────────────────────
  # Using DuckDB CLI:
  duckdb /path/to/backup/aleph_backup_20260501T120000Z.duckdb -c "PRAGMA integrity_check;"
  # Expected output: "ok"

  # Alternative via VerifyBackup (restoring application must do this):
  # db.VerifyBackup(backupPath)

Step 3: MOVE ASIDE THE CORRUPT DATABASE
────────────────────────────────────────
  mv data/aleph.duckdb data/aleph.duckdb.corrupt

Step 4: COPY BACKUP TO LIVE PATH
─────────────────────────────────
  cp /path/to/backup/aleph_backup_20260501T120000Z.duckdb data/aleph.duckdb

Step 5: RESTART APPLICATION
────────────────────────────
  docker compose start backend

Step 6: VERIFY RESTORED DATABASE
─────────────────────────────────
  docker compose exec backend /app/aleph-v2 \
    -exec "duckdb-check"  # or query a known table via API

Step 7: RUN MIGRATIONS (if needed)
───────────────────────────────────
  # The backup may be from an older schema version.
  # The application runs migrations on startup automatically.
  # Verify they completed:
  docker compose logs backend | grep "migration"
```

### 4.2 EXPORT DATABASE Restore

**Use when:** You have a directory from `ExportDatabase()` (schema.sql + Parquet files).

```
Step 1: STOP APPLICATION TRAFFIC
──────────────────────────────────
  docker compose stop backend

Step 2: VERIFY THE EXPORT
──────────────────────────
  ls /path/to/export/
  # Expected: schema.sql  load.sql  *.parquet

  # Check schema.sql is non-empty:
  wc -l /path/to/export/schema.sql

Step 3: MOVE ASIDE THE OLD DATABASE
────────────────────────────────────
  mv data/aleph.duckdb data/aleph.duckdb.old

Step 4: CREATE EMPTY DUCKDB AT THE LIVE PATH
──────────────────────────────────────────────
  touch data/aleph.duckdb  # Will be populated by IMPORT

Step 5: IMPORT THE EXPORT
──────────────────────────
  duckdb data/aleph.duckdb -c "IMPORT DATABASE '/path/to/export/';"

Step 6: RESTART APPLICATION
────────────────────────────
  docker compose start backend

Step 7: VERIFY
──────────────
  docker compose exec backend duckdb data/aleph.duckdb \
    -c "SELECT COUNT(*) AS table_count FROM information_schema.tables WHERE table_schema='main';"
```

### 4.3 Point-in-Time Recovery (best-effort)

DuckDB does not support native PITR. The closest approximation:

1. Restore the most recent backup before the point of failure
2. Identify and re-ingest any data that arrived after that backup
3. Use application-level audit logs (`audit_events` table in PostgreSQL) to identify the window

---

## 5. Verification Procedures

### Automated verification

Each backup is automatically verified at creation time by:
1. **DuckDB PRAGMA integrity_check** — full internal consistency check
2. **Table count sanity** — ensures at least one table exists (non-empty DB)

Run on-demand:
```bash
duckdb /path/to/backup.duckdb -c "PRAGMA integrity_check;"
```

### Export verification

For EXPORT DATABASE directories, the `VerifyExportBackup()` function checks:
1. `schema.sql` exists and is non-empty
2. `load.sql` exists and is non-empty
3. All Parquet files referenced in `load.sql` exist

---

## 6. Monitoring & Alerts

| Condition | Alert | Action |
|-----------|-------|--------|
| `AutoBackup` fails | Warning log | Check disk space, DuckDB file permissions |
| `VerifyBackup` fails | Error log + Sentry | Backup is corrupt; create new backup immediately |
| `BackupDir` disk > 90% | Warning log | Increase `BACKUP_KEEP` or add disk |
| Skipped backup (in-memory) | Info log | Expected for `:memory:` mode |

Logs are at `slog` level and visible in:
```bash
docker compose logs backend | grep "auto-backup\|export database\|verify"
```

---

## 7. Storage Sizing

| Backup type | Size ratio | Notes |
|-------------|------------|-------|
| File-copy `.duckdb` | 1x – 3x active DB | Includes internal fragmentation |
| EXPORT Parquet | 0.5x – 0.8x active DB | Columnar compression |

**Rule of thumb:** Allocate `BACKUP_KEEP × (2 × active_db_size)` for backup storage.
