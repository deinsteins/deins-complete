package middleware

import (
	"deinscomplete/api/internal/http/response"
	"deinscomplete/api/internal/usage"
	"net/http"
)

func MonthlyQuota(t usage.MonthlyTracker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			i, ok := InstallationFromContext(r.Context())
			e, eok := EntitlementsFromContext(r.Context())
			if !ok || !eok {
				response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
				return
			}
			subject := "installation:" + i.Installation.ID
			if i.Installation.UserID != "" {
				subject = "user:" + i.Installation.UserID
			}
			result := t.CheckAndConsume(r.Context(), subject, e.MonthlyCompletions, 1)
			if result.Err != nil {
				response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", GetRequestID(r.Context()))
				return
			}
			if !result.Allowed {
				retry(w, result.RetryAfter)
				response.WriteError(w, 429, "PLAN_QUOTA_EXCEEDED", "Monthly completion quota exceeded.", GetRequestID(r.Context()))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
