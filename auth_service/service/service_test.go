package service

import (
	"auth/auth_service/app"
	"auth/auth_service/config"
	"auth/proto"
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"reflect"
	"sync"
	"testing"
	"time"
)

const (
	addr       string = "127.0.0.1:8085"
	accessKey  string = "access_test_key"
	refreshKey string = "refresh_test_key"
)

func initTestConfig() *config.Config {
	return &config.Config{
		AccessKey:  accessKey,
		RefreshKey: refreshKey,
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
	conf := initTestConfig()
	err := StartService(ctx, addr, conf)
	if err != nil {
		t.Fatalf("cant start server initial: %v", err)
	}
	wait(1)
	finish() // stop server and release port
	wait(1)

	ctx, finish = context.WithCancel(context.Background())
	err = StartService(ctx, addr, conf)
	if err != nil {
		t.Fatalf("cant start server again: %v", err)
	}
	wait(1)
	finish()
	wait(1)
}

func TestServerStartError(t *testing.T) {
	ctx := context.Background()
	conf := initTestConfig()
	err := StartService(ctx, "bad_addr:8081", conf)
	if err == nil {
		t.Fatalf("expected error got %v", err)
	}
}

//check if we can connect more than 1 logger and receive equal logs
func TestMultipleLogging(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	conf := initTestConfig()
	err := StartService(ctx, addr, conf)
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

	logStream1, err := adm.Logging(adminCtx(), &proto.Nothing{})
	time.Sleep(1 * time.Millisecond)
	logStream2, err := adm.Logging(adminCtx(), &proto.Nothing{})

	var logData1 []*proto.Event
	var logData2 []*proto.Event

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
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 4; i++ {
			event, err := logStream1.Recv()
			if err != nil {
				t.Errorf("unexpected error: %v, awaiting event", err)
				return
			}

			logData1 = append(logData1, &proto.Event{
				Method:  event.Method,
				Login:   event.Login,
				Success: event.Success,
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 4; i++ {
			event, err := logStream2.Recv()
			if err != nil {
				t.Errorf("unexpected error: %v, awaiting event", err)
				return
			}

			logData2 = append(logData2, &proto.Event{
				Method:  event.Method,
				Login:   event.Login,
				Success: event.Success,
			})
		}
	}()

	ctx = context.Background()
	_, err = aut.Login(ctx, &proto.ReqUserData{Login: "not_exist", Password: "password"})
	if err == nil {
		t.Errorf("expected error got %v", err)
		return
	}
	time.Sleep(2 * time.Millisecond)

	tokens, err := aut.Register(ctx, &proto.ReqUserData{Login: "user1", Password: "password1"})
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
	if userData.Login != "user1" {
		t.Errorf("want login: %s, got: %s", "user1", userData.Login)
		return
	}
	time.Sleep(2 * time.Millisecond)

	_, err = aut.RefreshTokens(ctx, &proto.RefreshToken{RefreshToken: tokens.RefreshToken})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	time.Sleep(2 * time.Millisecond)

	wg.Wait()

	expectedLogData := []*proto.Event{
		{Method: "/main.Auth/Login", Login: "not_exist", Success: false},
		{Method: "/main.Auth/Register", Login: "user1", Success: true},
		{Method: "/main.Auth/Info", Success: true},
		{Method: "/main.Auth/RefreshTokens", Success: true},
	}

	if !reflect.DeepEqual(logData1, expectedLogData) {
		t.Fatalf("logs1 dont match\nhave %+v\nwant %+v", logData1, expectedLogData)
	}

	if !reflect.DeepEqual(logData2, expectedLogData) {
		t.Fatalf("logs1 dont match\nhave %+v\nwant %+v", logData2, expectedLogData)
	}
}

func TestAuthMethodsSuccess(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	conf := initTestConfig()
	err := StartService(ctx, addr, conf)
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

	login := "user"
	password := "pass"
	ctx = context.Background()
	_, err = aut.Register(ctx, &proto.ReqUserData{Login: login, Password: password})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	time.Sleep(2 * time.Millisecond)

	tokens, err := aut.Login(ctx, &proto.ReqUserData{Login: login, Password: password})
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
	if userData.Login != login {
		t.Errorf("want login: %s, got: %s", login, userData.Login)
		return
	}
	time.Sleep(2 * time.Millisecond)

	_, err = aut.RefreshTokens(ctx, &proto.RefreshToken{RefreshToken: tokens.RefreshToken})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
}

func TestRegisterAlreadyExistErr(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	conf := initTestConfig()
	err := StartService(ctx, addr, conf)
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

	login := "user"
	password := "pass"
	ctx = context.Background()
	if _, err = aut.Register(ctx, &proto.ReqUserData{Login: login, Password: password}); err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	time.Sleep(2 * time.Millisecond)

	errStr := "user with such login already exist"
	_, err = aut.Register(ctx, &proto.ReqUserData{Login: login, Password: password})
	if err == nil {
		t.Errorf("expected error got %v", err)
		return
	}
	statErr, ok := status.FromError(err)
	if !ok {
		t.Errorf("can't parse error to status, got bad err type: %T", err)
	}

	if statErr.Message() != errStr {
		t.Errorf("\nexpected error: %s,\n got: %s\n", errStr, statErr.Message())
	}
}

func TestLoginErrors(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	conf := initTestConfig()
	err := StartService(ctx, addr, conf)
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

	//error when password is invalid
	login := "user"
	pass, invalidPass := "pass", "bad_pass"
	errStr1 := "invalid password"
	ctx = context.Background()
	if _, err = aut.Register(ctx, &proto.ReqUserData{Login: login, Password: pass}); err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	time.Sleep(2 * time.Millisecond)

	_, err = aut.Login(ctx, &proto.ReqUserData{Login: login, Password: invalidPass})
	if err == nil {
		t.Errorf("expected error got %v", err)
		return
	}
	statErr, ok := status.FromError(err)
	if !ok {
		t.Errorf("can't parse error to status, got bad err type: %T", err)
	}

	if statErr.Message() != errStr1 {
		t.Errorf("\nexpected error: %s,\n got: %s\n", errStr1, statErr.Message())
	}

	//error when we don`t have user with such login
	login, pass = "unknown_user", "password"
	errStr2 := fmt.Sprintf("no user with such login: %s", login)
	_, err = aut.Login(ctx, &proto.ReqUserData{Login: login, Password: pass})
	if err == nil {
		t.Errorf("expected error got %v", err)
		return
	}
	statErr, ok = status.FromError(err)
	if !ok {
		t.Errorf("can't parse error to status, got bad err type: %T", err)
	}

	if statErr.Message() != errStr2 {
		t.Errorf("\nexpected error: %s,\n got: %s\n", errStr2, statErr.Message())
	}
}

func TestInfoErrors(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	conf := &config.Config{
		AccessKey: accessKey,
	}
	err := StartService(ctx, addr, conf)
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

	//generate valid token with unknown user id
	userID := "unknown_user_id"
	exp := time.Now().Add(time.Minute * time.Duration(5)).Unix()
	claims := app.UserClaims{
		userID,
		false,
		jwt.StandardClaims{
			ExpiresAt: exp,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tok.SignedString([]byte(accessKey))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	cases := []struct {
		token  string
		errStr string
	}{
		//invalid token
		{
			token + "___bad",
			"extracting user id from token err: couldn't handle this token",
		},
		//token with unknown user id
		{
			token,
			fmt.Sprintf("no user found with id: %v", userID),
		},
	}

	for i, c := range cases {
		ctx := context.Background()
		_, err := aut.Info(ctx, &proto.AccessToken{AccessToken: c.token})
		if err == nil {
			t.Errorf("[%v] expected error, got: %v", i, err)
			return
		}
		statErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("[%v] can't parse error to status, got bad err type: %T", i, err)
		}
		if statErr.Message() != c.errStr {
			t.Errorf("[%v] expected error: %v, got: %v", i, c.errStr, statErr.Message())
		}

	}
}

func TestRefreshTokensErrors(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	conf := &config.Config{
		RefreshKey: refreshKey,
	}
	err := StartService(ctx, addr, conf)
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

	//generate valid token with unknown user id
	userID := "unknown_user_id"
	exp := time.Now().Add(time.Minute * time.Duration(5)).Unix()
	claims := app.UserClaims{
		ID:    userID,
		Admin: false,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tok.SignedString([]byte(refreshKey))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	cases := []struct {
		token  string
		errStr string
	}{
		//invalid token
		{
			token + "___bad",
			"extracting user id from token err: couldn't handle this token",
		},
		//token with unknown user id
		{
			token,
			fmt.Sprintf("no user found with id: %v", userID),
		},
	}

	for i, c := range cases {
		ctx := context.Background()
		_, err := aut.RefreshTokens(ctx, &proto.RefreshToken{RefreshToken: c.token})
		if err == nil {
			t.Errorf("[%v] expected error, got: %v", i, err)
			return
		}
		statErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("[%v] can't parse error to status, got bad err type: %T", i, err)
		}
		if statErr.Message() != c.errStr {
			t.Errorf("[%v] expected error: %v, got: %v", i, c.errStr, statErr.Message())
		}

	}
}
