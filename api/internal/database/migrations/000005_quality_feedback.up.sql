ALTER TABLE quality_events DROP CONSTRAINT quality_events_event_type_check;
ALTER TABLE quality_events ADD CONSTRAINT quality_events_event_type_check
    CHECK (event_type IN ('shown', 'accepted', 'helpful', 'not-helpful'));
ALTER TABLE quality_events ADD COLUMN feedback_reason TEXT NOT NULL DEFAULT 'none'
    CHECK (feedback_reason IN ('none', 'general', 'incorrect-api', 'irrelevant', 'too-slow', 'too-much-code', 'other'));
CREATE UNIQUE INDEX quality_events_completion_feedback_idx ON quality_events(completion_id)
    WHERE event_type IN ('helpful', 'not-helpful');
