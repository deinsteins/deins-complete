package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			response := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
			next.ServeHTTP(response, request)
			logger.Info("request completed",
				"request_id", GetRequestID(request.Context()),
				"method", request.Method,
				"path", request.URL.Path,
				"status", response.status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (writer *statusWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}
