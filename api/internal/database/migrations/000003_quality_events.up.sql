CREATE TABLE quality_events (
    id UUID PRIMARY KEY,
    completion_id UUID NOT NULL,
    installation_id UUID NOT NULL REFERENCES installations(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN ('shown', 'accepted')),
    server_request_id TEXT,
    language TEXT NOT NULL,
    framework TEXT NOT NULL,
    focus TEXT NOT NULL,
    mode TEXT NOT NULL CHECK (mode IN ('fast', 'full')),
    source TEXT NOT NULL CHECK (source IN ('backend', 'cache')),
    latency_ms INTEGER NOT NULL CHECK (latency_ms >= 0 AND latency_ms <= 30000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (completion_id, event_type)
);

CREATE INDEX quality_events_created_at_idx ON quality_events(created_at);
CREATE INDEX quality_events_language_created_at_idx ON quality_events(language, created_at);
CREATE INDEX quality_events_framework_created_at_idx ON quality_events(framework, created_at);
