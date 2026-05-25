package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"trove/internal/api"
	"trove/internal/cli"
	"trove/internal/config"
	"trove/internal/db"
	"trove/internal/packages"
)

func main() {
	if err := run(); err != nil {
		log.Printf("trove exited: %v", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]

	switch commandMode(args) {
	case modeHelp:
		return cli.Run([]string{"help"})
	case modeServer:
		return runServer()
	case modeCLI:
		return cli.Run(args)
	default:
		return cli.Run(args)
	}

	return nil
}

type mode int

const (
	modeHelp mode = iota
	modeServer
	modeCLI
)

func commandMode(args []string) mode {
	if len(args) == 0 {
		return modeHelp
	}

	switch args[0] {
	case "serve", "server":
		return modeServer
	default:
		return modeCLI
	}
}

func runServer() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()

	var store packages.Store = packages.NewSeedMemoryStore()
	var readiness api.ReadinessCheck
	if cfg.Database.URL != "" {
		if cfg.Database.MigrateOnStartup {
			if err := db.RunMigrations(cfg.Database.URL); err != nil {
				return err
			}
		}

		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()

		store = packages.NewPostgresStore(pool)
		readiness = pool.Ping
	} else {
		log.Print("TROVE_DATABASE_URL is unset; using in-memory seeded store")
	}
	if cfg.Orgs.DefaultOrg != "" {
		writeStore, ok := store.(packages.WriteStore)
		if !ok {
			return errors.New("configured TROVE_ORG requires a writable store")
		}
		if _, err := writeStore.EnsureOrg(ctx, packages.CreateOrgRequest{Slug: cfg.Orgs.DefaultOrg, DisplayName: cfg.Orgs.DefaultOrg, Visibility: "private"}); err != nil {
			return err
		}
		log.Printf("ensured org %q exists", cfg.Orgs.DefaultOrg)
	}

	server := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           api.NewRouter(cfg, store, readiness),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("trove listening on %s", cfg.Server.Listen)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
