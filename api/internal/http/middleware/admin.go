package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"deinscomplete/api/internal/http/response"
	"net/http"
	"strings"
)

func AdminAuth(token string) func(http.Handler) http.Handler {
	expected := sha256.Sum256([]byte(token))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			candidate := r.Header.Get("X-DeinsComplete-Admin-Token")
			if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
				candidate = strings.TrimPrefix(h, "Bearer ")
			}
			actual := sha256.Sum256([]byte(candidate))
			if candidate == "" || subtle.ConstantTimeCompare(expected[:], actual[:]) != 1 {
				response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
