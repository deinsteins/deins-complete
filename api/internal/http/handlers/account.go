package handlers

import (
	"deinscomplete/api/internal/account"
	"deinscomplete/api/internal/http/middleware"
	"deinscomplete/api/internal/http/response"
	"deinscomplete/api/internal/usage"
	"encoding/json"
	"errors"
	"net/http"
)

type AccountHandler struct {
	service *account.Service
	monthly usage.MonthlyTracker
}

func AccountRequirement(required bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.WriteJSON(w, http.StatusOK, map[string]bool{"accountRequired": required})
	}
}

func NewAccountHandler(s *account.Service, m usage.MonthlyTracker) *AccountHandler {
	return &AccountHandler{service: s, monthly: m}
}
func (h *AccountHandler) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(v) == nil && ensureSingleJSONValue(d) == nil
}
func (h *AccountHandler) MagicRequest(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Email      string `json:"email"`
		InviteCode string `json:"inviteCode"`
	}
	if !h.decode(w, r, &b) {
		response.WriteError(w, 400, "INVALID_REQUEST", "Request is invalid.", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.RequestMagicCode(r.Context(), b.Email, b.InviteCode); err != nil {
		response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}
func (h *AccountHandler) MagicVerify(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if !h.decode(w, r, &b) {
		response.WriteError(w, 400, "INVALID_REQUEST", "Request is invalid.", middleware.GetRequestID(r.Context()))
		return
	}
	p, err := h.service.VerifyMagicCode(r.Context(), b.Email, b.Code)
	if err != nil {
		response.WriteError(w, 401, "INVALID_CREDENTIALS", "Invalid credentials.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, 200, pairJSON(p))
}
func (h *AccountHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var b struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !h.decode(w, r, &b) {
		response.WriteError(w, 400, "INVALID_REQUEST", "Request is invalid.", middleware.GetRequestID(r.Context()))
		return
	}
	p, err := h.service.Refresh(r.Context(), b.RefreshToken)
	if err != nil {
		response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, 200, pairJSON(p))
}
func pairJSON(p account.TokenPair) map[string]any {
	return map[string]any{"accessToken": p.AccessToken, "refreshToken": p.RefreshToken, "expiresIn": p.ExpiresIn}
}
func (h *AccountHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var b struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !h.decode(w, r, &b) {
		response.WriteError(w, 400, "INVALID_REQUEST", "Request is invalid.", middleware.GetRequestID(r.Context()))
		return
	}
	_ = h.service.Logout(r.Context(), b.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}
func (h *AccountHandler) Account(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIdentityFromContext(r.Context())
	u, err := h.service.User(r.Context(), id.UserID)
	if err != nil {
		response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	e, err := h.service.Entitlements(r.Context(), id.UserID)
	if err != nil {
		response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, 200, map[string]any{"user": map[string]string{"id": u.ID, "email": u.Email}, "plan": map[string]string{"code": e.Code}})
}
func (h *AccountHandler) Entitlements(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIdentityFromContext(r.Context())
	e, err := h.service.Entitlements(r.Context(), id.UserID)
	if err != nil {
		response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	used := 0
	if h.monthly != nil {
		used, err = h.monthly.Usage(r.Context(), "user:"+id.UserID)
		if err != nil {
			response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
			return
		}
	}
	response.WriteJSON(w, 200, map[string]any{"plan": e.Code, "features": map[string]bool{"repositoryContext": e.RepositoryContext, "streaming": e.Streaming, "premiumRouting": e.PremiumRouting}, "limits": map[string]int{"monthlyCompletions": e.MonthlyCompletions, "used": used, "remaining": max(0, e.MonthlyCompletions-used)}})
}
func (h *AccountHandler) Installations(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIdentityFromContext(r.Context())
	xs, err := h.service.Installations(r.Context(), id.UserID)
	if err != nil {
		response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	result := make([]map[string]any, 0, len(xs))
	for _, x := range xs {
		result = append(result, map[string]any{"id": x.ID, "status": x.Status, "createdAt": x.CreatedAt, "lastSeenAt": x.LastSeenAt})
	}
	response.WriteJSON(w, 200, result)
}
func (h *AccountHandler) Link(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.UserIdentityFromContext(r.Context())
	i, ok := middleware.InstallationFromContext(r.Context())
	if !ok {
		response.WriteError(w, 401, "UNAUTHORIZED", "Installation authentication required.", middleware.GetRequestID(r.Context()))
		return
	}
	x, err := h.service.LinkInstallation(r.Context(), u.UserID, i.Installation.ID)
	if err != nil {
		code := "LINK_FAILED"
		status := http.StatusConflict
		if errors.Is(err, account.ErrInstallationRevoked) {
			code = "INSTALLATION_REVOKED"
			status = 401
		}
		response.WriteError(w, status, code, "Installation cannot be linked.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, 200, map[string]any{"id": x.ID, "status": x.Status})
}
func (h *AccountHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.UserIdentityFromContext(r.Context())
	id := r.PathValue("id")
	if id == "" || h.service.RevokeInstallation(r.Context(), u.UserID, id) != nil {
		response.WriteError(w, 404, "NOT_FOUND", "Installation not found.", middleware.GetRequestID(r.Context()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
