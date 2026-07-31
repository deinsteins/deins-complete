package handlers

import (
	"deinscomplete/api/internal/account"
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

func RegisterInstallations(a *auth.Service, repo *account.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		var x registration
		if json.NewDecoder(r.Body).Decode(&x) != nil || x.InstallationID == "" || len(x.InstallationID) > 128 {
			response.WriteError(w, 400, "INVALID_REQUEST", "Installation registration is invalid.", middleware.GetRequestID(r.Context()))
			return
		}
		if repo != nil {
			installation, err := repo.EnsureInstallation(r.Context(), x.InstallationID)
			if err != nil {
				response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
				return
			}
			if installation.Status != "active" {
				response.WriteError(w, 401, "INSTALLATION_REVOKED", "Installation has been revoked.", middleware.GetRequestID(r.Context()))
				return
			}
		}
		t, e := a.Issue(x.InstallationID)
		if e != nil {
			response.WriteError(w, 500, "INTERNAL_ERROR", "Internal server error.", middleware.GetRequestID(r.Context()))
			return
		}
		response.WriteJSON(w, 200, map[string]any{"installation": map[string]string{"id": x.InstallationID}, "token": t})
	}
}
