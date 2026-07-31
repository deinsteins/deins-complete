package storage

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"time"

	"deinscomplete/api/internal/config"
	"github.com/redis/go-redis/v9"
)

// Client owns one reusable Redis connection pool. It stores only hashed IDs.
type Client struct{ *redis.Client }

func (client *Client) Ready(ctx context.Context) error { return client.Ping(ctx).Err() }

func NewRedis(cfg config.RedisConfig) (*Client, error) {
	options := &redis.Options{Addr: cfg.Addr, Username: cfg.Username, Password: cfg.Password, DB: cfg.DB, DialTimeout: cfg.ConnectTimeout, ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout}
	if cfg.TLS {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &Client{client}, nil
}

func InstallationKey(id string) string {
	sum := sha256.Sum256([]byte(id))
	return "deinscomplete:v1:" + hex.EncodeToString(sum[:])
}
func ExpireAtNextUTCDay(now time.Time) time.Duration {
	return now.UTC().Truncate(24 * time.Hour).Add(48 * time.Hour).Sub(now.UTC())
}
