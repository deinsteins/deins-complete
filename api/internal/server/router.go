package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"deinscomplete/api/internal/account"
	"deinscomplete/api/internal/accountauth"
	"deinscomplete/api/internal/auth"
	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/http/handlers"
	"deinscomplete/api/internal/http/middleware"
	"deinscomplete/api/internal/http/response"
	"deinscomplete/api/internal/ratelimit"
	"deinscomplete/api/internal/usage"
)

func newRouter(logger *slog.Logger, service *completion.Service, authService *auth.Service, enabled bool, streaming bool, limiter ratelimit.Limiter, tracker usage.Tracker, monthly usage.MonthlyTracker, readiness func(context.Context) error, repo *account.Repository, accounts *account.Service, accountTokens *accountauth.Service, accountRequired bool, adminToken string, qualityEnabled bool, qualitySamplePercent int) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.Recovery(logger), middleware.Logging(logger))
	router.Get("/", handlers.Root)
	router.Get("/health", handlers.Health)
	router.Get("/ready", handlers.Ready(readiness))
	router.Post("/v1/installations/register", handlers.RegisterInstallations(authService, repo))
	router.Get("/v1/account/requirement", handlers.AccountRequirement(accountRequired))
	if accounts != nil && accountTokens != nil && repo != nil {
		h := handlers.NewAccountHandler(accounts, monthly)
		authRoutes := chi.NewRouter()
		authRoutes.Use(middleware.PublicRateLimit(ratelimit.New(10, 5)))
		authRoutes.Post("/v1/auth/magic/request", h.MagicRequest)
		authRoutes.Post("/v1/auth/magic/verify", h.MagicVerify)
		authRoutes.Post("/v1/auth/refresh", h.Refresh)
		authRoutes.Post("/v1/auth/logout", h.Logout)
		router.Mount("/", authRoutes)
		user := middleware.UserAuth(accountTokens)
		router.With(user).Get("/v1/account", h.Account)
		router.With(user).Get("/v1/account/entitlements", h.Entitlements)
		router.With(user).Get("/v1/account/installations", h.Installations)
		router.With(user).Delete("/v1/account/installations/{id}", h.Revoke)
		router.With(user, middleware.InstallationToken(authService, repo)).Post("/v1/installations/link", h.Link)
	}
	if repo != nil && adminToken != "" {
		h := handlers.NewAdminHandler(repo, monthly, qualitySamplePercent)
		admin := middleware.AdminAuth(adminToken)
		router.Get("/admin", handlers.AdminPage)
		router.With(admin).Get("/v1/admin/overview", h.Overview)
		router.With(admin).Get("/v1/admin/users", h.Users)
		router.With(admin).Get("/v1/admin/installations", h.Installations)
		router.With(admin).Get("/v1/admin/invites", h.Invites)
		router.With(admin).Get("/v1/admin/quality", h.Quality)
		router.With(admin).Post("/v1/admin/invites", h.CreateInvite)
		router.With(admin).Post("/v1/admin/users/{id}/plan", h.SetPlan)
		router.With(admin).Post("/v1/admin/installations/{id}/revoke", h.RevokeInstallation)
	}
	if qualityEnabled && repo != nil && enabled {
		qualityHandler := handlers.NewQualityHandler(repo, qualitySamplePercent)
		quality := http.Handler(http.HandlerFunc(qualityHandler.Record))
		quality = middleware.InstallationStatus(repo)(quality)
		quality = middleware.Auth(authService)(quality)
		quality = middleware.PublicRateLimit(ratelimit.New(240, 40))(quality)
		router.Post("/v1/quality/events", quality.ServeHTTP)
	}
	completionHandler := http.Handler(handlers.NewCompletionHandler(service, logger))
	if enabled {
		if repo != nil {
			if monthly != nil {
				completionHandler = middleware.MonthlyQuota(monthly)(completionHandler)
			}
			completionHandler = middleware.Entitlements(repo)(completionHandler)
			completionHandler = middleware.RequireLinkedAccount(accountRequired)(completionHandler)
			completionHandler = middleware.InstallationStatus(repo)(completionHandler)
		}
		if tracker != nil {
			completionHandler = middleware.Quota(tracker)(completionHandler)
		}
		if limiter != nil {
			completionHandler = middleware.RateLimit(limiter)(completionHandler)
		}
		completionHandler = middleware.Auth(authService)(completionHandler)
	}
	router.Post("/v1/completions", completionHandler.ServeHTTP)
	if streaming {
		streamHandler := http.Handler(handlers.NewStreamHandler(service, logger))
		if enabled {
			if repo != nil {
				if monthly != nil {
					streamHandler = middleware.MonthlyQuota(monthly)(streamHandler)
				}
				streamHandler = middleware.Entitlements(repo)(streamHandler)
				streamHandler = middleware.RequireLinkedAccount(accountRequired)(streamHandler)
				streamHandler = middleware.InstallationStatus(repo)(streamHandler)
			}
			if tracker != nil {
				streamHandler = middleware.Quota(tracker)(streamHandler)
			}
			if limiter != nil {
				streamHandler = middleware.RateLimit(limiter)(streamHandler)
			}
			streamHandler = middleware.Auth(authService)(streamHandler)
		}
		router.Post("/v1/completions/stream", streamHandler.ServeHTTP)
	}
	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		response.WriteError(writer, http.StatusNotFound, "NOT_FOUND", "Route not found.", middleware.GetRequestID(request.Context()))
	})
	router.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		response.WriteError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed.", middleware.GetRequestID(request.Context()))
	})
	return router
}
