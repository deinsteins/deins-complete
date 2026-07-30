package handlers

import (
	"net/http"

	"deinscomplete/api/internal/http/response"
)

func Root(writer http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(writer, http.StatusOK, map[string]string{"name": "DeinsComplete API", "status": "ok"})
}

func Health(writer http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func Ready(writer http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}
