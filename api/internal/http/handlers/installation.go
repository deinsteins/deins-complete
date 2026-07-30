package handlers

import (
	"deinscomplete/api/internal/auth"
	"deinscomplete/api/internal/http/middleware"
	"deinscomplete/api/internal/http/response"
	"encoding/json"
	"net/http"
)

type registration struct {
	InstallationID string `json:"installationId"`
	Client         *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"client,omitempty"`
}

func RegisterInstallations(a *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var x registration
		if json.NewDecoder(r.Body).Decode(&x) != nil || x.InstallationID == "" || len(x.InstallationID) > 128 {
			response.WriteError(w, 400, "INVALID_REQUEST", "Installation registration is invalid.", middleware.GetRequestID(r.Context()))
			return
		}
		t, e := a.Issue(x.InstallationID)
		if e != nil {
			response.WriteError(w, 500, "INTERNAL_ERROR", "Internal server error.", middleware.GetRequestID(r.Context()))
			return
		}
		response.WriteJSON(w, 200, map[string]any{"installation": map[string]string{"id": x.InstallationID}, "token": t})
	}
}
