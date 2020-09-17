package app

import (
	"auth_client/api"
	agc "auth_client/auth_grpc_conn"
	"auth_client/config"
	"fmt"
	"log"
)

type App struct {
	Logger *log.Logger
	Config *config.Config
}

func New(conf *config.Config, log *log.Logger) (*App, error) {
	return &App{
		Logger: log,
		Config: conf,
	}, nil
}

func (a *App) Run() {
	addr := a.Config.AuthHost + ":" + a.Config.AuthPort
	authConn, err := agc.New(addr)
	if err != nil {
		a.Logger.Fatalf("new grpc conn err: %v", err)
	}
	fmt.Printf("success connected to %s\n", addr)

	appi := api.NewAPI(authConn, a.Logger, a.Config)
	appi.InitRouter()
	api.ServeAPI(appi)
}
