package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEnvUsesDefaults(t *testing.T) {
	cfg, err := LoadEnv(func(string) (string, bool) { return "", false }, Defaults())
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
	if !cfg.Orgs.AllowCreateOrg {
		t.Fatal("Orgs.AllowCreateOrg = false, want true")
	}
	if cfg.Orgs.DefaultOrg != "" {
		t.Fatalf("Orgs.DefaultOrg = %q, want empty", cfg.Orgs.DefaultOrg)
	}
	if !cfg.Packages.CreatePackageOnPush {
		t.Fatal("Packages.CreatePackageOnPush = false, want true")
	}
	if !cfg.Packages.CreateNamespaceOnPush {
		t.Fatal("Packages.CreateNamespaceOnPush = false, want true")
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
		"TROVE_ALLOW_CREATE_ORG":                     "false",
		"TROVE_ORG":                                  "sample-org",
		"TROVE_CREATE_PACKAGE_ON_PUSH":               "false",
		"TROVE_CREATE_NAMESPACE_ON_PUSH":             "false",
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
	}, Defaults())
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
	if cfg.Orgs.AllowCreateOrg || cfg.Orgs.DefaultOrg != "sample-org" {
		t.Fatalf("Orgs config was not overlaid: %+v", cfg.Orgs)
	}
	if cfg.Packages.CreatePackageOnPush || cfg.Packages.CreateNamespaceOnPush {
		t.Fatalf("Packages config was not overlaid: %+v", cfg.Packages)
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

func TestLoadEnvRequiresDefaultOrgWhenCreateOrgDisabled(t *testing.T) {
	env := map[string]string{
		"TROVE_ALLOW_CREATE_ORG": "false",
	}

	_, err := LoadEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}, Defaults())
	if err == nil || !strings.Contains(err.Error(), "TROVE_ORG") {
		t.Fatalf("LoadEnv() error = %v, want TROVE_ORG requirement", err)
	}
}

func TestLoadEnvRejectsInvalidDefaultOrg(t *testing.T) {
	env := map[string]string{
		"TROVE_ORG": "Invalid_Org",
	}

	_, err := LoadEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}, Defaults())
	if err == nil || !strings.Contains(err.Error(), "TROVE_ORG") {
		t.Fatalf("LoadEnv() error = %v, want TROVE_ORG validation", err)
	}
}

func TestLoadEnvReportsParseErrors(t *testing.T) {
	env := map[string]string{
		"TROVE_DATABASE_MIGRATE_ON_STARTUP":     "not-bool",
		"TROVE_CREATE_PACKAGE_ON_PUSH":          "not-bool",
		"TROVE_CREATE_NAMESPACE_ON_PUSH":        "not-bool",
		"TROVE_REVIEWS_MINIMUM_APPROVALS":       "not-int",
		"TROVE_STORAGE_MAX_ARTIFACT_FILE_BYTES": "not-int64",
	}

	_, err := LoadEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}, Defaults())
	if err == nil {
		t.Fatal("LoadEnv() error = nil, want parse errors")
	}

	message := err.Error()
	for _, want := range []string{
		"TROVE_DATABASE_MIGRATE_ON_STARTUP",
		"TROVE_CREATE_PACKAGE_ON_PUSH",
		"TROVE_CREATE_NAMESPACE_ON_PUSH",
		"TROVE_REVIEWS_MINIMUM_APPROVALS",
		"TROVE_STORAGE_MAX_ARTIFACT_FILE_BYTES",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not contain %q", message, want)
		}
	}
}

