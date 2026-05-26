package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trove/internal/config"
	"trove/internal/db"
	"trove/internal/packages"
)

func NewPostgresPackageStore(t *testing.T) *packages.PostgresStore {
	t.Helper()

	databaseURL := os.Getenv("TROVE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TROVE_TEST_DATABASE_URL is unset")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres admin pool: %v", err)
	}

	schema := "trove_test_" + randomHex(t, 8)
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := adminPool.Exec(ctx, "create schema "+quotedSchema); err != nil {
		adminPool.Close()
		t.Fatalf("create test schema: %v", err)
	}

	testURL, err := withSearchPath(databaseURL, schema)
	if err != nil {
		adminPool.Close()
		t.Fatalf("build test database URL: %v", err)
	}

	if err := db.RunMigrations(testURL); err != nil {
		_, _ = adminPool.Exec(context.Background(), "drop schema "+quotedSchema+" cascade")
		adminPool.Close()
		t.Fatalf("run test migrations: %v", err)
	}

	pool, err := db.Open(ctx, config.DatabaseConfig{URL: testURL})
	if err != nil {
		_, _ = adminPool.Exec(context.Background(), "drop schema "+quotedSchema+" cascade")
		adminPool.Close()
		t.Fatalf("open postgres test pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, "drop schema "+quotedSchema+" cascade")
		adminPool.Close()
	})

	return packages.NewPostgresStore(pool)
}

func withSearchPath(databaseURL string, schema string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func randomHex(t *testing.T, bytesLen int) string {
	t.Helper()
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("read random bytes: %v", err)
	}
	return hex.EncodeToString(b)
}
