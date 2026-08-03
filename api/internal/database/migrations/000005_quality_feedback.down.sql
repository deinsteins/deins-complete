DROP INDEX IF EXISTS quality_events_completion_feedback_idx;
ALTER TABLE quality_events DROP COLUMN IF EXISTS feedback_reason;
ALTER TABLE quality_events DROP CONSTRAINT quality_events_event_type_check;
ALTER TABLE quality_events ADD CONSTRAINT quality_events_event_type_check
    CHECK (event_type IN ('shown', 'accepted'));
