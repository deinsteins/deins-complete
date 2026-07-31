package usage

import (
	"context"
	"testing"
	"time"
)

func TestMonthlyQuotaAndIdempotentMerge(t *testing.T) {
	tracker := NewMonthly()
	tracker.now = func() time.Time { return time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC) }
	if !tracker.CheckAndConsume(context.Background(), "installation:a", 10, 6).Allowed {
		t.Fatal("expected allowance")
	}
	if err := tracker.MergeInstallationIntoUser(context.Background(), "installation:a", "user:u"); err != nil {
		t.Fatal(err)
	}
	if err := tracker.MergeInstallationIntoUser(context.Background(), "installation:a", "user:u"); err != nil {
		t.Fatal(err)
	}
	if used, _ := tracker.Usage(context.Background(), "user:u"); used != 6 {
		t.Fatalf("used=%d", used)
	}
	if tracker.CheckAndConsume(context.Background(), "user:u", 10, 5).Allowed {
		t.Fatal("expected limit rejection")
	}
}
