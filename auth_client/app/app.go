package app

import (
	"auth/auth_client/api"
	agc "auth/auth_client/auth_grpc_conn"
	"auth/auth_client/config"
	"auth/auth_client/logger"
	"log"
)

type App struct {
	Logger logger.Logger
	Config *config.Config
}

func New(conf *config.Config, log logger.Logger) (*App, error) {
	return &App{
		Logger: log,
		Config: conf,
	}, nil
}

func (a *App) Run() {
	authConn, err := agc.New(a.Config.AuthGRPCPort)
	if err != nil {
		//to logger
		log.Fatalf("can't connect to auth grpc: %v", err)
	}
	appi := api.NewAPI(authConn, a.Logger, a.Config)
	appi.InitRouter()
	api.ServeAPI(appi)
}
