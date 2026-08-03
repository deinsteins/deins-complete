package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"deinscomplete/api/internal/account"
	"deinscomplete/api/internal/accountauth"
	"deinscomplete/api/internal/auth"
	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/config"
	"deinscomplete/api/internal/database"
	"deinscomplete/api/internal/ratelimit"
	"deinscomplete/api/internal/storage"
	"deinscomplete/api/internal/usage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	httpServer *http.Server
	redis      *storage.Client
	database   *pgxpool.Pool
}

func New(configuration config.Config, logger *slog.Logger, service *completion.Service) (*Server, error) {
	var redisClient *storage.Client
	var accountRepo *account.Repository
	var databasePool *pgxpool.Pool
	var err error
	if configuration.Redis.Enabled {
		redisClient, err = storage.NewRedis(configuration.Redis)
		if err != nil {
			return nil, fmt.Errorf("redis unavailable: %w", err)
		}
	}
	if configuration.Database.Enabled {
		pool, openErr := database.Open(context.Background(), database.Config{URL: configuration.Database.URL, MaxOpenConns: configuration.Database.MaxOpenConns, MaxIdleConns: configuration.Database.MaxIdleConns, ConnMaxLifetime: configuration.Database.ConnMaxLifetime})
		if openErr != nil {
			if redisClient != nil {
				_ = redisClient.Close()
			}
			return nil, fmt.Errorf("database unavailable: %w", openErr)
		}
		databasePool = pool
		accountRepo = account.NewRepository(pool)
		if configuration.Quality.Enabled {
			if cleanupErr := accountRepo.DeleteQualityEventsBefore(context.Background(), time.Now().AddDate(0, 0, -configuration.Quality.RetentionDays)); cleanupErr != nil {
				pool.Close()
				if redisClient != nil {
					_ = redisClient.Close()
				}
				return nil, fmt.Errorf("quality event cleanup failed: %w", cleanupErr)
			}
		}
	}
	monthlyTracker := monthlyQuota(redisClient)
	var accountService *account.Service
	var accountTokens *accountauth.Service
	if accountRepo != nil {
		accountTokens = accountauth.New(configuration.Account.AccessTokenSecret, configuration.Account.AccessTokenTTL)
		var mailer account.EmailSender = &account.DevelopmentMailer{}
		if configuration.Account.SMTPAddr != "" {
			mailer = account.SMTPMailer{Addr: configuration.Account.SMTPAddr, From: configuration.Account.SMTPFrom, Username: configuration.Account.SMTPUsername, Password: configuration.Account.SMTPPassword}
		}
		accountService = account.NewService(accountRepo, accountTokens, mailer, configuration.Account.RegistrationMode, configuration.Account.RefreshTokenTTL, configuration.Account.MagicCodeTTL, monthlyTracker)
	}
	adminToken := ""
	if configuration.Admin.Enabled {
		adminToken = configuration.Admin.Token
	}
	return &Server{redis: redisClient, database: databasePool, httpServer: &http.Server{
		Addr:              fmt.Sprintf("%s:%d", configuration.Host, configuration.Port),
		Handler:           newRouter(logger, service, auth.New(configuration.Auth.Secret, configuration.Auth.Version, configuration.Auth.TokenTTL), configuration.Auth.Enabled, configuration.Streaming, rateLimit(configuration, redisClient), quota(configuration, redisClient), monthlyTracker, readiness(redisClient, databasePool), accountRepo, accountService, accountTokens, configuration.Account.Required, adminToken, configuration.Quality.Enabled),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}}, nil
}
func readiness(redisClient *storage.Client, db interface{ Ping(context.Context) error }) func(context.Context) error {
	if redisClient == nil && db == nil {
		return nil
	}
	return func(ctx context.Context) error {
		if redisClient != nil {
			if err := redisClient.Ready(ctx); err != nil {
				return err
			}
		}
		if db != nil {
			return db.Ping(ctx)
		}
		return nil
	}
}
func monthlyQuota(redisClient *storage.Client) usage.MonthlyTracker {
	if redisClient != nil {
		return usage.NewMonthlyRedis(redisClient.Client)
	}
	return usage.NewMonthly()
}
func rateLimit(c config.Config, redisClient *storage.Client) ratelimit.Limiter {
	if !c.RateLimit.Enabled {
		return nil
	}
	if redisClient != nil {
		return ratelimit.NewRedis(redisClient, c.RateLimit.RequestsPerMinute, c.RateLimit.Burst)
	}
	return ratelimit.New(c.RateLimit.RequestsPerMinute, c.RateLimit.Burst)
}
func quota(c config.Config, redisClient *storage.Client) usage.Tracker {
	if !c.UsageQuota.Enabled {
		return nil
	}
	if redisClient != nil {
		return usage.NewRedis(redisClient, c.UsageQuota.DailyRequests)
	}
	return usage.New(c.UsageQuota.DailyRequests)
}

func (server *Server) ListenAndServe() error { return server.httpServer.ListenAndServe() }
func (server *Server) Shutdown(ctx context.Context) error {
	err := server.httpServer.Shutdown(ctx)
	if server.redis != nil {
		_ = server.redis.Close()
	}
	if server.database != nil {
		server.database.Close()
	}
	return err
}
