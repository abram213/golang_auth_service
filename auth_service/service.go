package main

import (
	"auth/proto"
	"context"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
	"sync"
)

//todo: write comments, structure code

type MainServer struct {
	*AuthManager
	*AdminManager
}

type AuthManager struct{
	registeredUsers	map[string]user
	loginUsers     	map[string]user
	config 			jwtConfig
}

type AdminManager struct {
	ctx context.Context
	mu  *sync.RWMutex

	loggingBroadcast   chan *proto.Event
	loggingListeners   []chan *proto.Event
}

func NewMainServer(ctx context.Context, config jwtConfig) *MainServer {
	var logLs []chan *proto.Event
	logB := make(chan *proto.Event)
	return &MainServer{
		&AuthManager{
			config: config,
			registeredUsers: map[string]user{},
			loginUsers: map[string]user{},
		},
		&AdminManager{
			mu:                 &sync.RWMutex{},
			ctx:                ctx,
			loggingBroadcast:   logB,
			loggingListeners:   logLs,
		},
	}
}

func startService(ctx context.Context, addr string, config jwtConfig) error {
	g, ctx := errgroup.WithContext(ctx)
	ms := NewMainServer(ctx, config)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(ms.unaryInterceptor),
		grpc.StreamInterceptor(ms.streamInterceptor),
	)

	g.Go(func() error {
		proto.RegisterAuthServer(server, ms.AuthManager)
		proto.RegisterAdminServer(server, ms.AdminManager)
		fmt.Println("starting server at " + addr)
		return server.Serve(lis)
	})

	g.Go(func() error {
		for {
			select {
			case event := <-ms.loggingBroadcast:
				ms.mu.Lock()
				for _, ch := range ms.loggingListeners {
					ch <- event
				}
				ms.mu.Unlock()
			case <-ctx.Done():
				return nil
			}
		}
	})

	go func() {
		select {
		case <-ctx.Done():
			break
		}
		if server != nil {
			server.GracefulStop()
		}
	}()
	return nil
}

func (am *AuthManager) Register(ctx context.Context, data *proto.ReqUserData) (*proto.Tokens, error) {
	if _, ok := am.userIsRegistered(data.Login); ok {
		return nil, status.Errorf(codes.AlreadyExists, "user with such login already exist")
	}

	id := "dsdfekijdiw67s6d87wadn"
	user := user{
		id:           id,
		login:        data.Login,
		passwordHash: data.Password,
		admin:        false,
	}
	if err := user.hashPassword(); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("hashing password err: %v", err))
	}

	am.registeredUsers[data.Login] = user
	am.loginUsers[user.id] = user

	tokens, err := user.refreshTokens(am.config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("refresh user tokens err: %v", err))
	}

	return tokens, nil
}

func (am *AuthManager) Login(ctx context.Context, data *proto.ReqUserData) (*proto.Tokens, error) {
	user, ok := am.userIsRegistered(data.Login)
	if !ok {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("no user with such login: %s", data.Login))
	}
	if !user.passwordIsValid(data.Password) {
		return nil, status.Errorf(codes.Unauthenticated,"invalid password")
	}

	am.loginUsers[user.id] = user

	tokens, err := user.refreshTokens(am.config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("refresh user tokens err: %v", err))
	}
	return tokens, nil
}

func (am *AuthManager) Logout(ctx context.Context, req *proto.AccessToken) (*proto.Nothing, error) {
	userID, err := userIDFromToken(req.AccessToken, am.config.accessKey)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}

	am.logoutByID(userID)

	return &proto.Nothing{}, nil
}

func (am *AuthManager) Info(ctx context.Context, req *proto.AccessToken) (*proto.RespUserData, error) {
	userID, err := userIDFromToken(req.AccessToken, am.config.accessKey)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}
	user, ok := am.userIsLogin(userID)
	if !ok {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("no user found with id: %v", userID))
	}

	return &proto.RespUserData{
		Id: user.id,
		Login: user.login,
		Admin: user.admin,
	}, nil
}

func (am *AuthManager) RefreshTokens(ctx context.Context, req *proto.RefreshToken) (*proto.Tokens, error) {
	userID, err := userIDFromToken(req.RefreshToken, am.config.refreshKey)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}
	user, ok := am.userIsLogin(userID)
	if !ok {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("no user found with id: %v", userID))
	}

	tokens, err := user.refreshTokens(am.config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("refresh user tokens err: %v", err))
	}

	return tokens, nil
}

func (am *AdminManager) Logging(in *proto.Nothing, alSrv proto.Admin_LoggingServer) error {
	ch := am.addLogListenersCh()
	for {
		select {
		case event := <-ch:
			if err := alSrv.Send(event); err != nil {
				fmt.Printf("err sending logs from chan %v to client: %v", ch, err)
			}
		case <-am.ctx.Done():
			return nil
		}
	}
}

func (am *AdminManager) addLogListenersCh() chan *proto.Event {
	am.mu.Lock()
	defer am.mu.Unlock()
	ch := make(chan *proto.Event)
	am.loggingListeners = append(am.loggingListeners, ch)
	return ch
}

func (ms *MainServer) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	reply, err := handler(ctx, req)

	success := false
	if err == nil {
		success = true
	}

	var login string
	if userData, ok := req.(*proto.ReqUserData); ok {
		login = userData.Login
	}

	ms.loggingBroadcast <- &proto.Event{
		Method:  info.FullMethod,
		Login: login,
		Success: success,
	}

	/*fmt.Printf(`---
	info=%v
	req=%#v
	err=%v
	`,info.FullMethod, req, err)*/

	return reply, err
}

func (ms *MainServer) streamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	//get "key" from client req context
	key, err := ms.keyFromCtx(ss.Context())
	if err != nil {
		return status.Errorf(codes.Unauthenticated, err.Error())
	}

	//simple auth
	validKey := "admin_key"
	if key != validKey {
		return status.Errorf(codes.Unauthenticated, fmt.Sprintf("invalid admin key"))
	}

	return handler(srv, ss)
}

func (ms *MainServer) keyFromCtx(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("no metadata in incoming context")
	}
	mdValues := md.Get("key")
	if len(mdValues) < 1 {
		return "", status.Errorf(codes.Unauthenticated, "no key in context metadata")
	}
	return mdValues[0], nil
}

func (am *AuthManager) userIsRegistered(login string) (user, bool) {
	if user, ok := am.registeredUsers[login]; ok {
		return user, true
	}
	return user{}, false
}

func (am *AuthManager) userIsLogin(id string) (user, bool) {
	if user, ok := am.loginUsers[id]; ok {
		return user, true
	}
	return user{}, false
}

func (am *AuthManager) logoutByID(id string) {
	delete(am.loginUsers, id)
}