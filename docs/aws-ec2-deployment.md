# AWS EC2 private-beta deployment

Runtime: Ubuntu ARM64 on EC2 t4g.small, Docker Compose, Caddy, Redis, and the Go API on the internal Docker network. Public traffic is only HTTPS at `https://api.deinscomplete.web.id`; Redis is internal-only at `redis:6379` and must never have a host `ports:` mapping.

On the server, retain the existing `/app/deinscomplete/.env`; CI never copies or overwrites it. Add the non-secret image repository to the deployment shell when rolling back: `export DEINSCOMPLETE_IMAGE_REPOSITORY=ghcr.io/<owner>/<repository>/api`.

Required GitHub Actions secrets: `EC2_HOST`, `EC2_USER`, `EC2_SSH_PRIVATE_KEY_BASE64`, and trusted `EC2_KNOWN_HOSTS`. Create the base64 value locally with `base64 -w 0 ~/.ssh/<deployment-private-key>` (macOS: `base64 < ~/.ssh/<deployment-private-key> | tr -d '\n'`). Provider credentials and the auth secret remain only in the EC2 `.env`. For private GHCR images, authenticate Docker on EC2 once with a token that has package read access.

Deploy an immutable image:

```bash
cd /app/deinscomplete
./deploy-production.sh ghcr.io/<owner>/<repository>/api:v0.1.0-beta.1
```

Rollback:

```bash
export DEINSCOMPLETE_IMAGE_REPOSITORY=ghcr.io/<owner>/<repository>/api
./rollback-production.sh 0.1.0-beta.1
```

The scripts preserve `.env`, pull images, reconcile Compose without deleting volumes, and require `/ready` to succeed. Caddy obtains and persists TLS certificates in Docker volumes. Configure Docker host log rotation separately (for example `max-size=10m`, `max-file=3`). In-memory quotas and rate-limit state reset on API restart.
