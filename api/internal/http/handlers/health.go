package handlers

import (
	"context"
	"net/http"

	"deinscomplete/api/internal/http/response"
)

func Root(writer http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(writer, http.StatusOK, map[string]string{"name": "DeinsComplete API", "status": "ok"})
}

func Health(writer http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func Ready(check func(context.Context) error) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if check != nil && check(request.Context()) != nil {
			response.WriteError(writer, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", "")
			return
		}
		response.WriteJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
	}
}
