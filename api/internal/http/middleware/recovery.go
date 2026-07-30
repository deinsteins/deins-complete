package middleware

import (
	"log/slog"
	"net/http"

	"deinscomplete/api/internal/http/response"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("request panic", "request_id", GetRequestID(request.Context()), "panic", recovered)
					response.WriteError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", GetRequestID(request.Context()))
				}
			}()
			next.ServeHTTP(writer, request)
		})
	}
}
