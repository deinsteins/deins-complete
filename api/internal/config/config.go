package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Environment string
	Host        string
	Port        int
	LogLevel    string
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
	return config, nil
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
