# Changelog

## 0.1.17

- Added intent-aware completion policies for imports, function bodies, conditions, types, JSX props, members, arguments, object fields, and Tailwind classes.
- Added intent-specific prompt guidance, token/line limits, and conservative wrong-language output rejection.
- Connected the existing Helpful, Not Helpful, and Bad Suggestion commands to the opt-in privacy-safe quality pipeline.
- Added bounded negative-feedback reason categories without source code or free-form text.
- Added helpful/not-helpful totals and reason aggregates to the protected admin dashboard.

## 0.1.16

- Added deterministic server-side sampling for privacy-safe quality events.
- Added daily UTC quality trends and extension-version comparisons to the internal admin dashboard.
- Added a low-sample warning so beta quality ratios are not overinterpreted.

## 0.1.15

- Fixed production releases to run additive database migrations before restarting the API.
- Fixed the Compose migration service to execute the migration binary instead of the API entrypoint.
- Added a one-time, non-blocking consent prompt for optional privacy-safe quality insights.

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
