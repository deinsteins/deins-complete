# Production readiness

- [ ] `APP_ENV=production`, installation authentication, rate limiting, daily abuse guard, and plan quota are enabled.
- [ ] `AUTH_TOKEN_SECRET` is high entropy and not committed.
- [ ] `DATABASE_ENABLED=true` has a reachable PostgreSQL `DATABASE_URL`; `/ready` fails closed if this required dependency is unavailable while `/health` remains process-only.
- [ ] PostgreSQL backup/snapshot was taken before migration; additive migrations ran once before API replicas were updated.
- [ ] `REGISTRATION_MODE` is intentionally set to `open`, `invite`, or `disabled`; private beta normally uses `invite`.
- [ ] Account access/refresh tokens, magic codes, installation tokens, emails, and SQL parameters are absent from logs.
- [ ] Account endpoints have separate rate limits; revoking an installation blocks its existing installation token within the documented validation/cache window.
- [ ] Provider and optional fallback target configuration validates at startup.
- [ ] TLS termination, protected logs, and health/readiness probes are configured.
- [ ] Internal diagnostics are access-controlled before exposure.
- [ ] Repository and extension bundle were checked for secrets.
- [ ] Backend URL, cancellation, fallback, linked-account quota, plan change, revocation, database recovery, and anonymous fallback were tested in an Extension Development Host.
