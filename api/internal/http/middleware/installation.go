package middleware

import (
	"context"
	"deinscomplete/api/internal/account"
	"deinscomplete/api/internal/auth"
	"deinscomplete/api/internal/http/response"
	"net/http"
	"strings"
)

type installationKey struct{}
type entitlementKey struct{}
type InstallationIdentity struct{ Installation account.Installation }

func InstallationStatus(repo *account.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := IdentityFromContext(r.Context())
			if !ok {
				response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
				return
			}
			installation, err := repo.EnsureInstallation(r.Context(), id.InstallationID)
			if err != nil {
				response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", GetRequestID(r.Context()))
				return
			}
			if installation.Status != "active" {
				response.WriteError(w, 401, "INSTALLATION_REVOKED", "Installation has been revoked.", GetRequestID(r.Context()))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), installationKey{}, InstallationIdentity{installation})))
		})
	}
}
func InstallationToken(service *auth.Service, repo *account.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("X-DeinsComplete-Installation-Token")
			if raw == "" {
				response.WriteError(w, 401, "UNAUTHORIZED", "Installation authentication required.", GetRequestID(r.Context()))
				return
			}
			key, err := service.Validate(raw)
			if err != nil {
				response.WriteError(w, 401, "UNAUTHORIZED", "Installation authentication required.", GetRequestID(r.Context()))
				return
			}
			installation, err := repo.EnsureInstallation(r.Context(), key)
			if err != nil || installation.Status != "active" {
				response.WriteError(w, 401, "INSTALLATION_REVOKED", "Installation has been revoked.", GetRequestID(r.Context()))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), installationKey{}, InstallationIdentity{installation})))
		})
	}
}
func InstallationFromContext(ctx context.Context) (InstallationIdentity, bool) {
	v, ok := ctx.Value(installationKey{}).(InstallationIdentity)
	return v, ok
}
func Entitlements(repo *account.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			i, ok := InstallationFromContext(r.Context())
			if !ok {
				response.WriteError(w, 401, "UNAUTHORIZED", "Authentication required.", GetRequestID(r.Context()))
				return
			}
			e, err := repo.ResolveEntitlementsForInstallation(r.Context(), i.Installation.ID)
			if err != nil {
				response.WriteError(w, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", GetRequestID(r.Context()))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), entitlementKey{}, e)))
		})
	}
}
func EntitlementsFromContext(ctx context.Context) (account.Entitlements, bool) {
	v, ok := ctx.Value(entitlementKey{}).(account.Entitlements)
	return v, ok
}
func HasBearer(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ")
}
