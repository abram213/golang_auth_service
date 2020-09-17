package main

import (
	"auth_service/config"
	"auth_service/service"
	"auth_service/storage"
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
)

//todo: write tests

func main() {
	ctx, finish := context.WithCancel(context.Background())

	conf, err := config.InitConfig(".env")
	if err != nil {
		log.Fatalln("initial config error:", err)
	}

	//connect to storage
	db, err := storage.New(*conf)
	if err != nil {
		log.Fatalln("can`t create new storage:", err)
	}
	defer db.Close()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		finish()
		log.Fatalln("signal caught. shutting down...")
	}(wg)

	addr := conf.Host + ":" + conf.Port
	if err := service.StartService(ctx, addr, conf, db); err != nil {
		log.Fatalf("can`t start server: %v", err)
	}
	wg.Wait()
}
