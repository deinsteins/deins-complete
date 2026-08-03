ALTER TABLE quality_events ADD COLUMN client_version TEXT NOT NULL DEFAULT 'unknown';
CREATE INDEX quality_events_client_version_created_at_idx ON quality_events(client_version, created_at);
