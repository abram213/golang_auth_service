package service

import (
	"auth_service/app"
	"auth_service/config"
	"auth_service/proto"
	"auth_service/storage"
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

//MainServer struct combines Auth and Admin services
type MainServer struct {
	*AuthManager
	*AdminManager
}

//AuthManager contains jwt config and slice to store users
type AuthManager struct {
	users   []app.User
	config  *config.Config
	storage storage.Storage
}

//AdminManager contains channels to store logging connections
type AdminManager struct {
	ctx context.Context
	mu  *sync.RWMutex

	loggingBroadcast chan *proto.Event
	loggingListeners []chan *proto.Event
}

//newMainServer create new MainServer entity
func newMainServer(ctx context.Context, config *config.Config, db storage.Storage) *MainServer {
	var logLs []chan *proto.Event
	logB := make(chan *proto.Event)
	return &MainServer{
		&AuthManager{
			config:  config,
			storage: db,
		},
		&AdminManager{
			mu:               &sync.RWMutex{},
			ctx:              ctx,
			loggingBroadcast: logB,
			loggingListeners: logLs,
		},
	}
}

//StartService start gRPC server
func StartService(ctx context.Context, addr string, config *config.Config) error {
	//connect to db
	db, err := storage.New(*config)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	ms := newMainServer(ctx, config, db)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening %s err: %v", addr, err)
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
	if _, err := am.storage.GetUserByLogin(data.Login); err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "user with such login already exist")
	}

	user := app.User{
		Login:        data.Login,
		PasswordHash: data.Password,
	}
	if err := user.HashPassword(); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("hashing password err: %v", err))
	}

	if err := am.storage.CreateUser(&user); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("create user err: %v", err))
	}

	tokens, err := user.RefreshTokens(am.config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("refresh user tokens err: %v", err))
	}

	return tokens, nil
}

func (am *AuthManager) Login(ctx context.Context, data *proto.ReqUserData) (*proto.Tokens, error) {
	user, err := am.storage.GetUserByLogin(data.Login)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("get user err: %v", err))
	}
	if !user.PasswordIsValid(data.Password) {
		return nil, status.Errorf(codes.Unauthenticated, "invalid password")
	}

	tokens, err := user.RefreshTokens(am.config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("refresh user tokens err: %v", err))
	}
	return tokens, nil
}

func (am *AuthManager) Info(ctx context.Context, req *proto.AccessToken) (*proto.RespUserData, error) {
	userID, err := app.UserIDFromToken(req.AccessToken, am.config.AccessKey)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("extracting user id from token err: %v", err))
	}
	user, err := am.storage.GetUserByID(userID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("get user err: %v", err))
	}

	return &proto.RespUserData{
		Id:    int64(user.ID),
		Login: user.Login,
		Admin: user.Admin,
	}, nil
}

func (am *AuthManager) RefreshTokens(ctx context.Context, req *proto.RefreshToken) (*proto.Tokens, error) {
	userID, err := app.UserIDFromToken(req.RefreshToken, am.config.RefreshKey)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("extracting user id from token err: %v", err))
	}
	user, err := am.storage.GetUserByID(userID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("get user err: %v", err))
	}

	tokens, err := user.RefreshTokens(am.config)
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
				//to logger
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

	//if err == nil request is success
	success := false
	if err == nil {
		success = true
	}

	//get login from request if it exist
	var login string
	if userData, ok := req.(*proto.ReqUserData); ok {
		login = userData.Login
	}

	ms.loggingBroadcast <- &proto.Event{
		Method:  info.FullMethod,
		Login:   login,
		Success: success,
	}

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
