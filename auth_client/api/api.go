package api

import (
	"auth_client/config"
	"auth_client/errs"
	"auth_client/proto"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"google.golang.org/grpc/status"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Api struct {
	Logger   *log.Logger
	Port     string
	Config   *config.Config
	Router   *chi.Mux
	AuthGRPC proto.AuthClient
}

func NewAPI(authConn proto.AuthClient, log *log.Logger, conf *config.Config) *Api {
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
		fmt.Println("signal caught. shutting down...")
		if err := s.Shutdown(context.Background()); err != nil {
			api.Logger.Printf("server shutdown err: %v", err)
		}
		close(done)
	}()

	fmt.Printf("serving api at http://localhost%s\n", s.Addr)
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		api.Logger.Printf("serving err: %v", err)
		close(done)
	}
	<-done
}

func (a *Api) InitRouter() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(20 * time.Second))
	r.Use(middleware.Recoverer)

	r.Method("POST", "/register", a.handler(a.register))
	r.Method("POST", "/login", a.handler(a.login))
	r.Method("GET", "/info", a.handler(a.info))
	r.Method("POST", "/refresh_tokens", a.handler(a.refreshTokens))
	a.Router = r
}

func (a *Api) handler(f func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := f(w, r); err != nil {
			if st, ok := status.FromError(err); ok {
				aerr := errs.ApiError{
					Code:    errs.HTTPStatusFromCode(st.Code()),
					Message: st.Message(),
				}
				data, err := json.Marshal(aerr)
				if err != nil {
					a.Logger.Printf("internal server err: %v\n", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(aerr.Code)
				_, err = w.Write(data)
			} else if aerr, ok := err.(*errs.ApiError); ok {
				data, err := json.Marshal(aerr)
				if err != nil {
					a.Logger.Printf("internal server err: %v\n", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(aerr.Code)
				_, err = w.Write(data)
			} else {
				a.Logger.Printf("internal server err: %v\n", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}
	})
}
