#!/bin/bash
# PostgreSQL Backup Script for Aleph-v2
# Usage: ./scripts/backup-pg.sh

set -euo pipefail

# Configuration
BACKUP_DIR="/backups/postgres"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILENAME="aleph_pg_${TIMESTAMP}.sql"
BACKUP_PATH="${BACKUP_DIR}/${BACKUP_FILENAME}"

# Try to source .env for credentials
if [ -f "/app/.env" ]; then
    # shellcheck source=/dev/null
    source "/app/.env"
fi

# Environment variables with defaults
PGHOST="${PGHOST:-aleph-db}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-postgres}"
PGPASSWORD="${PGPASSWORD:-}"
PGDATABASE="${PGDATABASE:-aleph}"

# Export PGPASSWORD for pg_dump
export PGPASSWORD

echo "=== Aleph PostgreSQL Backup ==="
echo "Host: ${PGHOST}:${PGPORT}"
echo "Database: ${PGDATABASE}"
echo "User: ${PGUSER}"
echo "Backup directory: ${BACKUP_DIR}"
echo "Backup file: ${BACKUP_FILENAME}"
echo ""

# Ensure backup directory exists
mkdir -p "${BACKUP_DIR}"

# Verify PostgreSQL connection
echo "Verifying PostgreSQL connection..."
if ! pg_isready -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d "${PGDATABASE}" 2>/dev/null; then
    echo "ERROR: Cannot connect to PostgreSQL at ${PGHOST}:${PGPORT}"
    echo "Please check that PostgreSQL is running and accessible."
    exit 1
fi

echo "Connection successful!"
echo ""

# Perform backup
echo "Starting backup..."
if pg_dump -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d "${PGDATABASE}" \
    --verbose \
    --clean \
    --if-exists \
    --create \
    --no-owner \
    --no-privileges \
    --file "${BACKUP_PATH}"; then
    echo ""
    echo "Backup created: ${BACKUP_PATH}"
else
    echo "ERROR: pg_dump failed!"
    rm -f "${BACKUP_PATH}"
    exit 1
fi

# Compress backup
echo "Compressing backup..."
if gzip "${BACKUP_PATH}"; then
    COMPRESSED_PATH="${BACKUP_PATH}.gz"
    echo "Backup compressed: ${COMPRESSED_PATH}"
    
    # Show file info
    FILESIZE=$(du -h "${COMPRESSED_PATH}" | cut -f1)
    echo "Backup size: ${FILESIZE}"
else
    echo "WARNING: Compression failed, keeping uncompressed backup"
    COMPRESSED_PATH="${BACKUP_PATH}"
fi

# Clean up old backups (keep last 7 days)
echo "Cleaning up old backups..."
find "${BACKUP_DIR}" -name "aleph_pg_*.sql.gz" -type f -mtime +7 -delete 2>/dev/null || true

# Show current backups
echo ""
echo "Current backups in ${BACKUP_DIR}:"
ls -lh "${BACKUP_DIR}"

echo ""
echo "=== Backup completed successfully at $(date) ==="
