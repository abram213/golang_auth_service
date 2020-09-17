package main

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"log"
	"logger_client/proto"
	"sync"
	"time"
)

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
	defer conn.Close()

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

			eventTime := time.Unix(evt.Timestamp, 0).Format(time.RFC3339)
			marker := getColorMarker(evt.Code)
			fmt.Printf("%s %v | %s | %s | %s | %s\n",
				marker, eventTime, evt.Host, evt.Method, codes.Code(evt.Code).String(), evt.Err)
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

func getColorMarker(code int32) (marker string) {
	if code == 0 {
		marker = color.GreenString(" > ")
	} else {
		marker = color.RedString(" > ")
	}
	return
}
