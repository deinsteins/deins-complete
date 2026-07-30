package ratelimit

import (
	"context"
	"sync"
	"time"
)

type Result struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}
type Limiter interface {
	Allow(context.Context, string) Result
}
type bucket struct {
	tokens float64
	last   time.Time
}
type InMemory struct {
	mu               sync.Mutex
	buckets          map[string]bucket
	perMinute, burst int
	now              func() time.Time
}

func New(perMinute, burst int) *InMemory {
	return &InMemory{buckets: map[string]bucket{}, perMinute: perMinute, burst: burst, now: time.Now}
}
func (l *InMemory) Allow(_ context.Context, id string) Result {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for key, stale := range l.buckets {
		if now.Sub(stale.last) > 24*time.Hour {
			delete(l.buckets, key)
		}
	}
	b := l.buckets[id]
	if b.last.IsZero() {
		b.tokens = float64(l.burst)
	} else {
		b.tokens = min(float64(l.burst), b.tokens+now.Sub(b.last).Minutes()*float64(l.perMinute))
	}
	b.last = now
	if b.tokens < 1 {
		l.buckets[id] = b
		return Result{RetryAfter: time.Duration((1 - b.tokens) / float64(l.perMinute) * float64(time.Minute))}
	}
	b.tokens--
	l.buckets[id] = b
	return Result{Allowed: true, Remaining: int(b.tokens)}
}
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
