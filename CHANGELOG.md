# Changelog

## 0.1.15

- Fixed production releases to run additive database migrations before restarting the API.
- Fixed the Compose migration service to execute the migration binary instead of the API entrypoint.

## 0.1.14

- Added opt-in, privacy-safe shown and accepted completion quality insights.
- Added admin acceptance, latency, cache/backend, language, framework, and focus aggregates.
- Added bounded PostgreSQL retention; reporting remains disabled by default and never blocks completion.

## 0.1.13

- Added a native-themed Account Center with quota, entitlement, and installation management.
- Added interactive onboarding, manual completion triggering, conflict detection, and clearer completion status.
- Added persistent privacy-safe completion aggregates, adaptive local debounce, and copyable diagnostics/feedback reports.
- Improved React, Next.js, Vue, Svelte, Angular, and test-library context detection.

## 0.1.0-beta.1

- Inline completion through a managed backend.
- Installation authentication, admission controls, fallback routing, and FIM-aware targets.
- Privacy-safe diagnostics and local completion caching.
