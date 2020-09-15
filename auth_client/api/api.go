package api

import (
	"auth_client/config"
	"auth_client/logger"
	"auth_client/proto"
	"context"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Api struct {
	Logger   logger.Logger
	Port     string
	Config   *config.Config
	Router   *chi.Mux
	AuthGRPC proto.AuthClient
}

func NewAPI(authConn proto.AuthClient, log logger.Logger, conf *config.Config) *Api {
	a := &Api{
		log,
		conf.HttpPort,
		conf,
		chi.NewMux(),
		authConn,
	}
	a.InitRouter()
	return a
}

func ServeAPI(api *Api) {

	s := &http.Server{
		Addr:        ":" + api.Port,
		Handler:     api.Router,
		ReadTimeout: 1 * time.Minute,
	}

	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)

		signal.Notify(sigCh, os.Interrupt)

		//signal.Notify(sigint, syscall.SIGTERM) // sigterm signal sent from kubernetes, Kubernetes sends a SIGTERM signal which is different from SIGINT (Ctrl+Client).

		<-sigCh
		log.Println("signal caught. shutting down...")
		if err := s.Shutdown(context.Background()); err != nil {
			log.Printf("server shutdown err: %v", err)
		}
		close(done)
	}()

	log.Printf("serving api at http://localhost%s", s.Addr)
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("serving err: %v", err)
		close(done)
	}
	<-done
}

func (a *Api) InitRouter() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(20 * time.Second))
	r.Use(middleware.Recoverer)

	r.MethodFunc("GET", "/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("Hello"))
	})
	r.MethodFunc("POST", "/register", a.register)
	a.Router = r
}
