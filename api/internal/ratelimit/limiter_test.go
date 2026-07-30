package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestPerInstallationBucketRefills(t *testing.T) {
	l := New(60, 2)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }
	if !l.Allow(context.Background(), "a").Allowed || !l.Allow(context.Background(), "a").Allowed || l.Allow(context.Background(), "a").Allowed {
		t.Fatal("unexpected bucket admission")
	}
	if !l.Allow(context.Background(), "b").Allowed {
		t.Fatal("installations must be independent")
	}
	now = now.Add(time.Second)
	if !l.Allow(context.Background(), "a").Allowed {
		t.Fatal("expected refill")
	}
}
