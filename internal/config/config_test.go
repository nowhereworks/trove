package config

import (
	"strings"
	"testing"
)

func TestLoadEnvUsesDefaults(t *testing.T) {
	cfg, err := LoadEnv(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("LoadEnv() error = %v", err)
	}

	if cfg.Server.Listen != ":8080" {
		t.Fatalf("Server.Listen = %q, want :8080", cfg.Server.Listen)
	}
	if cfg.Auth.Mode != "dev" {
		t.Fatalf("Auth.Mode = %q, want dev", cfg.Auth.Mode)
	}
	if !cfg.Raw.RequireAuthByDefault {
		t.Fatal("Raw.RequireAuthByDefault = false, want true")
	}
	if cfg.Storage.Limits.MaxArtifactFileBytes != DefaultMaxArtifactFileBytes {
		t.Fatalf("MaxArtifactFileBytes = %d, want %d", cfg.Storage.Limits.MaxArtifactFileBytes, DefaultMaxArtifactFileBytes)
	}
}

func TestLoadEnvOverlaysValues(t *testing.T) {
	env := map[string]string{
		"TROVE_SERVER_LISTEN":                        ":9090",
		"TROVE_PUBLIC_URL":                           "https://trove.example.test",
		"TROVE_DATABASE_URL":                         "postgres://trove:test@localhost/trove",
		"TROVE_DATABASE_MIGRATE_ON_STARTUP":          "true",
		"TROVE_AUTH_MODE":                            "oidc",
		"TROVE_AUTH_DEV_MODE_ENABLED":                "false",
		"TROVE_STORAGE_MAX_ARTIFACTS_PER_VERSION":    "42",
		"TROVE_STORAGE_MAX_ARTIFACT_FILE_BYTES":      "1234",
		"TROVE_STORAGE_MAX_UNPACKED_PACKAGE_BYTES":   "5678",
		"TROVE_RAW_REQUIRE_AUTH_BY_DEFAULT":          "false",
		"TROVE_REVIEWS_MINIMUM_APPROVALS":            "2",
		"TROVE_SECURITY_UNSAFE_INSTRUCTION_SCANNING": "false",
	}

	cfg, err := LoadEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadEnv() error = %v", err)
	}

	if cfg.Server.Listen != ":9090" {
		t.Fatalf("Server.Listen = %q, want :9090", cfg.Server.Listen)
	}
	if cfg.Server.PublicURL != "https://trove.example.test" {
		t.Fatalf("Server.PublicURL = %q", cfg.Server.PublicURL)
	}
	if cfg.Database.URL == "" || !cfg.Database.MigrateOnStartup {
		t.Fatalf("Database config was not overlaid: %+v", cfg.Database)
	}
	if cfg.Auth.Mode != "oidc" || cfg.Auth.DevModeEnabled {
		t.Fatalf("Auth config was not overlaid: %+v", cfg.Auth)
	}
	if cfg.Storage.Limits.MaxArtifactsPerVersion != 42 {
		t.Fatalf("MaxArtifactsPerVersion = %d, want 42", cfg.Storage.Limits.MaxArtifactsPerVersion)
	}
	if cfg.Storage.Limits.MaxArtifactFileBytes != 1234 {
		t.Fatalf("MaxArtifactFileBytes = %d, want 1234", cfg.Storage.Limits.MaxArtifactFileBytes)
	}
	if cfg.Storage.Limits.MaxUnpackedPackageBytes != 5678 {
		t.Fatalf("MaxUnpackedPackageBytes = %d, want 5678", cfg.Storage.Limits.MaxUnpackedPackageBytes)
	}
	if cfg.Raw.RequireAuthByDefault {
		t.Fatal("Raw.RequireAuthByDefault = true, want false")
	}
	if cfg.Reviews.MinimumApprovals != 2 {
		t.Fatalf("Reviews.MinimumApprovals = %d, want 2", cfg.Reviews.MinimumApprovals)
	}
	if cfg.Security.UnsafeInstructionScanning {
		t.Fatal("Security.UnsafeInstructionScanning = true, want false")
	}
}

func TestLoadEnvReportsParseErrors(t *testing.T) {
	env := map[string]string{
		"TROVE_DATABASE_MIGRATE_ON_STARTUP":     "not-bool",
		"TROVE_REVIEWS_MINIMUM_APPROVALS":       "not-int",
		"TROVE_STORAGE_MAX_ARTIFACT_FILE_BYTES": "not-int64",
	}

	_, err := LoadEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err == nil {
		t.Fatal("LoadEnv() error = nil, want parse errors")
	}

	message := err.Error()
	for _, want := range []string{
		"TROVE_DATABASE_MIGRATE_ON_STARTUP",
		"TROVE_REVIEWS_MINIMUM_APPROVALS",
		"TROVE_STORAGE_MAX_ARTIFACT_FILE_BYTES",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not contain %q", message, want)
		}
	}
}
