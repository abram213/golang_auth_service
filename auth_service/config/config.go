package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

type Config struct {
	AccessKey     string
	RefreshKey    string
	AccessExpMin  int
	RefreshExpMin int
	Port          string
	Host          string
	Db            DbConf
}

type DbConf struct {
	Host string
	User string
	Pass string
	Name string
	Port string
}

func InitConfig(envPath string) (*Config, error) {
	if err := godotenv.Load(envPath); err != nil {
		return &Config{}, fmt.Errorf("no %s file found, err: %v", envPath, err)
	}

	config := &Config{
		AccessKey:     getEnv("ACCESS_KEY", "access_key"),
		RefreshKey:    getEnv("REFRESH_KEY", "refresh_key"),
		AccessExpMin:  getIntEnv("ACCESS_EXP_MIN", 60),
		RefreshExpMin: getIntEnv("REFRESH_EXP_MIN", 1440),
		Port:          getEnv("PORT", "8081"),
		Host:          getEnv("HOST", "localhost"),
		Db: DbConf{
			Host: getEnv("DB_HOST", "localhost"),
			User: getEnv("DB_USER", "postgres"),
			Pass: getEnv("DB_PASSWORD", "123456"),
			Name: getEnv("DB_NAME", "postgres"),
			Port: getEnv("DB_PORT", "5432"),
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

func getIntEnv(key string, defaultVal int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Fatalf("invalid %s format, need number\n", key)
	}
	return defaultVal
}
