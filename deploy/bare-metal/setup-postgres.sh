#!/bin/bash
set -euo pipefail

# Aleph Data OS — PostgreSQL Setup Script for Ubuntu 22.04/24.04
# Run as root or a user with sudo privileges

DB_NAME="${ALEPH_DB_NAME:-aleph}"
DB_USER="${ALEPH_DB_USER:-aleph}"
DB_PASSWORD="${ALEPH_DB_PASSWORD:-}"

if [[ -z "$DB_PASSWORD" ]]; then
  echo "Error: ALEPH_DB_PASSWORD must be set."
  echo "Usage: ALEPH_DB_PASSWORD=secret PASSWORD ./setup-postgres.sh"
  exit 1
fi

echo "==> Installing PostgreSQL 16..."
apt-get update
apt-get install -y postgresql-16 postgresql-contrib-16 postgresql-client-16

systemctl enable postgresql
systemctl start postgresql

echo "==> Creating database and user..."
sudo -u postgres psql -c "CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';" 2>/dev/null || echo "User already exists."
sudo -u postgres psql -c "CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};" 2>/dev/null || echo "Database already exists."
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};"

echo "==> Applying schema extensions..."
sudo -u postgres psql -d "${DB_NAME}" -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
sudo -u postgres psql -d "${DB_NAME}" -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;"

echo "==> Configuring pg_hba.conf for local + password auth..."
PG_HBA="/etc/postgresql/16/main/pg_hba.conf"
cp "${PG_HBA}" "${PG_HBA}.backup.$(date +%Y%m%d%H%M%S)"

# Ensure local connections use md5/scram-sha-256 instead of peer
sed -i 's/^local\s\+all\s\+all\s\+peer/local   all             all                                     scram-sha-256/' "${PG_HBA}"
sed -i 's/^host\s\+all\s\+all\s\+127.0.0.1\/32\s\+ident/host    all             all             127.0.0.1\/32            scram-sha-256/' "${PG_HBA}"
sed -i 's/^host\s\+all\s\+all\s\+::1\/128\s\+ident/host    all             all             ::1\/128                 scram-sha-256/' "${PG_HBA}"

echo "==> Restarting PostgreSQL..."
systemctl restart postgresql

echo "==> Done."
echo "Database: ${DB_NAME}"
echo "User:     ${DB_USER}"
echo "Connection string: postgresql://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable"
