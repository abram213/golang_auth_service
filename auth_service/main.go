package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
)

//todo: write tests

func main() {
	addr := "127.0.0.1:8081"
	ctx, finish := context.WithCancel(context.Background())

	config, err := initConfig("config.env")
	if err != nil {
		log.Fatalln("initial config error:", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		finish()
		log.Fatalln("signal caught. shutting down...")
		wg.Done()
	}(wg)

	if err := startService(ctx, addr, config); err != nil {
		log.Fatalf("can`t start server: %v", err)
	}
	wg.Wait()
}