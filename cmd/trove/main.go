package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"trove/internal/api"
	"trove/internal/config"
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

	server := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           api.NewRouter(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("trove listening on %s", cfg.Server.Listen)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
