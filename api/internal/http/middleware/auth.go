package middleware

import (
	"context"
	"deinscomplete/api/internal/auth"
	"deinscomplete/api/internal/http/response"
	"net/http"
	"strings"
)

type identityKey struct{}
type Identity struct{ InstallationID string }

func Auth(service *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
				return
			}
			id, e := service.Validate(strings.TrimPrefix(h, "Bearer "))
			if e != nil {
				response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, Identity{InstallationID: id})))
		})
	}
}
func IdentityFromContext(c context.Context) (Identity, bool) {
	v, ok := c.Value(identityKey{}).(Identity)
	return v, ok
}