func TestLoadFile(t *testing.T) {
	content := `
server:
  listen: ":7070"
  publicURL: "https://trove.fromfile.test"
database:
  url: "postgres://file:test@localhost/trove"
  migrateOnStartup: true
auth:
  mode: "oidc"
  devModeEnabled: false
  devToken: "file-token"
  cookieSecure: true
oidc:
  issuerURL: "https://issuer.test"
  clientID: "file-client-id"
  clientSecret: "file-client-secret"
  redirectURL: "https://trove.fromfile.test/callback"
orgs:
  allowCreateOrg: false
  defaultOrg: "file-org"
packages:
  createPackageOnPush: false
  createNamespaceOnPush: false
storage:
  mode: "postgres"
  limits:
    maxArtifactFileBytes: 2048
    maxUnpackedPackageBytes: 4096
    maxArtifactsPerVersion: 50
raw:
  requireAuthByDefault: false
  allowPublicNamespaces: false
  allowPublicPackages: false
reviews:
  requireApproval: false
  minimumApprovals: 3
  allowSelfApproval: true
security:
  secretScanning: false
  unsafeInstructionScanning: false
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fc, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	cfg := Defaults()
	applyFileConfig(&cfg, fc)

	if cfg.Server.Listen != ":7070" {
		t.Fatalf("Server.Listen = %q, want :7070", cfg.Server.Listen)
	}
	if cfg.Server.PublicURL != "https://trove.fromfile.test" {
		t.Fatalf("Server.PublicURL = %q", cfg.Server.PublicURL)
	}
	if cfg.Database.URL != "postgres://file:test@localhost/trove" {
		t.Fatalf("Database.URL = %q", cfg.Database.URL)
	}
	if !cfg.Database.MigrateOnStartup {
		t.Fatal("Database.MigrateOnStartup = false")
	}
	if cfg.Auth.Mode != "oidc" {
		t.Fatalf("Auth.Mode = %q, want oidc", cfg.Auth.Mode)
	}
	if cfg.Auth.DevModeEnabled {
		t.Fatal("Auth.DevModeEnabled = true")
	}
	if cfg.Auth.DevToken != "file-token" {
		t.Fatalf("Auth.DevToken = %q", cfg.Auth.DevToken)
	}
	if !cfg.Auth.CookieSecure {
		t.Fatal("Auth.CookieSecure = false")
	}
	if cfg.OIDC.IssuerURL != "https://issuer.test" {
		t.Fatalf("OIDC.IssuerURL = %q", cfg.OIDC.IssuerURL)
	}
	if cfg.Orgs.DefaultOrg != "file-org" {
		t.Fatalf("Orgs.DefaultOrg = %q", cfg.Orgs.DefaultOrg)
	}
	if cfg.Orgs.AllowCreateOrg {
		t.Fatal("Orgs.AllowCreateOrg = true")
	}
	if cfg.Packages.CreatePackageOnPush {
		t.Fatal("Packages.CreatePackageOnPush = true")
	}
	if cfg.Packages.CreateNamespaceOnPush {
		t.Fatal("Packages.CreateNamespaceOnPush = true")
	}
	if cfg.Storage.Limits.MaxArtifactFileBytes != 2048 {
		t.Fatalf("MaxArtifactFileBytes = %d, want 2048", cfg.Storage.Limits.MaxArtifactFileBytes)
	}
	if cfg.Storage.Limits.MaxUnpackedPackageBytes != 4096 {
		t.Fatalf("MaxUnpackedPackageBytes = %d, want 4096", cfg.Storage.Limits.MaxUnpackedPackageBytes)
	}
	if cfg.Storage.Limits.MaxArtifactsPerVersion != 50 {
		t.Fatalf("MaxArtifactsPerVersion = %d, want 50", cfg.Storage.Limits.MaxArtifactsPerVersion)
	}
	if cfg.Raw.RequireAuthByDefault {
		t.Fatal("Raw.RequireAuthByDefault = true")
	}
	if cfg.Raw.AllowPublicNamespaces {
		t.Fatal("Raw.AllowPublicNamespaces = true")
	}
	if cfg.Raw.AllowPublicPackages {
		t.Fatal("Raw.AllowPublicPackages = true")
	}
	if cfg.Reviews.RequireApproval {
		t.Fatal("Reviews.RequireApproval = true")
	}
	if cfg.Reviews.MinimumApprovals != 3 {
		t.Fatalf("Reviews.MinimumApprovals = %d, want 3", cfg.Reviews.MinimumApprovals)
	}
	if !cfg.Reviews.AllowSelfApproval {
		t.Fatal("Reviews.AllowSelfApproval = false")
	}
	if cfg.Security.SecretScanning {
		t.Fatal("Security.SecretScanning = true")
	}
	if cfg.Security.UnsafeInstructionScanning {
		t.Fatal("Security.UnsafeInstructionScanning = true")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("LoadFile() error = nil, want error")
	}
}

func TestLoadFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("invalid: yaml: : :"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile() error = nil, want parse error")
	}
}

func TestLoadWithFileEnvVarPrecedence(t *testing.T) {
	content := `
server:
  listen: ":7070"
database:
  url: "postgres://file:test@localhost/trove"
auth:
  mode: "oidc"
  devToken: "file-token"
orgs:
  defaultOrg: "file-org"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	env := map[string]string{
		"TROVE_SERVER_LISTEN": ":9999",
		"TROVE_AUTH_MODE":     "dev",
	}

	cfg, err := LoadEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}, func() Config {
		cfg := Defaults()
		fc, _ := LoadFile(path)
		applyFileConfig(&cfg, fc)
		return cfg
	}())
	if err != nil {
		t.Fatalf("LoadEnv() error = %v", err)
	}

	// Env var should override file value
	if cfg.Server.Listen != ":9999" {
		t.Fatalf("Server.Listen = %q, want :9999 (env should override file)", cfg.Server.Listen)
	}
	if cfg.Auth.Mode != "dev" {
		t.Fatalf("Auth.Mode = %q, want dev (env should override file)", cfg.Auth.Mode)
	}

	// File value should persist where no env var is set
	if cfg.Auth.DevToken != "file-token" {
		t.Fatalf("Auth.DevToken = %q, want file-token (no env var, should use file)", cfg.Auth.DevToken)
	}
	if cfg.Orgs.DefaultOrg != "file-org" {
		t.Fatalf("Orgs.DefaultOrg = %q, want file-org (no env var, should use file)", cfg.Orgs.DefaultOrg)
	}
}

func TestLoadWithFileOnly(t *testing.T) {
	content := `
server:
  listen: ":5555"
database:
  url: "postgres://onlyfile:test@localhost/trove"
orgs:
  defaultOrg: "only-file-org"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithFile(path)
	if err != nil {
		t.Fatalf("LoadWithFile() error = %v", err)
	}

	if cfg.Server.Listen != ":5555" {
		t.Fatalf("Server.Listen = %q, want :5555", cfg.Server.Listen)
	}
	if cfg.Database.URL != "postgres://onlyfile:test@localhost/trove" {
		t.Fatalf("Database.URL = %q", cfg.Database.URL)
	}
	if cfg.Orgs.DefaultOrg != "only-file-org" {
		t.Fatalf("Orgs.DefaultOrg = %q", cfg.Orgs.DefaultOrg)
	}
}

func TestLoadWithFileEmptyPath(t *testing.T) {
	cfg, err := LoadWithFile("")
	if err != nil {
		t.Fatalf("LoadWithFile(\"\") error = %v", err)
	}

	// Should be same as defaults
	if cfg.Server.Listen != ":8080" {
		t.Fatalf("Server.Listen = %q, want :8080", cfg.Server.Listen)
	}
	if cfg.Auth.Mode != "dev" {
		t.Fatalf("Auth.Mode = %q, want dev", cfg.Auth.Mode)
	}
}
