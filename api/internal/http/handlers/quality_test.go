package handlers

import (
	"testing"

	"github.com/google/uuid"
)

func TestQualityEventValidationAcceptsOnlyBoundedMetadata(t *testing.T) {
	eventID, completionID := uuid.NewString(), uuid.NewString()
	if !validQualityEvent(eventID, completionID, "shown", "request-1", "typescriptreact", "react", "component-props", "full", "backend", 125) {
		t.Fatal("expected valid quality event")
	}
	invalid := []struct {
		name  string
		valid bool
	}{
		{"bad UUID", validQualityEvent("bad", completionID, "shown", "", "ts", "none", "general", "fast", "backend", 1)},
		{"invented rejection", validQualityEvent(eventID, completionID, "rejected", "", "ts", "none", "general", "fast", "backend", 1)},
		{"path-like language", validQualityEvent(eventID, completionID, "shown", "", "/home/private.ts", "none", "general", "fast", "backend", 1)},
		{"newline request ID", validQualityEvent(eventID, completionID, "shown", "secret\nvalue", "ts", "none", "general", "fast", "backend", 1)},
		{"oversized latency", validQualityEvent(eventID, completionID, "shown", "", "ts", "none", "general", "fast", "backend", 30001)},
	}
	for _, item := range invalid {
		if item.valid {
			t.Fatalf("%s should be invalid", item.name)
		}
	}
}
