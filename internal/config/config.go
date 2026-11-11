package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string
	DatabaseURL string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("Can't get environment variables \n%v", )
	}

	config := &Config{
		ServerPort: os.Getenv("PORT"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	return config, nil
} 