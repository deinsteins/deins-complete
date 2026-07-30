package config

import "testing"

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
	if err != nil || configuration.AI.BaseURL != "https://api.example.com/v1" || configuration.AI.Timeout.Milliseconds() != 2000 {
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
