package providers

import (
	"fmt"
	"log/slog"

	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/completion/providers/openai"
	"deinscomplete/api/internal/config"
)

func NewProvider(configuration config.AIConfig, logger *slog.Logger) (Provider, error) {
	switch configuration.Provider {
	case "mock":
		return MockProvider{}, nil
	case "openai-compatible":
		return openai.New(configuration, logger)
	default:
		return nil, fmt.Errorf("unsupported AI_PROVIDER: %s", configuration.Provider)
	}
}

var _ completion.Provider = MockProvider{}
