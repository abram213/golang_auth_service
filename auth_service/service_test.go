package main

import (
	"auth/proto"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"reflect"
	"sync"
	"testing"
	"time"
)

const (
	addr string = "127.0.0.1:8085"
	accessKey string = "access_test_key"
	refreshKey string = "refresh_test_key"
)

func initTestConfig() jwtConfig {
	return jwtConfig{
		accessKey:     accessKey,
		refreshKey:    refreshKey,
	}
}

func wait(amout int) {
	time.Sleep(time.Duration(amout) * 10 * time.Millisecond)
}

func getGrpcConn(t *testing.T) *grpc.ClientConn {
	grcpConn, err := grpc.Dial(
		addr,
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("cant connect to grpc: %v", err)
	}
	return grcpConn
}

func adminCtx() context.Context {
	md := metadata.Pairs(
		"key", "admin_key",
	)
	return metadata.NewOutgoingContext(context.Background(), md)
}

//checking is server success start/stop and release port
func TestServerStartStop(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	config := initTestConfig()
	err := startService(ctx, addr, config)
	if err != nil {
		t.Fatalf("cant start server initial: %v", err)
	}
	wait(1)
	finish() // stop server and release port
	wait(1)

	ctx, finish = context.WithCancel(context.Background())
	err = startService(ctx, addr, config)
	if err != nil {
		t.Fatalf("cant start server again: %v", err)
	}
	wait(1)
	finish()
	wait(1)
}

func TestLogging(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	config := initTestConfig()
	err := startService(ctx, addr, config)
	if err != nil {
		t.Fatalf("cant start server initial: %v", err)
	}
	wait(1)
	defer func() {
		finish()
		wait(1)
	}()

	conn := getGrpcConn(t)
	defer conn.Close()

	aut := proto.NewAuthClient(conn)
	adm := proto.NewAdminClient(conn)

	logStream, err := adm.Logging(adminCtx(), &proto.Nothing{})
	time.Sleep(1 * time.Millisecond)

	var logData []*proto.Event

	wait(1)

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
			fmt.Println("looks like you dont send anything to log stream in 3 sec")
			t.Errorf("looks like you dont send anything to log stream in 3 sec")
		}
	}()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			event, err := logStream.Recv()
			if err != nil {
				t.Errorf("unexpected error: %v, awaiting event", err)
				return
			}

			logData = append(logData, &proto.Event{
				Method: event.Method,
				Login: event.Login,
				Success: event.Success,
			})
		}
	}()

	ctx = context.Background()
	tokens, err := aut.Register(ctx, &proto.ReqUserData{Login: "username1", Password: "password1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	time.Sleep(2 * time.Millisecond)

	userData, err := aut.Info(ctx, &proto.AccessToken{AccessToken: tokens.AccessToken})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if userData.Login != "username1" {
		t.Errorf("want login: %s, got: %s", "username1", userData.Login)
		return
	}
	time.Sleep(2 * time.Millisecond)

	_, err = aut.Login(ctx, &proto.ReqUserData{Login: "username2", Password: "password2"})
	if err == nil {
		t.Errorf("expected error got: %v", err)
		return
	}
	time.Sleep(2 * time.Millisecond)

	wg.Wait()

	expectedLogData := []*proto.Event{
		{Method: "/main.Auth/Register", Login: "username1", Success: true},
		{Method: "/main.Auth/Info", Success: true},
		{Method: "/main.Auth/Login", Login: "username2", Success: false},
	}

	if !reflect.DeepEqual(logData, expectedLogData) {
		t.Fatalf("logs dont match\nhave %+v\nwant %+v", logData, expectedLogData)
	}
}