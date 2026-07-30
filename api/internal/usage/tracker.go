package usage

import (
	"context"
	"sync"
	"time"
)

type Result struct {
	Allowed    bool
	Count      int
	RetryAfter time.Duration
}
type Tracker interface {
	CheckAndConsume(context.Context, string, int) Result
}
type entry struct {
	day   string
	count int
	last  time.Time
}
type InMemory struct {
	mu      sync.Mutex
	entries map[string]entry
	daily   int
	now     func() time.Time
}

func New(daily int) *InMemory {
	return &InMemory{entries: map[string]entry{}, daily: daily, now: time.Now}
}
func (t *InMemory) CheckAndConsume(_ context.Context, id string, amount int) Result {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now().UTC()
	for key, stale := range t.entries {
		if now.Sub(stale.last) > 24*time.Hour {
			delete(t.entries, key)
		}
	}
	day := now.Format("2006-01-02")
	e := t.entries[id]
	if e.day != day {
		e = entry{day: day}
	}
	e.last = now
	if e.count+amount > t.daily {
		return Result{Count: e.count, RetryAfter: time.Until(now.Truncate(24 * time.Hour).Add(24 * time.Hour))}
	}
	e.count += amount
	t.entries[id] = e
	return Result{Allowed: true, Count: e.count}
}
