package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/http/handlers"
	"deinscomplete/api/internal/http/middleware"
	"deinscomplete/api/internal/http/response"
)

func newRouter(logger *slog.Logger, service *completion.Service) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.Recovery(logger), middleware.Logging(logger))
	router.Get("/", handlers.Root)
	router.Get("/health", handlers.Health)
	router.Get("/ready", handlers.Ready)
	router.Post("/v1/completions", handlers.NewCompletionHandler(service, logger).ServeHTTP)
	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		response.WriteError(writer, http.StatusNotFound, "NOT_FOUND", "Route not found.", middleware.GetRequestID(request.Context()))
	})
	router.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		response.WriteError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed.", middleware.GetRequestID(request.Context()))
	})
	return router
}
