package ratelimit

import (
	"context"
	"deinscomplete/api/internal/storage"
	"github.com/redis/go-redis/v9"
	"time"
)

type Redis struct {
	client           redis.UniversalClient
	perMinute, burst int
}

func NewRedis(client redis.UniversalClient, perMinute, burst int) *Redis {
	return &Redis{client: client, perMinute: perMinute, burst: burst}
}

var tokenBucket = redis.NewScript(`local d=redis.call('HMGET',KEYS[1],'t','l'); local t=tonumber(d[1]) or tonumber(ARGV[1]); local l=tonumber(d[2]) or tonumber(ARGV[2]); t=math.min(tonumber(ARGV[1]),t+(tonumber(ARGV[2])-l)*tonumber(ARGV[3])); local ok=t>=1; if ok then t=t-1 end; redis.call('HMSET',KEYS[1],'t',t,'l',ARGV[2]); redis.call('PEXPIRE',KEYS[1],ARGV[4]); return {ok and 1 or 0,math.floor(t),math.ceil(math.max(0,1-t)/tonumber(ARGV[3]))}`)

func (r *Redis) Allow(ctx context.Context, id string) Result {
	now := time.Now()
	refill := float64(r.perMinute) / float64(time.Minute)
	out, err := tokenBucket.Run(ctx, r.client, []string{storage.InstallationKey(id) + ":ratelimit"}, r.burst, now.UnixNano(), refill, (5 * time.Minute).Milliseconds()).Int64Slice()
	if err != nil || len(out) != 3 {
		if err == nil {
			err = redis.ErrClosed
		}
		return Result{Err: err}
	}
	return Result{Allowed: out[0] == 1, Remaining: int(out[1]), RetryAfter: time.Duration(out[2])}
}
