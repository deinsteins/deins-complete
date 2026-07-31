package usage

import (
	"context"
	"sync"
	"time"
)

// MonthlyTracker applies plan limits to a user or anonymous installation.
type MonthlyTracker interface {
	CheckAndConsume(context.Context, string, int, int) Result
	Usage(context.Context, string) (int, error)
	MergeInstallationIntoUser(context.Context, string, string) error
}

type MonthlyInMemory struct {
	mu         sync.Mutex
	entries    map[string]int
	migrations map[string]bool
	now        func() time.Time
}

func NewMonthly() *MonthlyInMemory {
	return &MonthlyInMemory{entries: map[string]int{}, migrations: map[string]bool{}, now: time.Now}
}
func (tracker *MonthlyInMemory) CheckAndConsume(_ context.Context, subject string, limit, amount int) Result {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	key := tracker.key(subject)
	used := tracker.entries[key]
	if used+amount > limit {
		return Result{Count: used, RetryAfter: time.Until(nextMonth(tracker.now()))}
	}
	used += amount
	tracker.entries[key] = used
	return Result{Allowed: true, Count: used}
}
func (tracker *MonthlyInMemory) Usage(_ context.Context, subject string) (int, error) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	return tracker.entries[tracker.key(subject)], nil
}
func (tracker *MonthlyInMemory) MergeInstallationIntoUser(_ context.Context, installationSubject, userSubject string) error {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	migration := tracker.key(installationSubject) + "->" + tracker.key(userSubject)
	if tracker.migrations[migration] {
		return nil
	}
	tracker.entries[tracker.key(userSubject)] += tracker.entries[tracker.key(installationSubject)]
	tracker.migrations[migration] = true
	return nil
}
func (tracker *MonthlyInMemory) key(subject string) string {
	return tracker.now().UTC().Format("2006-01") + ":" + subject
}
func nextMonth(now time.Time) time.Time {
	return time.Date(now.UTC().Year(), now.UTC().Month()+1, 1, 0, 0, 0, 0, time.UTC)
}
