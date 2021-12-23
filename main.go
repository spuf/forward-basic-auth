package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var (
	Application = "forward-basic-auth"
	Version     string
)

type Config struct {
	Addr, Realm, Path string
	RequestTimeout    time.Duration
	ShutdownSignal    os.Signal
	Store             *UsersStore
}

func main() {
	if Version == "" {
		log.SetFlags(log.Flags() | log.Lshortfile)
	}

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s (%s):\n", Application, Version)
		flag.PrintDefaults()
	}

	cfg := &Config{
		RequestTimeout: 5 * time.Second,
		ShutdownSignal: os.Interrupt,
	}

	flag.StringVar(&cfg.Addr, "addr", ":4013", "Server address")
	flag.StringVar(&cfg.Realm, "realm", "Authenticate", "Realm")
	flag.StringVar(&cfg.Path, "users-file", "/var/forward-basic-auth/users.json", "Path to users file")

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

	cfg.Store = NewUsersStore()
	if err := cfg.Store.Load(cfg.Path); err != nil {
		log.Fatal(err)
	}
	if err := cfg.Store.Save(cfg.Path); err != nil {
		log.Fatal(err)
	}

	if err := NewServer(cfg).ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
