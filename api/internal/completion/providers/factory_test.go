package providers

import (
	"io"
	"log/slog"
	"testing"

	"deinscomplete/api/internal/config"
)

func TestFactorySelectsMockProvider(t *testing.T) {
	provider, err := NewProvider(config.AIConfig{Provider: "mock"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := provider.(MockProvider); !ok {
		t.Fatalf("got %T", provider)
	}
}

func TestFactoryRejectsUnknownProvider(t *testing.T) {
	_, err := NewProvider(config.AIConfig{Provider: "unknown"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error")
	}
}
