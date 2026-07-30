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
	Provider           string
	Timeout            time.Duration
	MaxTokens          int
	Temperature        float64
	MaxCompletionLines int
	MaxCompletionChars int
	OpenAI             OpenAIConfig
	Anthropic          AnthropicConfig
}

type OpenAIConfig struct{ BaseURL, APIKey, Model string }
type AnthropicConfig struct{ BaseURL, APIKey, Model, Version string }

type Config struct {
	Environment string
	Host        string
	Port        int
	LogLevel    string
	AI          AIConfig
	Auth        AuthConfig
	RateLimit   RateLimitConfig
	UsageQuota  UsageQuotaConfig
	Router      RouterConfig
}
type RouterConfig struct {
	FallbackEnabled bool
	MaxAttempts     int
	Timeout         time.Duration
	Fallback        AIConfig
}
type RateLimitConfig struct {
	Enabled           bool
	RequestsPerMinute int
	Burst             int
}
type UsageQuotaConfig struct {
	Enabled       bool
	DailyRequests int
}
type AuthConfig struct {
	Enabled  bool
	Secret   string
	Version  int
	TokenTTL time.Duration
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
			Provider:           valueOrDefault(lookup("AI_PROVIDER"), "mock"),
			OpenAI:             OpenAIConfig{BaseURL: lookup("AI_BASE_URL"), APIKey: lookup("AI_API_KEY"), Model: lookup("AI_MODEL")},
			Anthropic:          AnthropicConfig{BaseURL: lookup("ANTHROPIC_BASE_URL"), APIKey: lookup("ANTHROPIC_API_KEY"), Model: lookup("ANTHROPIC_MODEL"), Version: lookup("ANTHROPIC_VERSION")},
			Timeout:            10 * time.Second,
			MaxTokens:          128,
			Temperature:        0.1,
			MaxCompletionLines: 20, MaxCompletionChars: 8000,
		},
		Auth:       AuthConfig{Enabled: lookup("AUTH_ENABLED") == "true", Secret: lookup("AUTH_TOKEN_SECRET"), Version: 1},
		RateLimit:  RateLimitConfig{Enabled: lookup("RATE_LIMIT_ENABLED") == "true", RequestsPerMinute: 60, Burst: 10},
		UsageQuota: UsageQuotaConfig{Enabled: lookup("USAGE_QUOTA_ENABLED") == "true", DailyRequests: 2000},
		Router:     RouterConfig{FallbackEnabled: lookup("AI_FALLBACK_ENABLED") == "true", MaxAttempts: 2, Timeout: 8 * time.Second},
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
	if raw := lookup("AUTH_TOKEN_VERSION"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 {
			return Config{}, fmt.Errorf("invalid AUTH_TOKEN_VERSION value: %s", raw)
		}
		config.Auth.Version = v
	}
	if raw := lookup("AUTH_TOKEN_TTL_HOURS"); raw != "" {
		hours, err := strconv.Atoi(raw)
		if err != nil || hours < 0 || hours > 8760 {
			return Config{}, fmt.Errorf("invalid AUTH_TOKEN_TTL_HOURS value: %s", raw)
		}
		config.Auth.TokenTTL = time.Duration(hours) * time.Hour
	}
	if config.Environment == "production" && !config.Auth.Enabled {
		return Config{}, fmt.Errorf("AUTH_ENABLED must be true in production")
	}
	if config.Auth.Enabled && len(config.Auth.Secret) < 32 {
		return Config{}, fmt.Errorf("AUTH_TOKEN_SECRET must be at least 32 bytes when auth is enabled")
	}
	if err := parseAdmissionConfig(&config, lookup); err != nil {
		return Config{}, err
	}
	if err := parseAIConfig(&config.AI, config.Environment, lookup); err != nil {
		return Config{}, err
	}
	if err := parseRouterConfig(&config, lookup); err != nil {
		return Config{}, err
	}
	return config, nil
}
func parseRouterConfig(c *Config, lookup func(string) string) error {
	if raw := lookup("AI_MAX_ROUTER_ATTEMPTS"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 || v > 5 {
			return fmt.Errorf("invalid AI_MAX_ROUTER_ATTEMPTS value: %s", raw)
		}
		c.Router.MaxAttempts = v
	}
	if raw := lookup("AI_ROUTER_TIMEOUT_MS"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1000 || v > 60000 {
			return fmt.Errorf("invalid AI_ROUTER_TIMEOUT_MS value: %s", raw)
		}
		c.Router.Timeout = time.Duration(v) * time.Millisecond
	}
	if !c.Router.FallbackEnabled {
		return nil
	}
	c.Router.Fallback = AIConfig{Provider: lookup("AI_FALLBACK_PROVIDER"), Timeout: c.AI.Timeout, MaxTokens: c.AI.MaxTokens, Temperature: c.AI.Temperature, OpenAI: OpenAIConfig{BaseURL: lookup("AI_FALLBACK_BASE_URL"), APIKey: lookup("AI_FALLBACK_API_KEY"), Model: lookup("AI_FALLBACK_MODEL")}, Anthropic: AnthropicConfig{BaseURL: lookup("AI_FALLBACK_BASE_URL"), APIKey: lookup("AI_FALLBACK_API_KEY"), Model: lookup("AI_FALLBACK_MODEL"), Version: lookup("AI_FALLBACK_VERSION")}}
	if c.Router.Fallback.Provider == "" {
		return fmt.Errorf("AI_FALLBACK_PROVIDER is required when fallback is enabled")
	}
	return parseAIConfig(&c.Router.Fallback, c.Environment, func(k string) string {
		switch k {
		case "AI_PROVIDER":
			return c.Router.Fallback.Provider
		case "AI_BASE_URL":
			return c.Router.Fallback.OpenAI.BaseURL
		case "AI_API_KEY":
			return c.Router.Fallback.OpenAI.APIKey
		case "AI_MODEL":
			return c.Router.Fallback.OpenAI.Model
		case "ANTHROPIC_BASE_URL":
			return c.Router.Fallback.Anthropic.BaseURL
		case "ANTHROPIC_API_KEY":
			return c.Router.Fallback.Anthropic.APIKey
		case "ANTHROPIC_MODEL":
			return c.Router.Fallback.Anthropic.Model
		case "ANTHROPIC_VERSION":
			return c.Router.Fallback.Anthropic.Version
		}
		return ""
	})
}
func parseAdmissionConfig(c *Config, lookup func(string) string) error {
	if raw := lookup("RATE_LIMIT_REQUESTS_PER_MINUTE"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 || v > 10000 {
			return fmt.Errorf("invalid RATE_LIMIT_REQUESTS_PER_MINUTE value: %s", raw)
		}
		c.RateLimit.RequestsPerMinute = v
	}
	if raw := lookup("RATE_LIMIT_BURST"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 || v > 1000 {
			return fmt.Errorf("invalid RATE_LIMIT_BURST value: %s", raw)
		}
		c.RateLimit.Burst = v
	}
	if raw := lookup("USAGE_QUOTA_DAILY_REQUESTS"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 || v > 1000000 {
			return fmt.Errorf("invalid USAGE_QUOTA_DAILY_REQUESTS value: %s", raw)
		}
		c.UsageQuota.DailyRequests = v
	}
	return nil
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
	if raw := lookup("AI_MAX_COMPLETION_LINES"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 || v > 100 {
			return fmt.Errorf("invalid AI_MAX_COMPLETION_LINES value: %s", raw)
		}
		config.MaxCompletionLines = v
	}
	if raw := lookup("AI_MAX_COMPLETION_CHARS"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 100 || v > 20000 {
			return fmt.Errorf("invalid AI_MAX_COMPLETION_CHARS value: %s", raw)
		}
		config.MaxCompletionChars = v
	}
	switch config.Provider {
	case "mock":
		return nil
	case "openai-compatible":
		if config.OpenAI.BaseURL == "" || config.OpenAI.APIKey == "" || config.OpenAI.Model == "" {
			return fmt.Errorf("AI_BASE_URL, AI_API_KEY, and AI_MODEL are required for AI_PROVIDER=openai-compatible")
		}
		return validateProviderURL(&config.OpenAI.BaseURL, environment, "AI_BASE_URL")
	case "anthropic":
		if config.Anthropic.BaseURL == "" || config.Anthropic.APIKey == "" || config.Anthropic.Model == "" || config.Anthropic.Version == "" {
			return fmt.Errorf("ANTHROPIC_BASE_URL, ANTHROPIC_API_KEY, ANTHROPIC_MODEL, and ANTHROPIC_VERSION are required for AI_PROVIDER=anthropic")
		}
		return validateProviderURL(&config.Anthropic.BaseURL, environment, "ANTHROPIC_BASE_URL")
	default:
		return nil
	}
}

func validateProviderURL(value *string, environment, name string) error {
	parsedURL, err := url.Parse(*value)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return fmt.Errorf("invalid %s", name)
	}
	if environment == "production" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%s must use HTTPS in production", name)
	}
	*value = strings.TrimRight(*value, "/")
	return nil
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
