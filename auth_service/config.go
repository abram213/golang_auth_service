package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

type jwtConfig struct {
	accessKey  		string
	refreshKey 		string
	accessExpMin 	int
	refreshExpMin 	int
}

func initConfig(envPath string) (jwtConfig, error) {
	workDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return jwtConfig{}, fmt.Errorf("can`t find wokr dir, err: %v", err)
	}
	envPath = filepath.Join(workDir, envPath)
	if err := godotenv.Load(envPath); err != nil {
		return jwtConfig{}, fmt.Errorf("no %s file found, err: %v", envPath, err)
	}

	config := jwtConfig{
		accessKey:  	getEnv("ACCESS_KEY", "access_key"),
		refreshKey: 	getEnv("REFRESH_KEY", "refresh_key"),
		accessExpMin:   getIntEnv("ACCESS_EXP_MIN", 60),
		refreshExpMin:  getIntEnv("REFRESH_EXP_MIN", 1440),
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