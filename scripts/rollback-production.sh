#!/usr/bin/env bash
set -Eeuo pipefail
tag="${1:?usage: rollback-production.sh version-tag}"
[[ "$tag" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?$ ]] || { echo "invalid version tag" >&2; exit 2; }
: "${DEINSCOMPLETE_IMAGE_REPOSITORY:?set DEINSCOMPLETE_IMAGE_REPOSITORY (for example ghcr.io/owner/repo/api)}"
exec "$(dirname "$0")/deploy-production.sh" "${DEINSCOMPLETE_IMAGE_REPOSITORY}:${tag}"
