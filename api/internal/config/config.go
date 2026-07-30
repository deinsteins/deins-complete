package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type AIConfig struct {
	Provider    string
	BaseURL     string
	APIKey      string
	Model       string
	Timeout     time.Duration
	MaxTokens   int
	Temperature float64
}

type Config struct {
	Environment string
	Host        string
	Port        int
	LogLevel    string
	AI          AIConfig
}

func Load() (Config, error) {
	return parse(os.Getenv)
}

func parse(lookup func(string) string) (Config, error) {
	config := Config{
		Environment: valueOrDefault(lookup("APP_ENV"), "development"),
		Host:        valueOrDefault(lookup("HOST"), "127.0.0.1"),
		LogLevel:    valueOrDefault(lookup("LOG_LEVEL"), "info"),
		Port:        3001,
		AI: AIConfig{
			Provider:    valueOrDefault(lookup("AI_PROVIDER"), "mock"),
			BaseURL:     lookup("AI_BASE_URL"),
			APIKey:      lookup("AI_API_KEY"),
			Model:       lookup("AI_MODEL"),
			Timeout:     10 * time.Second,
			MaxTokens:   128,
			Temperature: 0.1,
		},
	}

	if rawPort := lookup("PORT"); rawPort != "" {
		port, err := strconv.Atoi(rawPort)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("invalid PORT value: %s", rawPort)
		}
		config.Port = port
	}
	if config.Environment != "development" && config.Environment != "test" && config.Environment != "production" {
		return Config{}, fmt.Errorf("invalid APP_ENV value: %s", config.Environment)
	}
	if config.Host == "" {
		return Config{}, fmt.Errorf("HOST must not be empty")
	}
	if config.LogLevel != "debug" && config.LogLevel != "info" && config.LogLevel != "warn" && config.LogLevel != "error" {
		return Config{}, fmt.Errorf("invalid LOG_LEVEL value: %s", config.LogLevel)
	}
	if err := parseAIConfig(&config.AI, config.Environment, lookup); err != nil {
		return Config{}, err
	}
	return config, nil
}

func parseAIConfig(config *AIConfig, environment string, lookup func(string) string) error {
	if raw := lookup("AI_TIMEOUT_MS"); raw != "" {
		milliseconds, err := strconv.Atoi(raw)
		if err != nil || milliseconds < 1000 || milliseconds > 60000 {
			return fmt.Errorf("invalid AI_TIMEOUT_MS value: %s", raw)
		}
		config.Timeout = time.Duration(milliseconds) * time.Millisecond
	}
	if raw := lookup("AI_MAX_TOKENS"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 2048 {
			return fmt.Errorf("invalid AI_MAX_TOKENS value: %s", raw)
		}
		config.MaxTokens = value
	}
	if raw := lookup("AI_TEMPERATURE"); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || value < 0 || value > 2 {
			return fmt.Errorf("invalid AI_TEMPERATURE value: %s", raw)
		}
		config.Temperature = value
	}
	if config.Provider != "openai-compatible" {
		return nil
	}
	if config.BaseURL == "" || config.APIKey == "" || config.Model == "" {
		return fmt.Errorf("AI_BASE_URL, AI_API_KEY, and AI_MODEL are required for AI_PROVIDER=openai-compatible")
	}
	parsedURL, err := url.Parse(config.BaseURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return fmt.Errorf("invalid AI_BASE_URL")
	}
	if environment == "production" && parsedURL.Scheme != "https" {
		return fmt.Errorf("AI_BASE_URL must use HTTPS in production")
	}
	config.BaseURL = strings.TrimRight(config.BaseURL, "/")
	return nil
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
