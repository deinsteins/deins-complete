# Completion streaming

When `STREAMING_ENABLED=true`, authenticated clients may call `POST /v1/completions/stream` with the same JSON body as `/v1/completions`.

The response is SSE. `chunk` events contain raw provider text for low-latency consumers. A `done` event contains the final, sanitized `text` and `requestId`; clients should use that final text for cached or displayed completion output. An `error` event is only possible after the HTTP response starts. Authentication, rate limiting, and quota checks run before SSE headers are sent.

Cancelling the HTTP request cancels the backend context and the upstream provider request. Router fallback is permitted only before a provider emits a chunk; output from two providers is never combined. If a target does not support native streaming, its normal result is delivered as a single chunk followed by `done`.
