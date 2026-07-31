# Accounts, entitlements, and PostgreSQL

Phase 19 keeps installation authentication for frictionless private-beta use and adds an optional DeinsComplete account layer. A new installation starts as anonymous Free. Signing in links the installation to an account; linked installations continue to use that account's plan even after the VS Code account session is signed out.

## Account authentication

Accounts use one-time magic codes rather than passwords. This avoids password storage, reset, and credential-stuffing concerns during private beta. `POST /v1/auth/magic/request` accepts an email and optional invite code, and always returns an acceptance response that does not disclose account eligibility. `POST /v1/auth/magic/verify` exchanges the short-lived, one-time code for a short-lived access token and a rotated, revocable refresh token.

The API stores only hashes for one-time codes and refresh tokens. Access tokens, refresh tokens, installation tokens, and source code must never be logged. The extension stores account credentials in VS Code SecretStorage, separately from its installation credential.

`REGISTRATION_MODE` is one of `open`, `invite`, or `disabled`; an unknown value is a startup error. Invite-only remains the recommended private-beta setting. Email delivery is supplied by the configured email sender abstraction; no OAuth provider is part of this phase.

## Linking and revocation

The extension links an installation by calling `POST /v1/installations/link` with both a user bearer token and `X-DeinsComplete-Installation-Token`. The server links only an unowned installation, or one already owned by that same user. It never silently transfers an installation to another account.

Installations are durable PostgreSQL records with `active` or `revoked` status. Installation-token validation checks durable status (with only bounded shared caching if configured), so a revoked signed token no longer authorizes completions. The API fails closed with `503 SERVICE_UNAVAILABLE` when the database is required but unavailable.

## Plans, usage, and features

The seeded plans are `free` and `pro`. Entitlements are data-backed and resolved by an entitlement service; handlers and routers do not select product behavior from a plan-name comparison. New accounts and anonymous installations resolve to Free.

The initial server-controlled defaults are:

| Plan | Monthly completions | Repository context | Streaming | Premium routing |
| --- | ---: | --- | --- | --- |
| Free | 2,000 | no (request is downgraded) | yes | no |
| Pro | 20,000 | yes | yes | yes |

Calendar-month usage is stored atomically in Redis. Linked installations share a `user:<uuid>` quota subject; anonymous installations use `installation:<uuid>`. When linking, the current month's anonymous usage is added once to the user's usage and a durable marker prevents a re-link from adding it again. Existing request-rate and daily-abuse guards remain separate from plan quota.

`GET /v1/account/entitlements` returns only plan-visible features and usage, never provider/model details. A plan change invalidates the safe Redis entitlement cache so an account need not reinstall or re-link.

## Local integration environment

Start disposable dependencies from the repository root:

```bash
docker compose -f api/docker-compose.integration.yml up -d --wait
export DATABASE_URL='postgres://deinscomplete:deinscomplete@localhost:5432/deinscomplete?sslmode=disable'
cd api
go run ./cmd/migrate up
go test ./...
docker compose -f docker-compose.integration.yml down -v
```

The integration Compose file deliberately exposes PostgreSQL and Redis only on loopback and does not persist their data. It is for development/testing, not production.

## Production migrations and recovery

Use managed PostgreSQL (Amazon RDS PostgreSQL is preferred) or an explicitly maintained temporary beta database; do not use Redis as account storage. Before a non-trivial migration, take and verify a PostgreSQL backup/snapshot. Deploy additive migrations first, then the API version. Run migrations once as a dedicated deployment step (`deinscomplete-api migrate up` or the release's migration command), never independently from every API replica.

`/health` is process-only. When `DATABASE_ENABLED=true`, `/ready` additionally requires PostgreSQL connectivity. A healthy connection pool reconnects on recovery; readiness returns and completion authorization resumes without an API restart.

## Creating private-beta invites

With `REGISTRATION_MODE=invite`, create one single-use invite per tester from the EC2 server. The invite is tied to the supplied email and its raw value is printed only once; send it privately and never commit it.

```bash
docker compose -f docker-compose.prod.yml run --rm api \
  /deinscomplete-admin create-invite tester@example.com 7
```

The final argument is an optional expiry in days (default `7`, maximum `30`). The tester must enter the same email and this code in **DeinsComplete: Sign In**.
