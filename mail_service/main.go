package main

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"gopkg.in/gomail.v2"
	"log"
	"os"
	"os/signal"
)

func main() {
	config, err := newConfig(".env")
	if err != nil {
		log.Fatalf("can`t init config: %v", err)
	}

	ms, err := newMailService(*config)
	if err != nil {
		log.Fatalf("init mail server err: %v", err)
	}

	mailCh := make(chan *gomail.Message)
	ctx, cancel := context.WithCancel(context.Background())

	go listenQueue(ctx, *config, mailCh)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		cancel()
	}()

	if err := ms.serve(ctx, mailCh); err != nil {
		log.Fatalf("mail server err: %v", err)
	}
}

type mail struct {
	Email string
	Title string
	Body  string
}

func listenQueue(ctx context.Context, conf config, out chan<- *gomail.Message) {
	natsURL := fmt.Sprintf("nats://%s:%v", conf.natsHost, conf.natsPort)
	conn, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("conn to nats server err: %v\n", err)
	}

	eConn, err := nats.NewEncodedConn(conn, nats.JSON_ENCODER)
	if err != nil {
		panic(err)
	}
	defer eConn.Close()
	log.Printf("connected to nats server: %v\n", natsURL)

	in := make(chan *mail, 64)
	chanSub, err := eConn.BindRecvChan("emails", in)
	for {
		select {
		case data, ok := <-in:
			if !ok {
				log.Println("can`t get value from channel")
				continue
			}
			m := gomail.NewMessage()
			m.SetAddressHeader("From", conf.username, conf.fromName)
			m.SetAddressHeader("To", data.Email, data.Email)
			m.SetHeader("Subject", data.Title)
			m.SetBody("text/plain", data.Body)
			out <- m
		case <-ctx.Done():
			if err := chanSub.Unsubscribe(); err != nil {
				log.Fatalf("unsubscribe err: %v\n", err)
			}
			return
		}
	}
}
