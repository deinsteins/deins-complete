#!/usr/bin/env bash
set -euo pipefail
base_url="${DEINSCOMPLETE_API_URL:-http://127.0.0.1:3001}"
curl --fail --silent --show-error "$base_url/health" | grep -q '"status":"ok"'
curl --fail --silent --show-error "$base_url/ready" | grep -q '"status":"ready"'
echo "DeinsComplete smoke test passed: $base_url"
