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
