package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	cfg    *Config
	server http.Server
}

func NewServer(cfg *Config) *Server {
	return &Server{
		cfg: cfg,
		server: http.Server{
			Addr:    cfg.Addr,
			Handler: newHandler(cfg),
		},
	}
}

func (s *Server) ListenAndServe() error {
	shutdownErr := make(chan error, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, s.cfg.ShutdownSignal)
	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.RequestTimeout)
		defer cancel()
		shutdownErr <- s.server.Shutdown(ctx)
	}()

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	if err := <-shutdownErr; err != nil {
		return err
	}

	return nil
}

func newHandler(cfg *Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Throttle(100))
	r.Use(middleware.Timeout(cfg.RequestTimeout))
	r.Use(middleware.SetHeader("Server", fmt.Sprintf("%s %s", Application, Version)))
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		fmt.Fprintln(w, http.StatusText(status))
	})

	r.Put("/users", func(w http.ResponseWriter, r *http.Request) {
		data := new(map[string]string)
		err := json.NewDecoder(r.Body).Decode(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for username, password := range *data {
			cfg.Store.Set(username, password)
		}
		if err := cfg.Store.Save(cfg.Path); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	r.Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			cred, ok := cfg.Store.Get(username)
			if ok {
				if subtle.ConstantTimeCompare([]byte(password), []byte(cred)) == 1 {
					w.Header().Set("X-Auth-User", username)
					w.WriteHeader(http.StatusOK)

					return
				}
			}
		}

		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s", charset="UTF-8"`, cfg.Realm))
		status := http.StatusUnauthorized
		http.Error(w, http.StatusText(status), status)
	})

	r.NotFound(http.NotFound)

	return r
}
