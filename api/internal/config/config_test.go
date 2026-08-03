package config

import (
	"testing"
	"time"
)

func TestParseDefaults(t *testing.T) {
	configuration, err := parse(func(string) string { return "" })
	if err != nil || configuration.Port != 3001 || configuration.Environment != "development" {
		t.Fatalf("unexpected configuration: %#v, %v", configuration, err)
	}
}

func TestParseRejectsInvalidPort(t *testing.T) {
	_, err := parse(func(key string) string {
		if key == "PORT" {
			return "99999"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestParseValidatesOpenAICompatibleConfiguration(t *testing.T) {
	values := map[string]string{"AI_PROVIDER": "openai-compatible", "AI_BASE_URL": "https://api.example.com/v1/", "AI_API_KEY": "key", "AI_MODEL": "model", "AI_TIMEOUT_MS": "2000", "AI_MAX_TOKENS": "64", "AI_TEMPERATURE": "0.2"}
	configuration, err := parse(func(key string) string { return values[key] })
	if err != nil || configuration.AI.OpenAI.BaseURL != "https://api.example.com/v1" || configuration.AI.Timeout.Milliseconds() != 2000 {
		t.Fatalf("got %#v, %v", configuration, err)
	}
}

func TestParseRejectsMissingOpenAICredentials(t *testing.T) {
	_, err := parse(func(key string) string {
		if key == "AI_PROVIDER" {
			return "openai-compatible"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected AI configuration error")
	}
}

func TestParseValidatesOnlyActiveAnthropicConfiguration(t *testing.T) {
	values := map[string]string{"AI_PROVIDER": "anthropic", "ANTHROPIC_BASE_URL": "https://api.anthropic.com/", "ANTHROPIC_API_KEY": "key", "ANTHROPIC_MODEL": "model", "ANTHROPIC_VERSION": "2023-06-01"}
	configuration, err := parse(func(key string) string { return values[key] })
	if err != nil || configuration.AI.Anthropic.BaseURL != "https://api.anthropic.com" {
		t.Fatalf("got %#v, %v", configuration, err)
	}
	_, err = parse(func(key string) string {
		if key == "AI_PROVIDER" {
			return "anthropic"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected Anthropic configuration error")
	}
}

func TestParseRedisConfiguration(t *testing.T) {
	values := map[string]string{
		"REDIS_ENABLED":            "true",
		"REDIS_ADDR":               "redis:6379",
		"REDIS_DB":                 "3",
		"REDIS_TLS_ENABLED":        "false",
		"REDIS_CONNECT_TIMEOUT_MS": "500",
		"REDIS_READ_TIMEOUT_MS":    "250",
		"REDIS_WRITE_TIMEOUT_MS":   "300",
	}
	configuration, err := parse(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Redis.DB != 3 ||
		configuration.Redis.ConnectTimeout != 500*time.Millisecond ||
		configuration.Redis.ReadTimeout != 250*time.Millisecond ||
		configuration.Redis.WriteTimeout != 300*time.Millisecond {
		t.Fatalf("unexpected Redis configuration: %#v", configuration.Redis)
	}
}

func TestParseRejectsInvalidRedisConfiguration(t *testing.T) {
	for key, value := range map[string]string{
		"REDIS_DB":                 "-1",
		"REDIS_TLS_ENABLED":        "sometimes",
		"REDIS_CONNECT_TIMEOUT_MS": "20",
	} {
		t.Run(key, func(t *testing.T) {
			_, err := parse(func(candidate string) string {
				if candidate == key {
					return value
				}
				return ""
			})
			if err == nil {
				t.Fatalf("expected %s validation error", key)
			}
		})
	}
}

func TestParseFIMFilenameContextIsExplicit(t *testing.T) {
	values := map[string]string{
		"AI_FIM_FILENAME_CONTEXT_ENABLED": "true",
	}
	configuration, err := parse(func(key string) string { return values[key] })
	if err != nil || !configuration.AI.FIMFilenameContext {
		t.Fatalf("unexpected FIM filename configuration: %#v, %v", configuration.AI, err)
	}

	values["AI_FIM_FILENAME_CONTEXT_ENABLED"] = "yes"
	if _, err := parse(func(key string) string { return values[key] }); err == nil {
		t.Fatal("expected invalid FIM filename context error")
	}
}

func TestParseDatabaseAccountConfiguration(t *testing.T) {
	values := map[string]string{
		"DATABASE_ENABLED":               "true",
		"DATABASE_URL":                   "postgres://user:password@localhost/deinscomplete",
		"ACCOUNT_ACCESS_TOKEN_SECRET":    "01234567890123456789012345678901",
		"REGISTRATION_MODE":              "invite",
		"DATABASE_MAX_OPEN_CONNS":        "12",
		"DATABASE_MAX_IDLE_CONNS":        "4",
		"ACCOUNT_REFRESH_TOKEN_TTL_DAYS": "14",
		"ACCOUNT_MAGIC_CODE_TTL_MINUTES": "10",
	}
	configuration, err := parse(func(key string) string { return values[key] })
	if err != nil || configuration.Database.MaxOpenConns != 12 || configuration.Account.RefreshTokenTTL != 14*24*time.Hour || configuration.Account.MagicCodeTTL != 10*time.Minute {
		t.Fatalf("unexpected account database configuration: %#v %#v %v", configuration.Database, configuration.Account, err)
	}
}

func TestParseRejectsIncompleteDatabaseAccountConfiguration(t *testing.T) {
	_, err := parse(func(key string) string {
		if key == "DATABASE_ENABLED" {
			return "true"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected database configuration error")
	}
}

func TestParseAdminConfiguration(t *testing.T) {
	values := map[string]string{
		"DATABASE_ENABLED":            "true",
		"DATABASE_URL":                "postgres://user:password@localhost/deinscomplete",
		"ACCOUNT_ACCESS_TOKEN_SECRET": "01234567890123456789012345678901",
		"ADMIN_ENABLED":               "true",
		"ADMIN_TOKEN":                 "admin-token-must-be-at-least-32-bytes",
	}
	configuration, err := parse(func(key string) string { return values[key] })
	if err != nil || !configuration.Admin.Enabled {
		t.Fatalf("unexpected admin configuration: %#v %v", configuration.Admin, err)
	}
	values["ADMIN_TOKEN"] = "short"
	if _, err := parse(func(key string) string { return values[key] }); err == nil {
		t.Fatal("expected weak admin token error")
	}
}
