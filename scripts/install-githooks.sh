#!/bin/sh
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
git config core.hooksPath .githooks
echo "OK: Git hooks installed from .githooks/"
