#!/usr/bin/env bash
set -Eeuo pipefail
root=/app/deinscomplete
image="${1:?usage: deploy-production.sh ghcr.io/owner/repo/api:tag}"
[[ "$image" =~ ^ghcr\.io/[A-Za-z0-9._/-]+:[A-Za-z0-9._-]+$ ]] || { echo "invalid image reference" >&2; exit 2; }
cd "$root"; test -f .env; test -f docker-compose.prod.yml; test -f Caddyfile
export DEINSCOMPLETE_API_IMAGE="$image"
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d --remove-orphans
for _ in {1..30}; do curl -fsS https://api.deinscomplete.web.id/ready >/dev/null && { printf '%s\n' "${image##*:}" > .deployed-version; echo "deployment ready"; exit 0; }; sleep 2; done
echo "readiness failed" >&2; exit 1
