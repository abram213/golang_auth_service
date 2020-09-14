package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

type DB struct {
	User string
	Pass string
	Name string
	Host string
	Port string
}

type Config struct {
	Port         string
	AuthGRPCPort string
	Db           DB
}

func New(envPath string) (*Config, error) {
	if err := godotenv.Load(envPath); err != nil {
		return &Config{}, fmt.Errorf("no %s file found, err: %v", envPath, err)
	}

	config := &Config{
		Port:         getEnv("HTTP_PORT", "8080"),
		AuthGRPCPort: getEnv("AUTH_GRPC_PORT", ""),
		Db: DB{
			Host: getEnv("DB_HOST", "8080"),
			User: getEnv("DB_USER", "8080"),
			Pass: getEnv("DB_PASSWORD", "8080"),
			Name: getEnv("DB_NAME", "8080"),
			Port: getEnv("DB_PORT", "8080"),
		},
	}
	return config, nil
}

func getEnv(key string, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultVal
}
