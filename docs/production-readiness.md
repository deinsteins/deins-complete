# Production readiness

- [ ] `APP_ENV=production`, authentication, rate limiting, and quota are enabled.
- [ ] `AUTH_TOKEN_SECRET` is high entropy and not committed.
- [ ] Provider and optional fallback target configuration validates at startup.
- [ ] TLS termination, protected logs, and health/readiness probes are configured.
- [ ] Internal diagnostics are access-controlled before exposure.
- [ ] Repository and extension bundle were checked for secrets.
- [ ] Backend URL, cancellation, fallback, quota, and recovery were tested in an Extension Development Host.
