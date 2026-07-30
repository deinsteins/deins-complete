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
)

type Server struct {
	httpServer *http.Server
}

func New(configuration config.Config, logger *slog.Logger, service *completion.Service) *Server {
	return &Server{httpServer: &http.Server{
		Addr:              fmt.Sprintf("%s:%d", configuration.Host, configuration.Port),
		Handler:           newRouter(logger, service, auth.New(configuration.Auth.Secret, configuration.Auth.Version), configuration.Auth.Enabled),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}}
}

func (server *Server) ListenAndServe() error              { return server.httpServer.ListenAndServe() }
func (server *Server) Shutdown(ctx context.Context) error { return server.httpServer.Shutdown(ctx) }
