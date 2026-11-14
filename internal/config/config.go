package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort     string
	DatabaseURL    string
	Environment    string
	AllowedOrigins []string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("Can't get environment variables \n%v", err)
	}

	config := &Config{
		ServerPort:     env("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		Environment:    env("ENVIRONMENT", "development"),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
	}

	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	port, err := strconv.Atoi(config.ServerPort)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("PORT must be a valid TCP port between 1 and 65535")
	}

	return config, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
