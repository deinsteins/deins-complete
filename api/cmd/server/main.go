package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/completion/providers"
	"deinscomplete/api/internal/config"
	"deinscomplete/api/internal/logging"
	"deinscomplete/api/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configuration, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(configuration.LogLevel)
	provider, err := providers.NewProvider(configuration.AI, logger)
	if err != nil {
		return err
	}
	service := completion.NewService(provider)
	server := server.New(configuration, logger, service)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server started", "address", fmt.Sprintf("%s:%d", configuration.Host, configuration.Port))
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}
	logger.Info("shutdown started")
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return err
	}
	logger.Info("shutdown complete")
	return nil
}
