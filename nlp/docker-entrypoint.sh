#!/bin/sh
set -e

if [ -f /run/secrets/aleph_api_key_secret ]; then
    export ALEPH_API_KEY_SECRET="$(cat /run/secrets/aleph_api_key_secret | tr -d '\n')"
fi

exec "$@"