package main

import (
	"auth_client/app"
	"auth_client/config"
	"log"
	"os"
)

//todo: write tests,

func main() {
	logger := log.New(os.Stdout, "", log.Lshortfile)

	conf, err := config.New(".env")
	if err != nil {
		logger.Fatalf("init config err: %v", err)
	}

	a, err := app.New(conf, logger)
	if err != nil {
		logger.Fatalf("creating app err: %v", err)
	}
	a.Run()
}
