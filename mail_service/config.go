package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

type config struct {
	mailHost string
	mailPort int
	username string
	password string
	fromName string

	natsHost string
	natsPort int
}

func newConfig(envPath string) (*config, error) {
	if err := godotenv.Load(envPath); err != nil {
		return &config{}, fmt.Errorf("no %s file found, err: %v", envPath, err)
	}

	config := &config{
		mailHost: getEnv("MAIL_HOST", "smtp.gmail.com"),
		mailPort: getIntEnv("MAIL_PORT", 465),
		username: getEnv("USERNAME", ""),
		password: getEnv("PASSWORD", ""),
		fromName: getEnv("FROM_NAME", "Test name"),

		natsHost: getEnv("NATS_HOST", "localhost"),
		natsPort: getIntEnv("NATS_PORT", 4222),
	}
	return config, nil
}

func getEnv(key string, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultVal
}

func getIntEnv(key string, defaultVal int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Fatalf("invalid %s format, need number\n", key)
	}
	return defaultVal
}
