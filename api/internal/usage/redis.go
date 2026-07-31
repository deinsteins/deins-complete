package usage

import (
	"context"
	"deinscomplete/api/internal/storage"
	"github.com/redis/go-redis/v9"
	"time"
)

type Redis struct {
	client redis.UniversalClient
	daily  int
}

func NewRedis(client redis.UniversalClient, daily int) *Redis {
	return &Redis{client: client, daily: daily}
}

var quota = redis.NewScript(`local n=redis.call('GET',KEYS[1]); n=tonumber(n) or 0; if n+tonumber(ARGV[1])>tonumber(ARGV[2]) then return {0,n} end; n=redis.call('INCRBY',KEYS[1],ARGV[1]); redis.call('PEXPIRE',KEYS[1],ARGV[3]); return {1,n}`)

func (t *Redis) CheckAndConsume(ctx context.Context, id string, amount int) Result {
	now := time.Now().UTC()
	key := storage.InstallationKey(id) + ":usage:" + now.Format("2006-01-02")
	out, err := quota.Run(ctx, t.client, []string{key}, amount, t.daily, storage.ExpireAtNextUTCDay(now).Milliseconds()).Int64Slice()
	if err != nil || len(out) != 2 {
		if err == nil {
			err = redis.ErrClosed
		}
		return Result{Err: err}
	}
	return Result{Allowed: out[0] == 1, Count: int(out[1]), RetryAfter: time.Until(now.Truncate(24 * time.Hour).Add(24 * time.Hour))}
}
