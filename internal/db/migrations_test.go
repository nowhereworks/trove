package db

import (
	"context"
	"os"
	"testing"
	"time"

	"trove/internal/config"
)

func TestRunMigrationsIntegration(t *testing.T) {
	databaseURL := os.Getenv("TROVE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TROVE_TEST_DATABASE_URL is unset")
	}

	if err := RunMigrations(databaseURL); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := Open(ctx, config.DatabaseConfig{URL: databaseURL})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer pool.Close()

	var version string
	if err := pool.QueryRow(ctx, `
		select package_versions.version
		from package_versions
		join packages on packages.id = package_versions.package_id
		join namespaces on namespaces.id = packages.namespace_id
		join organizations on organizations.id = namespaces.org_id
		where organizations.slug = 'companyx'
		  and namespaces.slug = 'platform'
		  and packages.name = 'agent-backend'
	`).Scan(&version); err != nil {
		t.Fatalf("query seeded package: %v", err)
	}
	if version != "1.0.0" {
		t.Fatalf("seeded version = %q, want 1.0.0", version)
	}
}
