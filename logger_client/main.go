package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"logger_client/proto"
	"sync"
)

//todo: make output pretty

func main() {
	config, err := newConfig(".env")
	if err != nil {
		log.Fatalf("can`t init config: %v", err)
	}
	addr := config.authHost + ":" + config.authPort

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalln("can`t connect to grpc:", err)
	}

	adminClient := proto.NewAdminClient(conn)

	//adding key to context
	ctx := ctxWithKey("admin_key")

	logStream, err := adminClient.Logging(ctx, &proto.Nothing{})
	if err != nil {
		log.Fatalln("logging func err:", err)
	}
	fmt.Printf("success connected to %v\n", addr)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			evt, err := logStream.Recv()
			if err != nil {
				log.Fatalf("unexpected error: %v, awaiting event", err)
			}
			fmt.Println(evt) //change output
		}
	}()
	wg.Wait()
}

func ctxWithKey(key string) context.Context {
	md := metadata.Pairs(
		"key", key,
	)
	return metadata.NewOutgoingContext(context.Background(), md)
}
