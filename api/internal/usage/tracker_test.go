package usage

import (
	"context"
	"testing"
	"time"
)

func TestDailyQuotaResetsInUTC(t *testing.T) {
	tracker := New(2)
	now := time.Date(2026, 1, 1, 23, 59, 0, 0, time.UTC)
	tracker.now = func() time.Time { return now }
	if !tracker.CheckAndConsume(context.Background(), "a", 1).Allowed || !tracker.CheckAndConsume(context.Background(), "a", 1).Allowed || tracker.CheckAndConsume(context.Background(), "a", 1).Allowed {
		t.Fatal("unexpected quota admission")
	}
	if !tracker.CheckAndConsume(context.Background(), "b", 1).Allowed {
		t.Fatal("installations must be independent")
	}
	now = now.Add(2 * time.Minute)
	if !tracker.CheckAndConsume(context.Background(), "a", 1).Allowed {
		t.Fatal("expected UTC reset")
	}
}
