# Private beta release checklist

- [ ] Version and `v<version>` tag match.
- [ ] Extension compile, lint, tests, and VSIX package pass.
- [ ] Go vet, tests, race tests, build, and Docker build pass.
- [ ] Production uses HTTPS, auth, rate limits, quotas, and strong secrets.
- [ ] Health, readiness, smoke test, fresh install, fallback, and rollback are verified.
- [ ] Secret scan and artifact inspection pass.
- [ ] Previous VSIX and image tag are retained for rollback.
- [ ] ARM64 and AMD64 manifest was published to GHCR.
- [ ] Production URL is `https://api.deinscomplete.web.id`; Caddy, health, readiness, registration, and completion smoke tests pass.
