package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

type config struct {
	authPort string
	authHost string
}

func newConfig(envPath string) (*config, error) {
	if err := godotenv.Load(envPath); err != nil {
		return &config{}, fmt.Errorf("no %s file found, err: %v", envPath, err)
	}

	config := &config{
		authPort: getEnv("AUTH_PORT", "8081"),
		authHost: getEnv("AUTH_HOST", "localhost"),
	}
	return config, nil
}

func getEnv(key string, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultVal
}
