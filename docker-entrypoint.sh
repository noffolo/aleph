#!/bin/sh
# docker-entrypoint.sh — Reads Docker secrets into env vars before starting the app.
# This allows all sensitive values to be passed as Docker secrets (opaque files)
# while keeping the application code unchanged (it still reads env vars).
#
# Priority: Docker secret file > existing env var > default
set -e

# KEY_ENCRYPTION_KEY
if [ -f /run/secrets/key_encryption_key ]; then
    export KEY_ENCRYPTION_KEY="$(cat /run/secrets/key_encryption_key | tr -d '\n')"
fi

# JWT_SECRET
if [ -f /run/secrets/jwt_secret ]; then
    export JWT_SECRET="$(cat /run/secrets/jwt_secret | tr -d '\n')"
fi

# POSTGRES_DSN (contains password)
if [ -f /run/secrets/postgres_dsn ]; then
    export POSTGRES_DSN="$(cat /run/secrets/postgres_dsn | tr -d '\n')"
fi

# ALEPH_API_KEY_SECRET_BACKEND
if [ -f /run/secrets/aleph_api_key_secret_backend ]; then
    export ALEPH_API_KEY_SECRET_BACKEND="$(cat /run/secrets/aleph_api_key_secret_backend | tr -d '\n')"
fi

exec "$@"