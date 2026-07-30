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
	"deinscomplete/api/internal/usage"
)

type Server struct {
	httpServer *http.Server
}

func New(configuration config.Config, logger *slog.Logger, service *completion.Service) *Server {
	return &Server{httpServer: &http.Server{
		Addr:              fmt.Sprintf("%s:%d", configuration.Host, configuration.Port),
		Handler:           newRouter(logger, service, auth.New(configuration.Auth.Secret, configuration.Auth.Version, configuration.Auth.TokenTTL), configuration.Auth.Enabled, rateLimit(configuration), quota(configuration)),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}}
}
func rateLimit(c config.Config) ratelimit.Limiter {
	if !c.RateLimit.Enabled {
		return nil
	}
	return ratelimit.New(c.RateLimit.RequestsPerMinute, c.RateLimit.Burst)
}
func quota(c config.Config) usage.Tracker {
	if !c.UsageQuota.Enabled {
		return nil
	}
	return usage.New(c.UsageQuota.DailyRequests)
}

func (server *Server) ListenAndServe() error              { return server.httpServer.ListenAndServe() }
func (server *Server) Shutdown(ctx context.Context) error { return server.httpServer.Shutdown(ctx) }
