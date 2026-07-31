package usage

import (
	"context"
	"time"

	"deinscomplete/api/internal/storage"
	"github.com/redis/go-redis/v9"
)

type MonthlyRedis struct{ client redis.UniversalClient }

func NewMonthlyRedis(client redis.UniversalClient) *MonthlyRedis {
	return &MonthlyRedis{client: client}
}

var monthlyQuota = redis.NewScript(`local n=tonumber(redis.call('GET',KEYS[1])) or 0; if n+tonumber(ARGV[1])>tonumber(ARGV[2]) then return {0,n} end; n=redis.call('INCRBY',KEYS[1],ARGV[1]); redis.call('PEXPIRE',KEYS[1],ARGV[3]); return {1,n}`)
var monthlyMerge = redis.NewScript(`if redis.call('SET',KEYS[3],'1','NX','PX',ARGV[1]) then local n=tonumber(redis.call('GET',KEYS[1])) or 0; if n>0 then redis.call('INCRBY',KEYS[2],n); redis.call('PEXPIRE',KEYS[2],ARGV[1]); end end; return 1`)

func (tracker *MonthlyRedis) CheckAndConsume(ctx context.Context, subject string, limit, amount int) Result {
	now := time.Now().UTC()
	out, err := monthlyQuota.Run(ctx, tracker.client, []string{monthlyKey(subject, now)}, amount, limit, monthlyTTL(now).Milliseconds()).Int64Slice()
	if err != nil || len(out) != 2 {
		if err == nil {
			err = redis.ErrClosed
		}
		return Result{Err: err}
	}
	return Result{Allowed: out[0] == 1, Count: int(out[1]), RetryAfter: time.Until(nextMonth(now))}
}
func (tracker *MonthlyRedis) Usage(ctx context.Context, subject string) (int, error) {
	value, err := tracker.client.Get(ctx, monthlyKey(subject, time.Now().UTC())).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return value, err
}
func (tracker *MonthlyRedis) MergeInstallationIntoUser(ctx context.Context, installationSubject, userSubject string) error {
	now := time.Now().UTC()
	ttl := monthlyTTL(now).Milliseconds()
	_, err := monthlyMerge.Run(ctx, tracker.client, []string{monthlyKey(installationSubject, now), monthlyKey(userSubject, now), monthlyMigrationKey(installationSubject, userSubject, now)}, ttl).Result()
	return err
}
func monthlyKey(subject string, now time.Time) string {
	return storage.InstallationKey(subject) + ":monthly_usage:" + now.UTC().Format("2006-01")
}
func monthlyMigrationKey(installation, user string, now time.Time) string {
	return storage.InstallationKey(installation+"\x00"+user) + ":monthly_merge:" + now.UTC().Format("2006-01")
}
func monthlyTTL(now time.Time) time.Duration {
	return nextMonth(now).Add(48 * time.Hour).Sub(now.UTC())
}
