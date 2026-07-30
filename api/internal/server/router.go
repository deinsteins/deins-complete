package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"deinscomplete/api/internal/auth"
	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/http/handlers"
	"deinscomplete/api/internal/http/middleware"
	"deinscomplete/api/internal/http/response"
	"deinscomplete/api/internal/ratelimit"
	"deinscomplete/api/internal/usage"
)

func newRouter(logger *slog.Logger, service *completion.Service, authService *auth.Service, enabled bool, streaming bool, limiter ratelimit.Limiter, tracker usage.Tracker) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.Recovery(logger), middleware.Logging(logger))
	router.Get("/", handlers.Root)
	router.Get("/health", handlers.Health)
	router.Get("/ready", handlers.Ready)
	router.Post("/v1/installations/register", handlers.RegisterInstallations(authService))
	completionHandler := http.Handler(handlers.NewCompletionHandler(service, logger))
	if enabled {
		if tracker != nil {
			completionHandler = middleware.Quota(tracker)(completionHandler)
		}
		if limiter != nil {
			completionHandler = middleware.RateLimit(limiter)(completionHandler)
		}
		completionHandler = middleware.Auth(authService)(completionHandler)
	}
	router.Post("/v1/completions", completionHandler.ServeHTTP)
	if streaming {
		streamHandler := http.Handler(handlers.NewStreamHandler(service, logger))
		if enabled {
			if tracker != nil {
				streamHandler = middleware.Quota(tracker)(streamHandler)
			}
			if limiter != nil {
				streamHandler = middleware.RateLimit(limiter)(streamHandler)
			}
			streamHandler = middleware.Auth(authService)(streamHandler)
		}
		router.Post("/v1/completions/stream", streamHandler.ServeHTTP)
	}
	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		response.WriteError(writer, http.StatusNotFound, "NOT_FOUND", "Route not found.", middleware.GetRequestID(request.Context()))
	})
	router.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		response.WriteError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed.", middleware.GetRequestID(request.Context()))
	})
	return router
}
