#!/usr/bin/env bash
set -euo pipefail
base_url="${DEINSCOMPLETE_API_URL:-http://127.0.0.1:3001}"
curl --fail --silent --show-error "$base_url/health" | grep -q '"status":"ok"'
curl --fail --silent --show-error "$base_url/ready" | grep -q '"status":"ready"'
installation="smoke-$(date +%s)-$$"
registration="$(curl --fail --silent --show-error -H 'content-type: application/json' -d "{\"installationId\":\"$installation\"}" "$base_url/v1/installations/register")"
token="$(printf '%s' "$registration" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"
test -n "$token"
if [[ "${SMOKE_COMPLETION:-0}" == 1 ]]; then
  curl --fail --silent --show-error -H "authorization: Bearer $token" -H 'content-type: application/json' -d '{"context":{"prefix":"const sum = ","suffix":";","language":"javascript","filePath":"smoke.js","cursorOffset":12}}' "$base_url/v1/completions" | grep -q '"completion"'
fi
unset token registration
echo "DeinsComplete smoke test passed: $base_url"
