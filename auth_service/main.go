package main

import (
	"auth_service/config"
	"auth_service/service"
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
	if err := service.StartService(ctx, addr, conf); err != nil {
		log.Fatalf("can`t start server: %v", err)
	}
	wg.Wait()
}
