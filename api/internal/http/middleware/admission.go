package middleware

import (
	"deinscomplete/api/internal/http/response"
	"deinscomplete/api/internal/ratelimit"
	"deinscomplete/api/internal/usage"
	"net/http"
	"strconv"
	"time"
)

func RateLimit(l ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := IdentityFromContext(r.Context())
			if !ok {
				response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
				return
			}
			result := l.Allow(r.Context(), id.InstallationID)
			if result.Err != nil {
				response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", GetRequestID(r.Context()))
				return
			}
			if !result.Allowed {
				retry(w, result.RetryAfter)
				response.WriteError(w, 429, "RATE_LIMITED", "Too many completion requests.", GetRequestID(r.Context()))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func Quota(t usage.Tracker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, _ := IdentityFromContext(r.Context())
			result := t.CheckAndConsume(r.Context(), id.InstallationID, 1)
			if result.Err != nil {
				response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", GetRequestID(r.Context()))
				return
			}
			if !result.Allowed {
				retry(w, result.RetryAfter)
				response.WriteError(w, 429, "QUOTA_EXCEEDED", "Daily completion quota exceeded.", GetRequestID(r.Context()))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func retry(w http.ResponseWriter, d time.Duration) {
	seconds := int(d.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
}
