package handlers

import (
	"testing"

	"github.com/google/uuid"
)

func TestQualityEventValidationAcceptsOnlyBoundedMetadata(t *testing.T) {
	eventID, completionID := uuid.NewString(), uuid.NewString()
	if !validQualityEvent(eventID, completionID, "shown", "request-1", "typescriptreact", "react", "component-props", "full", "backend", "0.1.17", "none", 125) {
		t.Fatal("expected valid quality event")
	}
	if !validQualityEvent(eventID, completionID, "not-helpful", "request-1", "typescriptreact", "react", "component-props", "full", "backend", "0.1.17", "incorrect-api", 125) {
		t.Fatal("expected valid feedback event")
	}
	invalid := []struct {
		name  string
		valid bool
	}{
		{"bad UUID", validQualityEvent("bad", completionID, "shown", "", "ts", "none", "general", "fast", "backend", "0.1.17", "none", 1)},
		{"invented rejection", validQualityEvent(eventID, completionID, "rejected", "", "ts", "none", "general", "fast", "backend", "0.1.17", "none", 1)},
		{"path-like language", validQualityEvent(eventID, completionID, "shown", "", "/home/private.ts", "none", "general", "fast", "backend", "0.1.17", "none", 1)},
		{"newline request ID", validQualityEvent(eventID, completionID, "shown", "secret\nvalue", "ts", "none", "general", "fast", "backend", "0.1.17", "none", 1)},
		{"invalid version", validQualityEvent(eventID, completionID, "shown", "", "ts", "none", "general", "fast", "backend", "../../secret", "none", 1)},
		{"reason on shown", validQualityEvent(eventID, completionID, "shown", "", "ts", "none", "general", "fast", "backend", "0.1.17", "irrelevant", 1)},
		{"unbounded reason", validQualityEvent(eventID, completionID, "not-helpful", "", "ts", "none", "general", "fast", "backend", "0.1.17", "contains-secret", 1)},
		{"oversized latency", validQualityEvent(eventID, completionID, "shown", "", "ts", "none", "general", "fast", "backend", "0.1.17", "none", 30001)},
	}
	for _, item := range invalid {
		if item.valid {
			t.Fatalf("%s should be invalid", item.name)
		}
	}
}

func TestQualitySamplingIsStablePerCompletion(t *testing.T) {
	id := uuid.NewString()
	if !qualitySampled(id, 100) || qualitySampled(id, 0) {
		t.Fatal("sampling boundaries are invalid")
	}
	first := qualitySampled(id, 25)
	for attempt := 0; attempt < 10; attempt++ {
		if qualitySampled(id, 25) != first {
			t.Fatal("completion sampling must be deterministic")
		}
	}
}
