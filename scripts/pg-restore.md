# PostgreSQL Backup and Restore Documentation

## Prerequisites

Before restoring a PostgreSQL backup, ensure:

- PostgreSQL 16 or compatible version is running
- The `aleph` database exists or can be created
- Backup files are available in the `backups/postgres/` directory
- You have sufficient disk space (backups can be large)

## Restore from Latest Backup

### Option 1: Restore using Docker Compose (Recommended)

```bash
# Find the latest backup
LATEST_BACKUP=$(ls -t backups/postgres/aleph_pg_*.sql.gz | head -n1)

# Copy and decompress backup into container
docker compose cp "$LATEST_BACKUP" aleph-db:/tmp/backup.sql.gz
docker compose exec aleph-db gunzip /tmp/backup.sql.gz

# Drop and recreate database
docker compose exec aleph-db psql -U postgres -c "DROP DATABASE IF EXISTS aleph;"
docker compose exec aleph-db psql -U postgres -c "CREATE DATABASE aleph;"

# Restore from backup
docker compose exec aleph-db psql -U postgres -d aleph -f /tmp/backup.sql
```

### Option 2: Restore using psql directly

```bash
# Find the latest backup
LATEST_BACKUP=$(ls -t backups/postgres/aleph_pg_*.sql.gz | head -n1)

# Set environment variables
export PGHOST=localhost
export PGPORT=5432
export PGUSER=postgres
export PGPASSWORD=your_password

# Drop existing database and create new
psql -c "DROP DATABASE IF EXISTS aleph;"
psql -c "CREATE DATABASE aleph;"

# Restore from compressed backup
gunzip -c "$LATEST_BACKUP" | psql -d aleph

# Or restore from uncompressed backup
LATEST_BACKUP_UNCOMPRESSED=$(ls -t backups/postgres/aleph_pg_*.sql | head -n1)
psql -d aleph < "$LATEST_BACKUP_UNCOMPRESSED"
```

### Option 3: Point-in-Time Restore (PITR)

```bash
# List available backups
ls -lh backups/postgres/aleph_pg_*.sql.gz

# Choose specific backup by date
BACKUP_DATE="20240501_120000"  # YYYYMMDD_HHMMSS format
BACKUP_FILE="backups/postgres/aleph_pg_${BACKUP_DATE}.sql.gz"

# Restore from specific backup
gunzip -c "$BACKUP_FILE" | docker compose exec -T aleph-db psql -U postgres -d aleph
```

## Running Backups

### Manual Backup

```bash
# Run backup manually
docker compose --profile backup up aleph-pg-backup

# Or run once and exit
docker compose --profile backup run --rm aleph-pg-backup /backup-pg.sh
```

### Automated Backup Schedule

Add to crontab for automated daily backups:

```bash
# Edit crontab
crontab -e

# Add entry for daily backup at 2 AM
0 2 * * * cd /path/to/aleph-v2 && docker compose --profile backup up aleph-pg-backup --exit-code-from aleph-pg-backup

# Or using Docker Compose run (recommended for cron)
0 2 * * * cd /path/to/aleph-v2 && docker compose --profile backup run --rm aleph-pg-backup /backup-pg.sh
```

## Backup Retention

By default, backups older than 7 days are automatically cleaned up. To modify retention:

1. Edit `scripts/backup-pg.sh`
2. Change the `mtime` parameter in the cleanup section (line ~86)
3. Restart the backup service

## Troubleshooting

### Connection Issues

```bash
# Verify PostgreSQL is healthy
docker compose ps aleph-db

# Check PostgreSQL logs
docker compose logs aleph-db

# Test connection manually
docker compose exec aleph-db pg_isready -U postgres -d aleph
```

### Restore Issues

- **"Database is being accessed by other users"**: Stop all services accessing the database first
- **"Permission denied"**: Ensure correct file permissions on backup files
- **"Archive is not a gzip file"**: File may already be decompressed

### Backup File Locations

- Backups are stored in: `./backups/postgres/`
- Naming format: `aleph_pg_YYYYMMDD_HHMMSS.sql.gz`
- Each backup is automatically compressed with gzip
