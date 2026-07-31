package middleware

import (
	"context"
	"deinscomplete/api/internal/accountauth"
	"deinscomplete/api/internal/auth"
	"deinscomplete/api/internal/http/response"
	"net/http"
	"strings"
)

type identityKey struct{}
type userIdentityKey struct{}
type Identity struct{ InstallationID string }
type UserIdentity struct{ UserID string }

func Auth(service *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			credentials := []string{r.Header.Get("X-DeinsComplete-Installation-Token")}
			if strings.HasPrefix(h, "Bearer ") {
				credentials = append([]string{strings.TrimPrefix(h, "Bearer ")}, credentials...)
			}
			for _, credential := range credentials {
				if credential == "" {
					continue
				}
				if id, err := service.Validate(credential); err == nil {
					next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, Identity{InstallationID: id})))
					return
				}
			}
			response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
		})
	}
}
func UserAuth(service *accountauth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
				return
			}
			id, err := service.ValidateAccessToken(strings.TrimPrefix(h, "Bearer "))
			if err != nil {
				response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIdentityKey{}, UserIdentity{UserID: id})))
		})
	}
}
func IdentityFromContext(c context.Context) (Identity, bool) {
	v, ok := c.Value(identityKey{}).(Identity)
	return v, ok
}
func UserIdentityFromContext(c context.Context) (UserIdentity, bool) {
	v, ok := c.Value(userIdentityKey{}).(UserIdentity)
	return v, ok
}
