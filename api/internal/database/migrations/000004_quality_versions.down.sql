DROP INDEX IF EXISTS quality_events_client_version_created_at_idx;
ALTER TABLE quality_events DROP COLUMN IF EXISTS client_version;
