package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	HttpPort string
	AuthPort string
	AuthHost string
}

func New(envPath string) (*Config, error) {
	if err := godotenv.Load(envPath); err != nil {
		return &Config{}, fmt.Errorf("no %s file found, err: %v", envPath, err)
	}

	config := &Config{
		HttpPort: getEnv("HTTP_PORT", "8082"),
		AuthPort: getEnv("AUTH_PORT", "8081"),
		AuthHost: getEnv("AUTH_HOST", "localhost"),
	}
	return config, nil
}

func getEnv(key string, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultVal
}
