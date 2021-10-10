package original

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	Application = "forward-basic-auth"
	Version     string
)

type config struct {
	addr, realm, path string
	requestTimeout    time.Duration
	store             *UsersStore
}

func main() {
	if Version == "" {
		log.SetFlags(log.Flags() | log.Lshortfile)
	}

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s (%s):\n", Application, Version)
		flag.PrintDefaults()
	}

	cfg := config{
		requestTimeout: 5 * time.Second,
	}

	flag.StringVar(&cfg.addr, "addr", ":4013", "Server address")
	flag.StringVar(&cfg.realm, "realm", "Authenticate", "Realm")
	flag.StringVar(&cfg.path, "users-file", "/mnt/db/users.json", "Path to users file")

	flag.VisitAll(func(f *flag.Flag) {
		envName := strings.ReplaceAll(strings.ToUpper(f.Name), "-", "_")
		if envVal, ok := os.LookupEnv(envName); ok {
			if err := flag.Set(f.Name, envVal); err != nil {
				panic(err)
			}
		}

		f.Usage = fmt.Sprintf("%s [%s]", f.Usage, envName)
	})
	flag.Parse()

	cfg.store = NewUsersStore()
	if err := cfg.store.Load(cfg.path); err != nil {
		panic(err)
	}
	if err := cfg.store.Save(cfg.path); err != nil {
		panic(err)
	}

	runServer(cfg)
}

func runServer(cfg config) {
	srv := http.Server{
		Addr:    cfg.addr,
		Handler: NewServer(cfg),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), cfg.requestTimeout)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func NewServer(cfg config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Throttle(100))
	r.Use(middleware.Timeout(cfg.requestTimeout))
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
			cfg.store.Set(username, password)
		}
		if err := cfg.store.Save(cfg.path); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	r.Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			cred, ok := cfg.store.Get(username)
			if ok {
				if subtle.ConstantTimeCompare([]byte(password), []byte(cred)) == 1 {
					w.Header().Set("X-Auth-User", username)
					w.WriteHeader(http.StatusOK)

					return
				}
			}
		}

		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s", charset="UTF-8"`, cfg.realm))
		status := http.StatusUnauthorized
		http.Error(w, http.StatusText(status), status)
	})

	r.NotFound(http.NotFound)

	return r
}
