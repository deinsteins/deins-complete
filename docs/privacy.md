# Privacy

DeinsComplete sends bounded current-file context and, when enabled, small relevant workspace snippets to the managed backend so an AI provider can generate an inline completion. Repository context can be disabled per workspace.

## Optional quality insights

`deinscomplete.qualityInsights.enabled` is **off by default**. The extension offers a one-time consent prompt; dismissing or declining it prevents repeated notifications. When a tester explicitly enables it and the operator enables `QUALITY_EVENTS_ENABLED`, the extension sends only:

- whether a suggestion was shown or accepted;
- an explicit helpful/not-helpful vote and bounded reason category when the tester invokes a feedback command;
- a random completion correlation ID and safe backend request ID;
- bounded language, framework, completion-focus, fast/full, and cache/backend categories;
- the extension version category, so regressions can be compared between releases;
- completion latency in milliseconds.

Quality events never include source code, completion text, free-form feedback, file paths, repository snippets, package lists, provider/model identity, email addresses, installation tokens, account tokens, or API keys. There is no inferred “rejected” event because VS Code does not provide a reliable rejection signal. Explicit negative reasons are limited to categories such as incorrect API, irrelevant, slow, or too much code. Enabled events use the configured retention period (30 days by default), cleaned up during API startup.

The server may retain a deterministic percentage of completion IDs through `QUALITY_EVENTS_SAMPLE_PERCENT`. The shown and accepted events for one completion use the same sampling decision, preserving paired acceptance ratios. Operational logs and the admin dashboard expose aggregate counts and latency only; low-volume windows are labelled directional. Disable either the extension setting or `QUALITY_EVENTS_ENABLED` to stop collection.
