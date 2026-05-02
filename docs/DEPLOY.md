# Aleph-v2 Deployment & Rollback Guide

## Pre-Deploy Checklist

1. **Backup PostgreSQL**
   ```bash
   docker compose exec aleph-db pg_dump -U postgres aleph > backups/aleph_pre_deploy_$(date +%Y%m%d_%H%M%S).sql
   ```
2. **Backup DuckDB**
   ```bash
   docker compose cp aleph-backend:/app/data/aleph.duckdb backups/aleph_pre_deploy_$(date +%Y%m%d_%H%M%S).duckdb
   ```
3. **Verify migration status**
   ```bash
   docker compose exec aleph-db psql -U postgres -d aleph -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 5;"
   ```
4. **Dry-run migrations** (in staging first)
5. **Notify team** with expected downtime window

## PostgreSQL Migration Rollback

Each `.up.sql` migration has a corresponding `.down.sql` file in `migrations/postgres/`.

### List applied migrations
```bash
docker compose exec aleph-db psql -U postgres -d aleph -c "SELECT version, applied_at FROM schema_migrations ORDER BY version;"
```

### Roll back a specific migration (example: 000009)
```bash
docker compose exec aleph-db psql -U postgres -d aleph -f migrations/postgres/000009_add_constraints.down.sql
# Then remove from tracking table
docker compose exec aleph-db psql -U postgres -d aleph -c "DELETE FROM schema_migrations WHERE version = 9;"
```

### Full rollback (reverse order)
```bash
for f in $(ls migrations/postgres/*.down.sql | sort -r); do
  docker compose exec aleph-db psql -U postgres -d aleph -f "$f"
done
docker compose exec aleph-db psql -U postgres -d aleph -c "TRUNCATE schema_migrations;"
```

## DuckDB Rollback

DuckDB is a single-file database — rollback means restoring a backup copy.

### Restore from backup
```bash
docker compose stop aleph-backend
docker compose cp backups/aleph_pre_deploy_20260502_120000.duckdb aleph-backend:/app/data/aleph.duckdb
docker compose start aleph-backend
```

### Verify integrity
```bash
docker compose exec aleph-backend ./aleph --health-check-duckdb
```

## Docker Image Rollback

```bash
docker compose pull aleph-backend:previous-tag
docker compose up -d aleph-backend
```

## Rollback Decision Matrix

| Problem | Action | Downtime |
|---------|--------|----------|
| Failed migration (no data loss) | Run `.down.sql` + re-apply fix | ~1 min |
| Data corruption in PostgreSQL | Restore `pg_dump` backup | ~5-15 min |
| DuckDB corruption | Restore file backup | ~1 min |
| Bad deploy (logic bug) | Tag previous image + redeploy | ~2 min |
| Full recovery needed | `docker compose down && restore volumes from snapshot && docker compose up -d` | ~15-30 min |

## Emergency Contacts

Ensure on-call rotation is set before deploying to production.
