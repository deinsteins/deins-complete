# Privacy

DeinsComplete sends bounded current-file context and, when enabled, small relevant workspace snippets to the managed backend so an AI provider can generate an inline completion. Repository context can be disabled per workspace.

## Optional quality insights

`deinscomplete.qualityInsights.enabled` is **off by default**. The extension offers a one-time consent prompt; dismissing or declining it prevents repeated notifications. When a tester explicitly enables it and the operator enables `QUALITY_EVENTS_ENABLED`, the extension sends only:

- whether a suggestion was shown or accepted;
- a random completion correlation ID and safe backend request ID;
- bounded language, framework, completion-focus, fast/full, and cache/backend categories;
- completion latency in milliseconds.

Quality events never include source code, completion text, file paths, repository snippets, package lists, provider/model identity, email addresses, installation tokens, account tokens, or API keys. There is no inferred “rejected” event because VS Code does not provide a reliable rejection signal. Enabled events use the configured retention period (30 days by default), cleaned up during API startup.

Operational logs and the admin dashboard expose aggregate counts and latency only. Disable either the extension setting or `QUALITY_EVENTS_ENABLED` to stop collection.
