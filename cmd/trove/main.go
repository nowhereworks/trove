package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"trove/internal/api"
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
