# AWS EC2 private-beta deployment

Runtime: Ubuntu ARM64 on EC2 t4g.small, Docker Compose, Caddy, Redis, the Go API, and managed PostgreSQL. Public traffic is only HTTPS at `https://api.deinscomplete.web.id`; Redis is internal-only at `redis:6379` and must never have a host `ports:` mapping. PostgreSQL is the persistent account/entitlement store; Amazon RDS PostgreSQL is preferred. Do not expose a self-hosted beta database publicly.

On the server, retain the existing `/app/deinscomplete/.env`; CI never copies or overwrites it. Add the non-secret image repository to the deployment shell when rolling back: `export DEINSCOMPLETE_IMAGE_REPOSITORY=ghcr.io/<owner>/<repository>/api`.

Required GitHub Actions secrets: `EC2_HOST`, `EC2_USER`, `EC2_SSH_PRIVATE_KEY_BASE64`, and trusted `EC2_KNOWN_HOSTS`. Create the base64 value locally with `base64 -w 0 ~/.ssh/<deployment-private-key>` (macOS: `base64 < ~/.ssh/<deployment-private-key> | tr -d '\n'`). Provider credentials, auth secrets, email sender credentials, and `DATABASE_URL` remain only in the EC2 `.env`. For private GHCR images, authenticate Docker on EC2 once with a token that has package read access.

## PostgreSQL accounts database (single-EC2 beta)

`docker-compose.prod.yml` includes an internal `postgres:17-alpine` service and persistent `postgres_data` volume. PostgreSQL has no public host port; only the API reaches it through the `deinscomplete` Docker network. Before enabling accounts, add these values to `/app/deinscomplete/.env` (never commit it):

```env
POSTGRES_DB=deinscomplete
POSTGRES_USER=deinscomplete
POSTGRES_PASSWORD=<openssl rand -hex 32>
DATABASE_ENABLED=true
DATABASE_URL=postgres://deinscomplete:<same-password>@postgres:5432/deinscomplete?sslmode=disable
```

Use a hexadecimal password so it needs no URL escaping. After pulling the account-enabled API image, start the database, run the additive migration once, then start the API:

```bash
docker compose -f docker-compose.prod.yml up -d postgres redis
docker compose -f docker-compose.prod.yml --profile maintenance run --rm migrate
docker compose -f docker-compose.prod.yml up -d api caddy
```

`postgres_data` is the account database. Back it up before schema changes; do not use `docker compose down -v` in production.

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

Before each production database migration, take a verified RDS snapshot/backup. Run the release migration command exactly once, then deploy the API; keep migrations additive so the preceding API can be rolled back. The scripts preserve `.env`, pull images, reconcile Compose without deleting volumes, and require `/ready` to succeed. Caddy obtains and persists TLS certificates in Docker volumes. Configure Docker host log rotation separately (for example `max-size=10m`, `max-file=3`). Redis-backed counters survive API restart when Redis persistence is configured; PostgreSQL remains the source of truth for accounts, installations, and plans.
