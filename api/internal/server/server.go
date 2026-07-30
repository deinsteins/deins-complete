package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"deinscomplete/api/internal/auth"
	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/config"
	"deinscomplete/api/internal/ratelimit"
	"deinscomplete/api/internal/storage"
	"deinscomplete/api/internal/usage"
)

type Server struct {
	httpServer *http.Server
	redis      *storage.Client
}

func New(configuration config.Config, logger *slog.Logger, service *completion.Service) (*Server, error) {
	var redisClient *storage.Client
	var err error
	if configuration.Redis.Enabled {
		redisClient, err = storage.NewRedis(configuration.Redis)
		if err != nil {
			return nil, fmt.Errorf("redis unavailable: %w", err)
		}
	}
	return &Server{redis: redisClient, httpServer: &http.Server{
		Addr:              fmt.Sprintf("%s:%d", configuration.Host, configuration.Port),
		Handler:           newRouter(logger, service, auth.New(configuration.Auth.Secret, configuration.Auth.Version, configuration.Auth.TokenTTL), configuration.Auth.Enabled, configuration.Streaming, rateLimit(configuration, redisClient), quota(configuration, redisClient)),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}}, nil
}
func rateLimit(c config.Config, redisClient *storage.Client) ratelimit.Limiter {
	if !c.RateLimit.Enabled {
		return nil
	}
	if redisClient != nil {
		return ratelimit.NewRedis(redisClient, c.RateLimit.RequestsPerMinute, c.RateLimit.Burst)
	}
	return ratelimit.New(c.RateLimit.RequestsPerMinute, c.RateLimit.Burst)
}
func quota(c config.Config, redisClient *storage.Client) usage.Tracker {
	if !c.UsageQuota.Enabled {
		return nil
	}
	if redisClient != nil {
		return usage.NewRedis(redisClient, c.UsageQuota.DailyRequests)
	}
	return usage.New(c.UsageQuota.DailyRequests)
}

func (server *Server) ListenAndServe() error { return server.httpServer.ListenAndServe() }
func (server *Server) Shutdown(ctx context.Context) error {
	err := server.httpServer.Shutdown(ctx)
	if server.redis != nil {
		_ = server.redis.Close()
	}
	return err
}
